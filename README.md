# APIM Core

APIM Core is a lightweight, high-performance API gateway written in Go. It provides a single entry point for your APIs with key-based access control, rate limiting, and an optional terminal UI (TUI) for monitoring and administration. Configuration is YAML-driven with hot-reload; no database is required.

---

## Features

- **API gateway**: Path-based routing, API key validation, and configurable backends per product
- **Security**: IP blacklisting, CIDR support, and rate limiting (RPS/burst)
- **Geo-fencing**: Regional access control based on GeoIP (extensible for production providers)
- **Multi-tenancy**: Tenant-aware routing and metrics
- **Management TUI**: Real-time dashboard, traffic view, admin and security panels
- **Developer portal**: Optional embedded portal for API documentation
- **Observability**: Prometheus metrics and health endpoints
- **Hot-reload**: Edit `config.yaml` and see changes applied without restart

---

## Requirements

- **Go 1.24+** (for building from source)
- **Docker** (optional, for containerized runs)

---

## Quick start

### Using the binary (recommended)

Download the latest release for your platform from [Releases](https://github.com/Navante-Solutions/apimCore/releases), then:

```bash
./apim --config config.yaml --tui
```

- Gateway: `http://localhost:8080`
- Management and health: `http://localhost:8081`

### From source

```bash
git clone https://github.com/Navante-Solutions/apimCore.git
cd apimCore
go build -o apim ./cmd/apim
./apim --config config.yaml --tui
```

### Using Docker

```bash
docker run -p 8080:8080 -p 8081:8081 -v $(pwd)/config.yaml:/etc/apim/config.yaml ghcr.io/navante-solutions/apimcore:latest
```

Set `APIM_CONFIG` if your config path differs. See [Getting started](docs/getting-started.md) for details.

---

## Configuration

APIM Core is configured via a single YAML file (default: `config.yaml`). Main sections:

| Section        | Purpose                                  |
|----------------|------------------------------------------|
| `gateway`      | Listen address for the API proxy         |
| `server`       | Listen address for admin, metrics, portal |
| `products`     | API products and backend routes          |
| `subscriptions`| API keys and product access              |
| `security`     | Rate limits, IP blacklist, geo-fencing   |
| `devportal`    | Developer portal path and toggle         |

Full reference: [Configuration guide](docs/configuration.md). Example configs: [docs/examples](docs/examples/).

---

## Management TUI

With `--tui`, APIM Core starts an in-process terminal UI. Use **F3** for the main menu.

| View           | Shortcut | Description                          |
|----------------|----------|--------------------------------------|
| Dashboard      | F3       | Overview, uptime, event log          |
| Traffic        | F4       | Request list with status and GeoIP   |
| Administration | F5       | Products, APIs, subscriptions        |
| Security       | F6       | Blacklist and geo-fencing            |
| System health  | F7       | Health and metrics                   |

---

## Ports and endpoints

| Port  | Purpose                    | Endpoints (examples)        |
|-------|----------------------------|-----------------------------|
| 8080  | API gateway (proxy)        | Your API paths              |
| 8081  | Management and metrics     | `/health`, `/metrics`, Admin API, Dev Portal |

Example:

```bash
curl http://localhost:8081/health
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [Getting started](docs/getting-started.md) | Install, run, and first checks |
| [Configuration](docs/configuration.md)   | Config file reference and behavior |
| [Architecture](docs/architecture.md)     | Components and data flow |
| [Production readiness](docs/tui-production-readiness.md) | TUI and gateway production checklist |

Example configurations: [docs/examples](docs/examples/) (basic, security, multi-tenant, geo-fencing).

---

## Contributing

Contributions are welcome. Please open an issue or pull request on [GitHub](https://github.com/Navante-Solutions/apimCore).

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
