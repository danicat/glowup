package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/server"
)

// Input schema for the glowUp flow
type Input struct {
	URL         string `json:"url,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

func main() {
	// Auto-detect Cloud Run or server environment
	if len(os.Args) < 2 {
		if os.Getenv("K_SERVICE") != "" || os.Getenv("PORT") != "" {
			runServer([]string{})
			return
		}
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "server":
		runServer(os.Args[2:])
	case "restore":
		runRestore(os.Args[2:])
	case "client":
		runClient(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: glowup <command> [flags]

Commands:
  server   Start the GlowUp Genkit flow server
  restore  Run the flow locally (requires Vertex AI credentials)
  client   Send an HTTP request to a running GlowUp server`)
}

// ----------------------------------------------------------------------------
// SERVER SUBCOMMAND
// ----------------------------------------------------------------------------
func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	port := fs.String("port", "8080", "HTTP port to listen on")
	fs.Parse(args)

	// Override PORT from environment if set, otherwise use flag
	envPort := os.Getenv("PORT")
	if envPort != "" {
		*port = envPort
	}
	listenAddr := ":" + *port

	ctx := context.Background()

	// Initialize Genkit with the Vertex AI plugin
	g := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.VertexAI{}),
	)

	defineGlowUpFlow(g)

	log.Printf("GlowUp Flow Server started. Listening on %s", listenAddr)
	mux := http.NewServeMux()
	for _, flow := range genkit.ListFlows(g) {
		mux.HandleFunc("POST /"+flow.Name(), genkit.Handler(flow))
	}

	if err := server.Start(ctx, listenAddr, mux); err != nil {
		if ctx.Err() != nil {
			log.Println("GlowUp server shutting down gracefully...")
			return
		}
		log.Fatal(err)
	}
}

// ----------------------------------------------------------------------------
// RESTORE SUBCOMMAND (Local execution)
// ----------------------------------------------------------------------------
func runRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	urlFlag := fs.String("url", "", "URL or Data URI of the image to restore")
	fileFlag := fs.String("file", "", "Path to local image file to restore")
	outFlag := fs.String("out", "", "Output filename (optional)")
	fs.Parse(args)

	if *urlFlag == "" && *fileFlag == "" {
		fmt.Println("Error: must provide either --url or --file")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialize Genkit with the Vertex AI plugin
	g := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.VertexAI{}),
	)

	glowup := defineGlowUpFlow(g)

	input := Input{}
	baseName := determineBaseName(*fileFlag, *urlFlag, *outFlag)

	if *fileFlag != "" {
		uri, ctype, err := fileToDataURI(*fileFlag)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
		input.URL = uri
		input.ContentType = ctype
	} else {
		input.URL = *urlFlag
		input.ContentType = "image/jpeg" // Default if just a URL is provided
	}

	fmt.Println("Running GlowUp Flow locally...")
	out, err := glowup.Run(ctx, input)
	if err != nil {
		log.Fatalln("Flow execution failed:", err)
	}

	err = saveResult(out, baseName)
	if err != nil {
		log.Fatalln("Failed to save image:", err)
	}
}

// ----------------------------------------------------------------------------
// CLIENT SUBCOMMAND (Remote execution)
// ----------------------------------------------------------------------------
func runClient(args []string) {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	addr := fs.String("server", "http://localhost:8080", "Address of the GlowUp server")
	urlFlag := fs.String("url", "", "URL or Data URI of the image to restore")
	fileFlag := fs.String("file", "", "Path to local image file to restore")
	outFlag := fs.String("out", "", "Output filename (optional)")
	fs.Parse(args)

	if *urlFlag == "" && *fileFlag == "" {
		fmt.Println("Error: must provide either --url or --file")
		fs.Usage()
		os.Exit(1)
	}

	input := Input{}
	baseName := determineBaseName(*fileFlag, *urlFlag, *outFlag)

	if *fileFlag != "" {
		uri, ctype, err := fileToDataURI(*fileFlag)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
		input.URL = uri
		input.ContentType = ctype
	} else {
		input.URL = *urlFlag
		input.ContentType = "image/jpeg"
	}

	fmt.Printf("Sending request to %s/glowUp...\n", *addr)

	// Prepare Genkit Handler request body
	reqBody := map[string]interface{}{
		"data": input,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatalf("Failed to encode request: %v", err)
	}

	resp, err := http.Post(*addr+"/glowUp", "application/json", bytes.NewReader(b))
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Fatalf("Server returned error (status %d): %s", resp.StatusCode, string(b))
	}

	var respBody struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	err = saveResult(respBody.Result, baseName)
	if err != nil {
		log.Fatalf("Failed to save image: %v", err)
	}
}

// ----------------------------------------------------------------------------
// HELPERS
// ----------------------------------------------------------------------------

func defineGlowUpFlow(g *genkit.Genkit) *core.Flow[Input, string, struct{}] {
	return genkit.DefineFlow(g, "glowUp", func(ctx context.Context, input Input) (string, error) {
		prompt := genkit.LookupPrompt(g, "glowup")
		if prompt == nil {
			return "", errors.New("prompt 'glowup' not found")
		}

		resp, err := prompt.Execute(ctx, ai.WithInput(input))
		if err != nil {
			return "", fmt.Errorf("generation failed: %w", err)
		}

		return resp.Media(), nil
	})
}

func fileToDataURI(path string) (uri, contentType string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	contentType = http.DetectContentType(data)
	encoded := base64.StdEncoding.EncodeToString(data)
	uri = fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
	return uri, contentType, nil
}

func determineBaseName(fileFlag, urlFlag, outFlag string) string {
	if outFlag != "" {
		return outFlag
	}

	if fileFlag != "" {
		ext := filepath.Ext(fileFlag)
		return strings.TrimSuffix(filepath.Base(fileFlag), ext) + "_restored"
	}

	if urlFlag != "" {
		if u, err := url.Parse(urlFlag); err == nil && u.Path != "" {
			base := filepath.Base(u.Path)
			if base != "/" && base != "." && base != "" {
				ext := filepath.Ext(base)
				return strings.TrimSuffix(base, ext) + "_restored"
			}
		}
	}
	return "restored"
}

func getUniqueFilename(filename string) string {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return filename
	}

	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	counter := 1

	for {
		newFilename := fmt.Sprintf("%s_%d%s", base, counter, ext)
		if _, err := os.Stat(newFilename); os.IsNotExist(err) {
			return newFilename
		}
		counter++
	}
}

func saveResult(resultData, requestedFilename string) error {
	// resultData should be a data URI: data:<mimetype>;base64,<data>
	if !strings.HasPrefix(resultData, "data:") {
		return errors.New("unsupported encoding format in response")
	}

	parts := strings.Split(resultData, ";base64,")
	if len(parts) != 2 {
		return errors.New("invalid data URI format")
	}

	// Try to guess the correct extension from the data URI
	mimeType := strings.TrimPrefix(parts[0], "data:")
	ext := ".png" // fallback
	if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
		ext = exts[0]
		// standardize jpeg extension
		if ext == ".jpe" || ext == ".jpeg" {
			ext = ".jpg"
		}
	}

	// Add extension if requestedFilename doesn't already have one
	if filepath.Ext(requestedFilename) == "" {
		requestedFilename += ext
	}

	finalName := getUniqueFilename(requestedFilename)

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}

	if err := os.WriteFile(finalName, decoded, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Printf("Successfully saved restored image to %s\n", finalName)
	return nil
}
