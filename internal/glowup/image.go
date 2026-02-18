package glowup

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
)

// ImageInput is the common input for all image-based flows.
type ImageInput struct {
	ImageBase64   string `json:"image_base64,omitempty"`
	ImageFilePath string `json:"image_file_path,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
}

// PrepareImageData normalizes a single image input (base64, file path, or URL) into a data URI string.
func PrepareImageData(input ImageInput) (string, error) {
	if input.ImageBase64 != "" {
		if strings.HasPrefix(input.ImageBase64, "data:") {
			return input.ImageBase64, nil
		}

		// Decode a small portion to detect content type
		data, err := base64.StdEncoding.DecodeString(input.ImageBase64)
		if err != nil {
			return "", fmt.Errorf("decode base64: %w", err)
		}

		contentType := http.DetectContentType(data)
		return fmt.Sprintf("data:%s;base64,%s", contentType, input.ImageBase64), nil
	}

	if input.ImageFilePath != "" {
		return fileToDataURI(input.ImageFilePath)
	}

	if input.ImageURL != "" {
		return urlToDataURI(input.ImageURL)
	}

	return "", fmt.Errorf("no image provided: set image_base64, image_file_path, or image_url")
}

// ExtractImageOutput extracts image data from a Genkit model response.
// It first looks for explicit image content parts, then falls back to text.
func ExtractImageOutput(resp *ai.ModelResponse) (string, error) {
	for _, part := range resp.Message.Content {
		if strings.HasPrefix(part.ContentType, "image/") {
			return part.Text, nil
		}
	}

	// Fallback: check if text contains a data URI
	text := resp.Text()
	if strings.Contains(text, "data:image/") {
		return text, nil
	}

	// Return whatever text we got — may be an error message
	return text, nil
}

// fileToDataURI reads a file and returns its contents as a data URI.
func fileToDataURI(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	contentType := http.DetectContentType(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data)), nil
}

// urlToDataURI fetches a URL and returns its contents as a data URI.
func urlToDataURI(url string) (string, error) {
	resp, err := http.Get(url) //nolint:gosec // URL is user-provided input
	if err != nil {
		return "", fmt.Errorf("failed to fetch image from URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {

		return "", fmt.Errorf("failed to fetch image, status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// SaveImageResult decodes base64 image data and saves it to a file.
func SaveImageResult(resultData, filename string) error {
	// Strip data URI prefix if present
	if idx := strings.Index(resultData, ","); idx != -1 {
		resultData = resultData[idx+1:]
	}

	// Clean whitespace
	resultData = strings.ReplaceAll(resultData, "\n", "")
	resultData = strings.ReplaceAll(resultData, "\r", "")

	decoded, err := base64.StdEncoding.DecodeString(resultData)
	if err != nil {
		// Save as text for debugging
		txtFile := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".txt"
		_ = os.WriteFile(txtFile, []byte(resultData), 0644)
		return fmt.Errorf("decode base64: %w (raw saved to %s)", err, txtFile)
	}

	if err := os.WriteFile(filename, decoded, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
