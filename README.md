# GlowUp (Go)

GlowUp is a professional-grade photo restoration tool powered by [Firebase Genkit](https://github.com/firebase/genkit) and the **Gemini 3 Pro Image (Nano Banana Pro)** model. It can transform vintage, damaged, or monochrome photographs into high-fidelity colorized versions.

This project demonstrates a unified Go binary architecture that acts as a Server, a Local CLI, and a Remote Client.

## 📦 Build

To build the unified binary:

```bash
go build -o glowup main.go
```

## 🚀 Usage

The `glowup` binary supports three primary subcommands.

### 1. `server`
Starts the Genkit HTTP server to host the restoration flow as an API endpoint.

```bash
# Set credentials and start
export GOOGLE_CLOUD_PROJECT=your-project-id
export GOOGLE_CLOUD_LOCATION=global
./glowup server --port 8080
```

### 2. `restore`
Runs the restoration flow locally on your machine. This requires Vertex AI credentials and talks directly to the AI model.

```bash
# Restore a local file
./glowup restore --file ./assets/vintage.jpg

# Restore from a URL
./glowup restore --url "https://example.com/photo.jpg"

# Specify a custom output name
./glowup restore --file ./assets/vintage.jpg --out my_restoration.png
```

### 3. `client`
A lightweight subcommand that sends an HTTP request to a running GlowUp server. This does **not** require AI credentials on the client side.

```bash
# Point to a local or remote server
./glowup client --server http://localhost:8080 --file ./assets/vintage.jpg

# Point to a Cloud Run deployment
./glowup client --server https://glowup-xyz.a.run.app --url "https://example.com/photo.jpg"
```

## ✨ Smart Features

*   **Intelligent Naming:** Automatically generates output filenames based on input (e.g., `photo.jpg` -> `photo_restored.png`).
*   **Collision Prevention:** If an output file already exists, it appends a counter (e.g., `photo_restored_1.png`) to avoid overwriting your work.
*   **MIME Detection:** Automatically detects the output image type from the AI response and applies the correct file extension.
*   **Multi-modal Input:** Automatically converts local files into Base64 Data URIs to satisfy the model's multimodal requirements.

## ☁️ Deployment (Cloud Run)

You can deploy the GlowUp server to Google Cloud Run in minutes using the "Deploy from Source" feature:

```bash
gcloud run deploy glowup \
  --source . \
  --region us-central1 \
  --set-env-vars GOOGLE_CLOUD_LOCATION=global,GOOGLE_CLOUD_PROJECT=your-project-id \
  --allow-unauthenticated
```

## ⚙️ Configuration

| Variable | Description |
| :--- | :--- |
| `GOOGLE_CLOUD_PROJECT` | Google Cloud Project ID (Required for `server` and `restore`) |
| `GOOGLE_CLOUD_LOCATION` | Vertex AI location (Must be `global` for Nano Banana Pro) |
| `PORT` | The port the server should listen on (Defaults to `8080`) |

---

## 🧪 Development

To experiment with the flows and prompts interactively using the Genkit Developer UI:

```bash
genkit start -- go run main.go
```
