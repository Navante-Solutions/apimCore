# ApimCore documentation

This folder contains the official documentation for ApimCore.

## Guides

| Document | Description |
|----------|-------------|
| [Getting started](getting-started.md) | Installation (binary, source, Docker), first run, and health check |
| [Configuration](configuration.md) | Config file structure, gateway, server, products, subscriptions, security, and hot-reload |
| [Architecture](architecture.md) | Components, store, gateway, meter, and management server |
| [Deployment](deployment.md) | AWS, Azure, Kubernetes; ingress, egress, health probes, internal vs external |

## Example configurations

| File | Description |
|------|-------------|
| [basic.yaml](examples/basic.yaml) | Single product, one API, minimal security |
| [security.yaml](examples/security.yaml) | Stricter rate limits and IP protection |
| [multi_tenant.yaml](examples/multi_tenant.yaml) | Multi-tenant setup with several products |
| [geo_fencing.yaml](examples/geo_fencing.yaml) | Regional access control (e.g. EU vs global) |
| [domain_routing.yaml](examples/domain_routing.yaml) | Domain and path-based routing for multiple backends |
| [headers_and_rewrite.yaml](examples/headers_and_rewrite.yaml) | Header injection and path prefix stripping with tenant support |

Copy or adapt these files to your `config.yaml` and adjust paths, backends, and keys as needed.
