# Production readiness

This document lists current mocks and placeholders in the TUI and gateway pipeline, and what is required to run APIM Core with real data in production. It is intended for operators and contributors who need to harden deployments or implement the missing pieces.

---

## 1. Traffic and latency metrics

**Current state:** `AvgLatency` is sent as `0` in the TUI ticker and in `hub.SystemStats`.

**To implement:**

- In the meter, expose a function that computes average latency from usage (e.g. `AvgLatencySince(since time.Time) float64`). The store already records `ResponseTimeMs` in `RequestUsage`; aggregate in a time window (e.g. last hour).
- In the metrics ticker in `main.go`, call that function and set `AvgLatency` (in ms) in `MetricsUpdateMsg` and `hub.SystemStats`.

**Relevant:** `internal/meter/meter.go`, `internal/store/store.go`, `cmd/apim/main.go`.

---

## 2. CPU and memory (system vitals)

**Current state:** Dashboard uses fallback values (e.g. 0.42 / 0.65) when `CPUUsage` / `RAMUsage` are zero. The hub Collector currently sends mock data.

**To implement:**

- Use a system library (e.g. `github.com/shirou/gopsutil/v3`) in the hub Collector to read real CPU percent and memory (e.g. `mem.VirtualMemory()`).
- Start the Collector in `main.go` when the TUI is active and inject the hub; call `PublishStats` with real values on each tick.
- In the TUI, keep the fallback only until the first real `SystemStats` is received.

**Relevant:** `internal/hub/hub.go`, `cmd/apim/main.go`, `internal/tui/tui.go`.

---

## 3. Rate limit and blocked counters

**Current state:** `RateLimited` and `Blocked` in `SystemStats` are not updated; the gateway does not emit events when returning 429 or 403.

**To implement:**

- In the gateway, when responding with 429 (rate limit) or 403 (blacklist/geo), call `Hub.PublishTraffic` with a `TrafficEvent` with the appropriate status and action (e.g. "RATE_LIMIT", "BLOCKED"). Maintain in-memory counters (e.g. atomic or sync) and include them in `SystemStats` (either from the gateway or an aggregator).
- Alternatively, aggregate from traffic events in the TUI or a central component over a time window and feed into `SystemStats`.

**Relevant:** `internal/gateway/gateway.go`, `internal/gateway/security.go`, `internal/hub/hub.go`, `cmd/apim/main.go`.

---

## 4. Node and cluster (security card)

**Current state:** TUI shows fixed values such as "NODE: US-EAST-1A" and "NODES: 12 ACTIVE".

**To implement:**

- **NODE:** Read from environment (e.g. `APIM_NODE_ID` or `HOSTNAME`); if unset, use OS hostname or "local".
- **NODES:** For single-node, show "1" or "N/A". For multi-node, use a config or cluster API (e.g. `APIM_CLUSTER_NODES`); until then, show "1" or "single node".
- Pass these values into the TUI model from `main.go` (config or env).

**Relevant:** `config/config.go`, `cmd/apim/main.go`, `internal/tui/tui.go`.

---

## 5. Sparkline (performance trend)

**Current state:** Sparkline uses fixed data `[10, 20, 15, 30, 25]`.

**To implement:**

- Keep a rolling buffer in the TUI model (e.g. request count per minute, last N points). Each tick, append the count for the last interval and trim to cap.
- Either the meter/store exposes "requests per interval for last N intervals", or the main ticker sends "requests in last interval" and the TUI builds the series.

**Relevant:** `internal/tui/tui.go`, `cmd/apim/main.go`.

---

## 6. Config view

**Current state:** Config view shows a fixed path and "Editable Console - Coming Soon".

**To implement:**

- Pass the actual config path into the TUI model (e.g. `ConfigPath`) and display "Loaded from: " + path (or "default: config.yaml" if empty).
- For production: either keep view-only with text like "View-only. Edit file and press [R] to reload.", or design a safe in-app edit flow.

**Relevant:** `internal/tui/tui.go`, `cmd/apim/main.go`.

---

## 7. Developer portal view

**Current state:** Placeholder values for "Public APIs", "Documentation %", "Status".

**To implement:**

- **Public APIs:** Use `len(store.ListDefinitions())` or count only published product APIs.
- **Documentation:** If definitions have a spec URL or similar, compute the share with docs; otherwise show "N/A".
- **Status:** Derive from process/health (e.g. "LIVE" when server is up).

**Relevant:** `internal/tui/tui.go`, store API.

---

## 8. Admin view – tenants

**Current state:** Tenants list is hardcoded (e.g. "Walmart, Target, Acme").

**To implement:**

- Get unique tenant IDs from the store (e.g. from subscriptions and optional usage). Add a helper such as `store.UniqueTenantIDs() []string` if needed.
- Display up to N tenants with ", ..." or a summary like "X tenants".

**Relevant:** `internal/store/store.go`, `internal/tui/tui.go`.

---

## 9. GeoIP (gateway and traffic)

**Current state:** GeoIP middleware uses mock resolution (e.g. "Local", "US" for 8.8.8.8, or derived from IP parity).

**To implement:**

- Integrate a real provider: MaxMind GeoLite2 (`.mmdb` + e.g. `github.com/oschwald/geoip2-golang`) or an HTTP geo service.
- Add config (e.g. `GeoIP.DBPath` or `GeoIP.Provider`). In the middleware, resolve country from IP, set headers, and apply geo-fencing rules. TUI traffic view will then show real country data.

**Relevant:** `config/config.go`, `internal/gateway/security.go`, deployment docs (e.g. GeoLite2 download).

---

## 10. Uptime and system stats (hub Collector)

**Current state:** Collector sends mock uptime, CPU, memory, active connections, and geo threats.

**To implement:**

- **Uptime:** Record process start time (e.g. in main or hub) and send `time.Since(processStartTime)` in the Collector.
- **CPU/Memory:** See section 2 (gopsutil or equivalent).
- **ActiveConns:** Optional: use server `ConnState` or gateway connection counters; or omit until implemented.
- **RateLimited / Blocked:** See section 3.
- **GeoThreats:** Aggregate from real traffic (e.g. blocked/error by country) or remove from SystemStats until an aggregator exists.

**Relevant:** `internal/hub/hub.go`, `cmd/apim/main.go`.

---

## 11. Suggested dependencies

| Area           | Dependency                          | Use case                |
|----------------|-------------------------------------|-------------------------|
| CPU / memory   | `github.com/shirou/gopsutil/v3`     | Collector system stats  |
| GeoIP          | `github.com/oschwald/geoip2-golang` + MaxMind DB | Real country per IP |
| Config path    | Existing `APIM_CONFIG`              | Pass through to TUI     |
| Average latency| None (store + meter)                | `AvgLatencySince`       |
| Sparkline      | None (buffer in model)              | Requests per interval   |
| Node / cluster | Env (`APIM_NODE_ID`, etc.)          | Security card           |

---

## 12. Suggested implementation order

1. Real metrics: latency and request counts (meter/store + main: AvgLatency, sparkline buffer).
2. Config path and copy: Model.ConfigPath, config view and portal view from store.
3. Admin tenants: real tenant list from store.
4. Portal: API count and documentation % from store.
5. CPU/RAM: gopsutil in Collector, started from main.
6. Rate limit / blocked: gateway emits 429/403 events and counters; feed into SystemStats.
7. Node/cluster: env or config and TUI.
8. GeoIP: config + middleware with MaxMind (or other provider).
9. Uptime, ActiveConns, GeoThreats: finalize Collector and aggregators.

With these in place, the TUI and gateway can run in production with real data and no mocks.
