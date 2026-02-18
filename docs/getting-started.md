# Getting Started with APIM Core

APIM Core is a lightweight, high-performance API Gateway designed for the Navante solution. It provides a single entry point for multiple backends, request metering, and a developer portal.

## Prerequisites

- [Go](https://go.dev/) 1.22 or higher.
- [Docker](https://www.docker.com/) (optional, for containerized deployment).

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/navantesolutions/apimCore.git
   cd apimCore
   ```

2. Download dependencies:
   ```bash
   go mod tidy
   ```

3. Build the application:
   ```bash
   go build -o apim ./cmd/apim
   ```

## Running the Application

### Local Mode

Start the APIM using the default `config.yaml`:
```bash
./apim
```

### Docker Mode

Build and run using the provided Dockerfile:
```bash
docker build -t apimcore:latest .
docker run -p 8080:8080 -p 8081:8081 apimcore:latest
```

## Quick Test

By default, the gateway listens on `:8080` and the management server on `:8081`.

Send a health check request:
```bash
curl http://localhost:8081/health
```
Response: `OK`

Next, head over to the [Configuration Guide](configuration.md) to set up your APIs.
