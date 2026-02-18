package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/petruzalek/glowup/internal/glowup"
)

func main() {
	fileFlag := flag.String("file", "", "Path to local image file")
	urlFlag := flag.String("url", "", "URL of image")
	flag.Parse()

	ctx := context.Background()

	g, err := initGlowUp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	_ = g

	if *fileFlag == "" && *urlFlag == "" {
		log.Println("GlowUp started. Listening on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
		return
	}

	runGlowUp(ctx, *fileFlag, *urlFlag)
}

func initGlowUp(ctx context.Context) (interface{}, error) {
	cfg := glowup.Config{
		ProjectID: getEnv("GOOGLE_CLOUD_PROJECT", "daniela-genai-sandbox"),
		Location:  getEnv("GOOGLE_CLOUD_LOCATION", "global"),
		PromptDir: getEnv("PROMPT_DIR", "prompts"),
	}
	return glowup.Init(ctx, cfg)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func runGlowUp(ctx context.Context, file, url string) {
	fmt.Println("Running GlowUp Flow...")
	var input glowup.Input
	if file != "" {
		input.ImageFilePath = file
	} else {
		input.ImageURL = url
	}

	output, err := glowup.GlowUpFlow.Run(ctx, input)
	if err != nil {
		log.Fatalf("Restoration failed: %v", err)
	}

	if err := glowup.SaveImageResult(output.RestoredImageBase64, "glowup_result.jpg"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Saved to glowup_result.jpg")
}
