package glowup

import (
	"context"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

// Input is the input for the GlowUp flow.
type Input struct {
	ImageBase64   string `json:"image_base64,omitempty"`
	ImageFilePath string `json:"image_file_path,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
}

// Output is the output from the GlowUp flow.
type Output struct {
	RestoredImageBase64 string `json:"restored_image_base64"`
}

// Constants for model configuration
const (
	ModelName = "vertexai/gemini-3-pro-image-preview"
)

// Flow export to be usable by CLI
var GlowUpFlow *core.Flow[Input, *Output, struct{}]

// Register registers the GlowUp flow with the given Genkit instance.
func Register(g *genkit.Genkit) {
	// Define the GlowUp Flow
	GlowUpFlow = genkit.DefineFlow(g, "glowUp", func(ctx context.Context, input Input) (*Output, error) {
		imgInput := ImageInput(input)

		// 1. Prepare Image Data
		imgData, err := PrepareImageData(imgInput)
		if err != nil {
			return nil, err
		}

		// 2. Load Prompt
		prompt := genkit.LookupPrompt(g, "glowup")
		if prompt == nil {
			return nil, fmt.Errorf("prompt 'glowup' not found")
		}

		// 3. Generate
		// Extract content type from Data URI
		contentType := "image/jpeg"
		if strings.HasPrefix(imgData, "data:") {
			if parts := strings.Split(imgData, ";"); len(parts) > 0 {
				contentType = strings.TrimPrefix(parts[0], "data:")
			}
		}

		resp, err := prompt.Execute(ctx, ai.WithInput(map[string]any{
			"photo":       imgData,
			"contentType": contentType,
		}))

		if err != nil {
			return nil, fmt.Errorf("generation failed: %w", err)
		}

		// 4. Extract Output
		outputData, err := ExtractImageOutput(resp)
		if err != nil {
			return nil, err
		}

		return &Output{
			RestoredImageBase64: outputData,
		}, nil
	})
}
