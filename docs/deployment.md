# Deployment (AWS, Azure, Kubernetes)

This document covers network and deployment aspects for running ApimCore in the cloud or in Kubernetes, including ingress, egress, and internal vs external access.

## Listen addresses and ports

| Port | Role | Typical use |
|------|------|-------------|
| 8080 | Gateway (proxy) | Ingress from clients; expose via load balancer or Ingress. |
| 8081 | Management (health, metrics, admin, devportal) | Prefer **internal only**; do not expose to the internet. |

In Go, `listen: ":8080"` binds to **all interfaces** (`0.0.0.0:8080`), which is correct inside containers and VMs. No change is required for AWS, Azure, or K8s.

Override via config or environment:

```yaml
gateway:
  listen: ":8080"
server:
  listen: ":8081"
```

Or:

- `APIM_GATEWAY_LISTEN=:8080`
- `APIM_SERVER_LISTEN=:8081`

## Ingress (traffic into APIM)

- **Gateway (8080):** This is the API entry point. Put a load balancer (ALB, NLB, Azure LB) or Kubernetes Ingress in front. TLS should be terminated at the LB/Ingress; the container can stay HTTP.
- **Server (8081):** Expose only on internal networks (private subnet, cluster-internal Service, or Ingress with internal annotation). This avoids exposing `/metrics`, Admin API, and Dev Portal to the internet.

**Kubernetes example (conceptual):**

- One **Service** with two ports (8080, 8081).
- **Ingress** only for port 8080 (gateway) to the public.
- Access to 8081 (health, metrics) only from inside the cluster (e.g. Prometheus, internal tools) or via a second, internal Ingress/Service.

## Egress (APIM calling backends)

The gateway acts as a reverse proxy and sends requests **to your backends** (egress). Points to watch:

1. **Network path:** Backends can be:
   - **Internal:** ClusterIP Services, internal DNS (e.g. `http://backend.default.svc.cluster.local:8080`), or VPC-private IPs.
   - **External:** Public URLs; ensure outbound internet or VPC endpoints are allowed.

2. **Timeouts:** The gateway uses a configurable backend timeout (default 30s) so a stuck backend does not hold connections indefinitely. In `config.yaml` set `gateway.backend_timeout_seconds` (e.g. `60`). Hot-reload applies the new value.

3. **Proxy:** If egress goes through an HTTP proxy (corporate or cloud), set `HTTP_PROXY` / `HTTPS_PROXY` (and `NO_PROXY` if needed) in the pod or task environment. The gateway’s HTTP client respects these.

4. **TLS to backends:** HTTPS backends are supported; the default client uses the system CA bundle (and respects `SSL_CERT_FILE` / `HTTPS_PROXY` where applicable).

## Health and readiness

- **Liveness:** `GET /health` returns 200. Use for liveness probes.
- **Readiness:** `GET /ready` returns 200. Use for readiness probes (so traffic is sent only when the process is up).

Example (Kubernetes):

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /ready
    port: 8081
  initialDelaySeconds: 2
  periodSeconds: 5
```

Probe **port 8081** (management server); the gateway (8080) and server (8081) run in the same process, so one healthy process implies both are up.

## Docker

The image exposes 8080 and 8081 and includes a HEALTHCHECK so orchestrators can detect unhealthy containers (see the project Dockerfile).

## Kubernetes (minimal)

- **Deployment:** One container, two ports (8080, 8081). Mount config via ConfigMap or a volume; set `APIM_CONFIG` if the path differs.
- **Service:** Two ports (e.g. 8080 → gateway, 8081 → management). Expose 8080 via Ingress; keep 8081 cluster-internal unless you need external metrics/health.
- **Config:** Prefer ConfigMap + volume; avoid storing secrets (e.g. API key values) in ConfigMap; use a secret store or inject env for sensitive data if needed later.
- **Resource limits:** Set `requests`/`limits` for CPU/memory based on load; the process is single-binary and relatively small.

## Internal vs external summary

| Concern | Recommendation |
|--------|----------------|
| Gateway (8080) | Ingress from internet or internal clients; TLS at LB/Ingress. |
| Management (8081) | Internal only: metrics, health, admin, devportal. |
| Backends (egress) | Internal URLs when backends are in the same cluster/VPC; use timeouts and HTTP_PROXY if required. |

No code change is strictly required for “running on AWS/Azure/K8s”: bind to `:8080` and `:8081`, expose 8080 via your platform’s ingress, keep 8081 internal, and configure backends and (if needed) proxy env vars. The optional backend timeout and HEALTHCHECK improve robustness and operability.
