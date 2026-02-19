# ApimCore documentation

This folder contains the official documentation for ApimCore.

## Guides

| Document | Description |
|----------|-------------|
| [Getting started](getting-started.md) | Installation (binary, source, Docker), first run, and health check |
| [Configuration](configuration.md) | Config file structure, gateway, server, products, subscriptions, security, and hot-reload |
| [Architecture](architecture.md) | Components, store, gateway, meter, and management server |
| [Deployment](deployment.md) | AWS, Azure, Kubernetes; ingress, egress, health probes, internal vs external |

## Reference

| Document | Description |
|----------|-------------|
| [Production readiness](tui-production-readiness.md) | Checklist for production: real metrics, GeoIP, rate-limit events, and TUI data sources |

## Example configurations

| File | Description |
|------|-------------|
| [basic.yaml](examples/basic.yaml) | Single product, one API, minimal security |
| [security.yaml](examples/security.yaml) | Stricter rate limits and IP protection |
| [multi_tenant.yaml](examples/multi_tenant.yaml) | Multi-tenant setup with several products |
| [geo_fencing.yaml](examples/geo_fencing.yaml) | Regional access control (e.g. EU vs global) |

Copy or adapt these files to your `config.yaml` and adjust paths, backends, and keys as needed.
