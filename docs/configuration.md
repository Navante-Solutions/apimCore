# Configuration guide

ApimCore is driven by a single declarative YAML file. You manage products, APIs, subscriptions, and security without a separate database or admin UI (although the TUI and Admin API can expose this state). By default the file is named `config.yaml`; you can override it with `-f` / `-config` (e.g. `apimcore -f ./config/myconfig.yml --tui`) or the `APIM_CONFIG` environment variable.

## File structure

The config file is organized into sections: `gateway`, `server`, `products`, `subscriptions`, and optionally `security` and `devportal`.

## Gateway

Configures the API proxy (inbound traffic to your backends).

```yaml
gateway:
  listen: ":8080"
  backend_timeout_seconds: 30
```

- `listen`: Address and port the gateway binds to (e.g. `:8080` or `0.0.0.0:8080`).
- `backend_timeout_seconds`: Timeout for each request to a backend (default 30). Prevents stuck backends from holding connections; important in cloud/Kubernetes.

## Server

Configures the management server (health, metrics, Admin API, Developer Portal).

```yaml
server:
  listen: ":8081"
```

- `listen`: Address and port for the management server.

## Products and APIs

APIs are grouped into **products**. Each product has a unique `slug` and a list of APIs, each with a path prefix and backend URL.

```yaml
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
```

- `slug`: Unique identifier for the product (used in subscriptions).
- `path_prefix`: URL path prefix; requests starting with this path are routed to the given backend.
- `target_url`: Backend base URL for that API.
- `host`: Optional. When set, the request `Host` header must match (e.g. `api.example.com`). Enables routing by domain; leave empty or use `*` for path-only matching.
- `add_headers`: Optional. Map of header names to values added to every request sent to this backend (e.g. `X-Backend-Version: "v1"`, `X-Source: apimcore`). Useful for multi-tenant or backend identification.
- `strip_path_prefix`: Optional. When `true`, the path prefix is removed before forwarding. Example: request `/api/v1/users` with `path_prefix: "/api/v1"` is sent to the backend as `/users`. Default: `false` (path is forwarded as-is).

## Host and path routing

You can route by **host** (domain) and **path** together. Point DNS for your domains to the server where ApimCore runs; set the gateway to listen on port 80 (or put a reverse proxy in front). Each API entry can specify `host` and `path_prefix`. Matching order: the gateway tries host+path first, then path only. Define more specific paths before broader ones (e.g. `/landingpage` before `/`).

**Example:** `api.mydomain.com` to app A:5000, `mydomain.com/landingpage` to app C:8081, `mydomain.com/` to app B:8080. Full config: [examples/domain_routing.yaml](examples/domain_routing.yaml).

## Subscriptions and API keys

Access to products is granted via **subscriptions** and **keys**. Clients send a key in the `X-Api-Key` header.

```yaml
subscriptions:
  - developer_id: "default-dev"
    product_slug: "edu"
    tenant_id: "tenant-001"
    plan: "premium"
    keys:
      - name: "Default Key"
        value: "dev-key-123"
```

- `product_slug`: Must match a product `slug`.
- `tenant_id`: Optional. When set, the gateway adds the header `X-Tenant-Id` with this value on every request to the backend. Use it when your backends are multi-tenant and identify the tenant by this header.
- `keys[].value`: Secret sent by the client in `X-Api-Key`. Validate and protect these like passwords.

## Security

Optional section for rate limiting, IP blacklist, and geo-fencing.

- **Rate limit**: Global or per-tenant RPS and burst.
- **IP blacklist**: Block specific IPs or CIDR ranges.
- **Geo-fencing**: Allow or deny regions (requires GeoIP resolution; see [Production readiness](tui-production-readiness.md) for real GeoIP).

Example (see [examples/security.yaml](examples/security.yaml) for a full sample):

```yaml
security:
  ip_blacklist: []
  rate_limit:
    enabled: true
    rps: 10
    burst: 20
```

## Developer portal

Optional embedded developer portal for API documentation.

```yaml
devportal:
  enabled: true
  path: /devportal
```

- `path`: URL path where the portal is served (e.g. `http://localhost:8081/devportal`).

## Hot-reload

Hot-reload is **opt-in**. Start with `-hot-reload` to watch the config file and reload when it changes (about every 5 seconds). You will see a log line: `config file changed, reloading...`

Without `-hot-reload`, config is loaded once at startup. You can still reload manually by pressing **[R]** in the TUI, or restart the process.

No restart is required when hot-reload is enabled. Use it to add products, APIs, keys, or adjust security settings on the fly.

## Environment variables

| Option | Description |
|--------|-------------|
| `-f`, `-config` | Config file path (e.g. `apimcore -f ./config/myconfig.yml --tui`). |
| `-hot-reload` | Watch config file and reload on change (default: off). |
| `APIM_CONFIG` | Same as `-f` when the flag is not set (default: `config.yaml`). |
| `APIM_GATEWAY_LISTEN` | Override `gateway.listen`. |
| `APIM_SERVER_LISTEN` | Override `server.listen`. |
| `-use-db` | Persist BLOCKED/RATE_LIMIT events to SQLite at `data/apimcore.db` (creates `data/` if needed). |
| `-use-file-log PATH` | Persist to a JSONL file at PATH. Ignored if `-use-db` is set. |
| `APIM_FILE_LOG` | Path to JSONL file when `-use-file-log` is not set. |

## Security event log

By default there is **no persistence**: events stay in memory only and are lost when the process exits.

- **Default**: No persistence (no flag = no file, no DB).
- **`-use-db`**: SQLite at `data/apimcore.db`. The `data/` directory is created if needed.
- **`-use-file-log=<path>`** (or **`APIM_FILE_LOG`**): JSONL file at the given path.

When both `-use-db` and `-use-file-log` are set, `-use-db` wins.

## Example configs

Ready-to-use examples are in [docs/examples](examples/):

- [basic.yaml](examples/basic.yaml): One product, one API, minimal security.
- [security.yaml](examples/security.yaml): Stricter rate limits and IP protection.
- [multi_tenant.yaml](examples/multi_tenant.yaml): Multiple products and tenants.
- [geo_fencing.yaml](examples/geo_fencing.yaml): Regional access control.
- [domain_routing.yaml](examples/domain_routing.yaml): One gateway, multiple domains and paths (api.mydomain.com, mydomain.com, mydomain.com/landingpage).

Copy one and adapt paths, backends, and keys to your environment.
