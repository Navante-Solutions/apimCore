# Getting started with ApimCore

This guide walks you through installing and running ApimCore. For configuration details, see [Configuration](configuration.md). For an overview of the system, see [Architecture](architecture.md).

## Prerequisites

- [Go](https://go.dev/) 1.24 or higher (for building from source)
- [Docker](https://www.docker.com/) (optional, for containerized deployment)

## Installation

### Option 1: Pre-built binary

1. Open [Releases](https://github.com/Navante-Solutions/apimCore/releases).
2. Download the archive for your OS and architecture (e.g. `apimCore_linux_amd64.tar.gz`).
3. Extract and run:
   ```bash
   tar -xzf apimCore_linux_amd64.tar.gz
   ./apimcore -f config.yaml --tui
   ```

### Option 2: Build from source

1. Clone the repository:
   ```bash
   git clone https://github.com/Navante-Solutions/apimCore.git
   cd apimCore
   ```

2. Download dependencies and build:
   ```bash
   go mod tidy
   go build -o apimcore ./cmd/apim
   ```

3. Run with your config and optional TUI:
   ```bash
   ./apimcore -f config.yaml --tui
   ```

### Option 3: Docker

1. Build the image (or use a published image):
   ```bash
   docker build -t apimcore:latest .
   ```

2. Run with config mounted:
   ```bash
   docker run -p 8080:8080 -p 8081:8081 -v "$(pwd)/config.yaml:/etc/apimcore/config.yaml" apimcore:latest
   ```

   The default config path inside the container is `/etc/apimcore/config.yaml`. Override with:
   ```bash
   docker run -e APIM_CONFIG=/path/in/container/config.yaml ...
   ```

## Running the application

- **Without TUI** (gateway and management server only):
  ```bash
  ./apimcore -f config.yaml
  ```

- **With TUI** (adds the in-process management interface):
  ```bash
  ./apimcore -f config.yaml --tui
  ```

If `-f` / `-config` is omitted, ApimCore uses `APIM_CONFIG` if set, otherwise `config.yaml` in the current directory.

## Quick check

By default the gateway listens on `:8080` and the management server on `:8081`.

Health check:

```bash
curl http://localhost:8081/health
```

Expected response: `OK`.

Prometheus metrics:

```bash
curl http://localhost:8081/metrics
```

## Next steps

- Copy or adapt an [example configuration](examples/) and point `--config` at it.
- Read the [Configuration guide](configuration.md) to define products, APIs, subscriptions, and security.
- Use the TUI (with `--tui`) to explore the dashboard, traffic, and admin views (F3 for the menu).
