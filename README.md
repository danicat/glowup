# GlowUp (Go)

GlowUp is a Genkit-powered image enhancement service written in Go. It demonstrates how to wrap AI image processing flows into both a command-line interface and a web server.

## ✨ Features

*   **Image Enhancement:** Uses Genkit flows to process and enhance images.
*   **Dual Modes:** Run it as a CLI tool for one-off tasks or as a long-running HTTP server.

## 🚀 Usage

### 1. CLI Mode

Enhance a local image file directly from the terminal:

```bash
go run cmd/main.go --file path/to/your/image.jpg
```

Or provide an image URL:

```bash
go run cmd/main.go --url https://example.com/image.jpg
```

The result will be saved to `glowup_result.jpg`.

### 2. Server Mode

Start the HTTP server to handle requests:

```bash
go run cmd/main.go
```

The server listens on port `8080`.

## ⚙️ Configuration

The application uses the following environment variables:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `GOOGLE_CLOUD_PROJECT` | `<YOUR_PROJECT_ID>` | Google Cloud Project ID |
| `GOOGLE_CLOUD_LOCATION` | `global` | Vertex AI location |
| `PROMPT_DIR` | `prompts` | Directory containing `.prompt` files |

## 🧪 Development

To run the Genkit Developer UI and inspect the flows:

```bash
genkit start
```
