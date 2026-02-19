# ApimCore

ApimCore is a lightweight, high-performance API gateway written in Go. It provides a single entry point for your APIs with key-based access control, rate limiting, and an optional terminal UI (TUI) for monitoring and administration. Configuration is YAML-driven; optional hot-reload with `-hot-reload`. No database required.

---

## Features

- **API gateway**: Path-based routing, API key validation, and configurable backends per product
- **Security**: IP blacklisting, CIDR support, and rate limiting (RPS/burst)
- **Geo-fencing**: Regional access control based on GeoIP (extensible for production providers)
- **Multi-tenancy**: Tenant-aware routing and metrics
- **Management TUI**: Real-time dashboard, traffic view, admin and security panels
- **Developer portal**: Optional embedded portal for API documentation
- **Observability**: Prometheus metrics and health endpoints
- **Hot-reload** (opt-in): With `-hot-reload`, config file changes are applied automatically; otherwise use [R] in TUI or restart

---

## Requirements

- **Go 1.24+** (for building from source)
- **Docker** (optional, for containerized runs)

---

## Quick start

### Using installers (recommended)

Installers are automatically generated for each release:

- **Linux (Ubuntu/Debian)**: `.deb` packages for x86_64 and ARM64
- **Linux (RedHat/Fedora)**: `.rpm` packages for x86_64 and ARM64
- **Windows**: `.zip` archives with the binary (x86_64, x86, ARM64)
- **macOS**: `.tar.gz` archives for x86_64 and ARM64

Download the appropriate package from [Releases](https://github.com/Navante-Solutions/apimCore/releases):

**Linux (.deb):**
```bash
sudo dpkg -i apimcore_*.deb
```

**Linux (.rpm):**
```bash
sudo rpm -i apimcore_*.rpm
```

**Windows:**
Extract the `.zip` for your architecture and run `apimcore.exe`.

**macOS:**
```bash
tar -xzf apimcore_*_darwin_*.tar.gz
sudo mv apimcore /usr/local/bin/
```

### Using the binary

Download the latest release binary for your platform from [Releases](https://github.com/Navante-Solutions/apimCore/releases), then:

```bash
./apimcore -f config.yaml --tui
```

- Gateway: `http://localhost:8080`
- Management and health: `http://localhost:8081`

### From source

```bash
git clone https://github.com/Navante-Solutions/apimCore.git
cd apimCore
go build -o apimcore ./cmd/apimcore
./apimcore -f config.yaml --tui
```

### Using Docker

```bash
docker run -p 8080:8080 -p 8081:8081 -v $(pwd)/config.yaml:/etc/apimcore/config.yaml ghcr.io/navante-solutions/apimcore:latest
```

Set `APIM_CONFIG` if your config path differs. See [Getting started](docs/getting-started.md) for details.

---

## Configuration

ApimCore is configured via a single YAML file (default: `config.yaml`). Below is a complete example with explanations. Copy and adapt to your environment.

```yaml
gateway:
  listen: ":8080"
  backend_timeout_seconds: 30

server:
  listen: ":8081"

products:
  - name: "Educational Services"
    slug: "edu"
    description: "Core educational management APIs"
    apis:
      - name: "EduCore"
        path_prefix: "/educore"
        target_url: "http://localhost:8082"
      - name: "IntegraCore"
        path_prefix: "/integra"
        target_url: "http://localhost:8083"
        host: "api.example.com"

  - name: "Platform Services"
    slug: "platform"
    description: "Identity and campus infrastructure"
    apis:
      - name: "IdentityCore"
        path_prefix: "/identity"
        target_url: "http://localhost:8084"
        openapi_spec_url: "https://api.example.com/identity/openapi.json"
        version: "v1"

subscriptions:
  - developer_id: "default-dev"
    product_slug: "edu"
    plan: "premium"
    keys:
      - name: "Default Key"
        value: "dev-key-123"
  - developer_id: "another-dev"
    product_slug: "platform"
    plan: "basic"
    keys:
      - name: "Platform Key"
        value: "platform-secret-456"

security:
  ip_blacklist:
    - "1.2.3.4"
    - "192.168.100.0/24"
  allowed_countries:
    - "BR"
    - "US"
    - "Local"
  rate_limit:
    enabled: true
    requests_per_second: 10
    burst: 20

devportal:
  enabled: true
  path: "/devportal"
```

**Section reference**

| Section | Field | Description |
|---------|-------|-------------|
| **gateway** | `listen` | Address and port for the API proxy (e.g. `:8080` or `0.0.0.0:8080`). Incoming requests hit this first. |
| | `backend_timeout_seconds` | Max time to wait for each backend response. Default: 30. |
| **server** | `listen` | Address for admin API, metrics, health, and developer portal (e.g. `:8081`). |
| **products** | | List of API products. Each product groups one or more APIs. |
| | `name` | Human-readable product name. |
| | `slug` | Short identifier. Must be unique. Used by `subscriptions` to link keys to products. |
| | `description` | Optional description. |
| | **apis** | List of APIs in this product. |
| | `name` | API name (for display and metrics). |
| | `path_prefix` | URL path prefix. Requests starting with this path are routed to the backend (e.g. `/educore`, `/identity`). |
| | `target_url` | Backend URL (e.g. `http://localhost:8082`). |
| | `host` | Optional. Match by `Host` header. Use `*` or leave empty for path-only matching. |
| | `add_headers` | Optional. Map of headers added to every request to this backend (e.g. multi-tenant or backend identification). |
| | `strip_path_prefix` | Optional. When true, path prefix is removed before forwarding (e.g. `/api/v1/users` with prefix `/api/v1` becomes `/users`). |
| | `openapi_spec_url` | Optional. URL to OpenAPI spec for the developer portal. |
| | `version` | Optional. API version. |
| **subscriptions** | | Maps developers and keys to products. |
| | `developer_id` | Developer identifier. |
| | `product_slug` | Must match a product `slug`. Grants access to that product's APIs. |
| | `tenant_id` | Optional. When set, gateway adds `X-Tenant-Id` header to backend requests (multi-tenant backends). |
| | `plan` | Plan name (e.g. "premium", "basic"). Informational. |
| | **keys** | API keys for this subscription. Clients send the key in the `X-Api-Key` header. |
| | `name` | Key label. |
| | `value` | Secret value. Treat like a password; keep it private. |
| **security** | | Optional. Controls access and limits. |
| | `ip_blacklist` | List of IPs or CIDRs to block (e.g. `1.2.3.4`, `192.168.100.0/24`). |
| | `allowed_countries` | If non-empty, only requests from these country codes are allowed. Empty means all countries. Use `Local` for localhost. |
| | **rate_limit** | Per-IP rate limiting. |
| | `enabled` | Turn rate limiting on or off. |
| | `requests_per_second` | Max requests per second per IP. |
| | `burst` | Max burst size (additional tokens). |
| **devportal** | | Embedded developer portal. |
| | `enabled` | Enable or disable the portal. |
| | `path` | URL path where the portal is served (e.g. `/devportal`). |

Full reference: [Configuration guide](docs/configuration.md). More examples: [docs/examples](docs/examples/) (basic, security, multi-tenant, geo-fencing, **domain routing**, **headers and path rewrite** for tenant_id, add_headers, strip_path_prefix).

---

## Management TUI

With `--tui`, ApimCore starts an in-process terminal UI. Use **F3** for the main menu.

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

Metrics: Prometheus scrape at `/metrics`. Aggregated summary at `GET /api/admin/metrics/summary?hours=1` (P95/P99 latency, error rate, RPS per route, rate limit hits, usage by tenant and version, backend vs gateway latency).

---

## Documentation

| Document | Description |
|----------|-------------|
| [Getting started](docs/getting-started.md) | Install, run, and first checks |
| [Configuration](docs/configuration.md)   | Config file reference and behavior |
| [Architecture](docs/architecture.md)     | Components and data flow |
| [Deployment](docs/deployment.md)       | AWS, Azure, Kubernetes; ingress, egress, internal vs external |
| [Production readiness](docs/tui-production-readiness.md) | TUI and gateway production checklist |

Example configurations: [docs/examples](docs/examples/) (basic, security, multi-tenant, geo-fencing, domain routing, headers and path rewrite).

---

## Contributing

Contributions are welcome. Please open an issue or pull request on [GitHub](https://github.com/Navante-Solutions/apimCore).

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
