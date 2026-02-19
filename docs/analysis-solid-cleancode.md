# Analysis: SOLID, Clean Code, Performance, Maintainability

This document summarizes a full pass over the codebase for structure, SOLID, clean code, performance, and maintainability. Recommendations are ordered by impact and effort.

**Applied (refactor):** Config RPP->RPS and default constants; gateway `resolveRoute`, `trafficEventFromRequest`, rate-limiter cap and constants; main split into `parseFlags`, `loadConfig`, `setupPersistence`, `setupMuxes`, `runGateway`, `runManagementServer`, `runTUI` with constants; TUI and hub constants; admin split into `admin_products.go`, `admin_definitions.go`, `admin_subscriptions.go`, `admin_keys.go`, `admin_usage.go`.

---

## 1. Current structure (files and sizes)

| Path | Lines (approx) | Role |
|------|----------------|------|
| `cmd/apimcore/main.go` | ~309 | Flags, config, wiring, servers, TUI bootstrap, hot-reload, metrics ticker |
| `config/config.go` | ~115 | Config types, Load, Default, env overrides |
| `internal/gateway/gateway.go` | ~395 | Gateway struct, routing, proxy, rate-limit middleware, rebuildHandler |
| `internal/gateway/security.go` | ~105 | IP blacklist, GeoIP (mock) middlewares |
| `internal/gateway/middleware.go` | ~15 | Middleware type and Chain |
| `internal/tui/tui.go` | ~1043 | Model, Update (big switch), View, all view helpers, Init |
| `internal/tui/components.go` | ~217 | renderGlobalMap, renderHeader, renderFooter, renderSparkline |
| `internal/hub/hub.go` | ~124 | Broadcaster, Collector (mock stats) |
| `internal/store/store.go` | ~444 | Products, definitions, subscriptions, keys, usage, PopulateFromConfig |
| `internal/meter/meter.go` | ~99 | Prometheus metrics, Record, StatsSince, AvgLatencySince |
| `internal/securitylog/log.go` | ~227 | Logger interface, file + SQLite backends (async) |
| `internal/admin/admin.go` | ~254 | HTTP handlers for products, definitions, subscriptions, keys, usage |
| `internal/devportal/devportal.go` | ~123 | Dev portal HTTP handlers |

Observations:

- **main.go** concentrates flags, config loading, store/gateway/hub/secLog wiring, hot-reload loop, both servers (gateway + management), TUI vs non-TUI branches, and the metrics goroutine. That is several responsibilities in one place.
- **internal/tui/tui.go** is the largest file (~1043 lines): one `Update()` switch handles all message types and key commands; view logic is split across many methods but all in one package.
- **internal/gateway** is already split (gateway, security, middleware). Good.
- **internal/securitylog** uses an interface and two implementations; structure is good.
- **config**, **hub**, **meter**, **devportal** are small and focused.

---

## 2. SOLID

### Single Responsibility (SRP)

- **main.go**: Violates SRP. It handles CLI, config bootstrap, dependency wiring, server setup, TUI bootstrap, and hot-reload. Recommendation: extract “app bootstrap” (config + wiring + server setup) into a separate package or at least into clearly named functions (e.g. `runServer()`, `runWithTUI()`) so `main()` only parses flags and delegates.
- **Gateway**: Does routing, auth (key lookup), proxy, security (blacklist, geo, rate limit), and publishing to the hub. Acceptable for a gateway “facade,” but routing + auth could be separated (e.g. “Router” that returns backend + subscription) for clarity and testing.
- **TUI Model**: One type handles all views, all messages, and all key bindings. Splitting by “view” or by “message handler” would better respect SRP (see below).

### Open/Closed (OCP)

- **securitylog**: Open for extension (new backends) via `Logger` interface; closed for modification. Good.
- **Gateway**: Adding a new middleware requires changing `rebuildHandler()`. The middleware list is already data-driven from config; extending with new middleware types would mean touching that function. Acceptable; could be improved later with a registry of middleware builders.

### Liskov Substitution (LSP)

- Not heavily used. `Logger` implementations (file vs SQLite) are used interchangeably; behavior is consistent.

### Interface Segregation (ISP)

- **Logger** is a small interface (`Append`, `Close`). Good.
- **Store** is a concrete struct; admin and gateway depend on `*store.Store`. No interface. For unit tests, passing a small interface (e.g. “key lookup + subscription”) would allow mocks without needing the full store. Optional improvement.

### Dependency Inversion (DIP)

- **main** wires concrete types (store, meter, hub, gateway, secLog). No interfaces there.
- **Gateway** depends on `*store.Store`, `*meter.Meter`, `*hub.Broadcaster`. If these were interfaces (e.g. `KeyResolver`, `MetricsRecorder`, `TrafficPublisher`), gateway would depend on abstractions and tests would be easier. Not mandatory at current scale but would improve testability and DIP.

**Summary SOLID**: The main gaps are SRP in `main` and in the TUI `Model`, and the lack of interfaces for Store/Meter/Hub when used by the gateway. The rest is acceptable or already in good shape (e.g. securitylog).

---

## 3. Clean code

### Naming

- **RPP** in `config.RateLimitConfig`: the YAML tag is `requests_per_second`, but the Go field is `RPP`. Consider renaming to `RPS` (or `RequestsPerSecond`) to match docs and YAML semantics and avoid confusion.

### Magic numbers

- Traffic list cap: `100` (tui).
- Log lines cap: `1000` (tui).
- Hot-reload interval: `5 * time.Second` (main).
- Metrics tick: `2 * time.Second` (main).
- Hub traffic channel buffer: `100`.
- Security log async buffer: `2000`.

Recommendation: define named constants (e.g. in each package or in a shared `internal/constants` if you want to tune them in one place). Improves readability and future tuning.

### Duplication

- **TrafficEvent** construction is repeated in gateway (proxyHandler, rate limit, blacklist, geo). The same struct is built in several places with small variations. A small helper (e.g. `newTrafficEvent(r, action, status, ...)`) would reduce duplication and mistakes.
- **Route resolution** in `proxyHandler`: “host+path then path-only” is done twice (config products, then store definitions). Could be a single helper `findTarget(cfg, store, host, path, apiKey)` returning (targetApi, apiDef, sub) to simplify the handler and make routing testable.

### Long functions

- **main()**: Long; multiple phases. Splitting into `parseFlags()`, `loadConfig()`, `setupPersistence()`, `runServers()`, `runTUI()` (or similar) would improve readability.
- **Update()** in tui: Large switch with many cases. Could be split by message type (e.g. `handleKeyMsg()`, `handleWindowSize()`, `handleTrafficEvent()`) or by view (e.g. each view has a small “handle key” function). Same behavior, easier to navigate and test.
- **proxyHandler**: Long; does routing, auth, proxy, recording. Extracting “resolve route” and “publish traffic” into helpers would shorten it and clarify steps.

### Comments and documentation

- Public packages (config, gateway, hub, store, admin, etc.) would benefit from short package comments and, where non-obvious, from doc comments on exported types/functions. No need for inline comments on every line; focus on “why” and contracts.

---

## 4. Performance

### Already in good shape

- **Hub.PublishTraffic**: Non-blocking send; drops when buffer full so the gateway is not blocked. Good.
- **Security log**: Async writer with buffered channel; `Append()` is non-blocking (drops when full). Good.
- **Gateway**: Uses `RWMutex` for config/handler; read path is fast. Atomic counters for blocked/rate-limited. Good.

### Improvements to consider

- **Rate limiters (gateway)**: One limiter per IP in a map; no eviction. Under many distinct IPs the map can grow without bound. Options: periodic cleanup of old entries, or LRU/cap (e.g. max 100k IPs, evict oldest). Reduces risk of memory exhaustion under abuse.
- **Store usage**: `usage` is a slice; `UsageSince(since)` scans all entries. For very high traffic and long retention this can become slow and memory-heavy. Options: cap slice size and drop oldest; or use a ring buffer / time-windowed structure. Same for `RequestCountHistory` in TUI (already capped at 20). Acceptable as-is for moderate load; worth documenting as a scaling limit.
- **TUI traffic**: Only last 100 events kept; logs last 1000. Reasonable; no change needed unless you want configurable limits.

### Concurrency

- No shared mutable state between gateway and TUI beyond the hub channels and atomic counters. Safe.
- Hot-reload updates store and gateway config; gateway uses `UpdateConfig` under mutex. Correct.

---

## 5. Maintainability and splitting into more files

### High impact, reasonable effort

1. **cmd/apimcore/main.go**
   - Extract:
     - Flag parsing and help into a small `flags` or `cli` block (or keep in main but in a `parseFlags()` that returns a struct).
     - “Persistence” setup (useDB, useFileLog, secLog) into a function that returns `Logger`.
     - Server setup (gateway mux, management mux, handlers) into a function that returns the management mux and maybe a “run” function.
     - TUI branch (metrics goroutine + tea.Program) into a function like `runTUI(cfg, st, gw, hb, ...)`.
   - Keep `main()` as: parse flags → load config → create store/gateway/hub/secLog → start hot-reload (if needed) → run gateway goroutine → run server (or TUI). This keeps the same behavior but makes main readable and testable in parts.

2. **internal/tui**
   - **tui.go** is very long. Options:
     - **By view**: move each view’s rendering and (if desired) key handling to separate files, e.g. `view_dashboard.go`, `view_traffic.go`, `view_security.go`, `view_config.go`, etc. Same package `tui`, so they all see `Model` and shared styles.
     - **By concern**: e.g. `messages.go` (Update handlers for tea.KeyMsg, WindowSizeMsg, LogMsg, hub.SystemStats, TrafficEvent), `view.go` (View() and body switch), `model.go` (NewModel, Init, updateTrafficTable). This keeps one “view” file per screen and one place for message handling.
   - Prefer splitting by view first (dashboard, traffic, security, config, health, admin, portal, global) so that each file has a clear, single responsibility (one screen). Then, if `Update()` is still too big, extract message handlers into helpers or a separate file.

3. **internal/gateway**
   - **gateway.go**: Extract routing into a function, e.g. `(g *Gateway) resolveRoute(host, path, apiKey string) (targetApi *config.ApiConfig, apiDef *store.ApiDefinition, sub *store.Subscription)`. Then `proxyHandler` becomes: resolve route → if nil then 404 → prepare request → proxy → record. Shorter and easier to test routing in isolation.
   - Optionally extract a small helper to build `hub.TrafficEvent` from `*http.Request` + action + status so all middlewares and proxyHandler use it (reduces duplication and mistakes).

### Medium impact, optional

4. **internal/admin**
   - One file with many handlers. Could split by resource: `admin_products.go`, `admin_keys.go`, `admin_usage.go`, etc. Same package, same `Handler`; just group related handlers. Improves navigation; not required for correctness.

5. **internal/store**
   - Single file with products, definitions, subscriptions, keys, usage. For Go this is common. If it grows (e.g. more entities or methods), consider splitting by domain (products_definitions.go, subscriptions_keys.go, usage.go) in the same package. Low priority until the file becomes hard to navigate.

6. **Config field name**
   - Rename `RateLimitConfig.RPP` to `RPS` (and keep YAML `requests_per_second`) for clarity. Update gateway and TUI to use the new name.

### Lower priority

7. **Interfaces for testing**
   - Introduce small interfaces where they help tests (e.g. gateway depending on a “KeyStore” or “SubscriptionResolver” instead of `*store.Store`). Optional; do when you add more gateway or store tests.

8. **Constants**
   - Replace magic numbers (buffer sizes, intervals, caps) with named constants in the respective packages.

---

## 6. Summary table

| Area | Status | Action |
|------|--------|--------|
| **Structure** | main and tui too large | Split main into bootstrap + run functions; split tui by view (and optionally by message handling). |
| **SOLID** | SRP and DIP weakest | Reduce main’s responsibilities; optionally add interfaces for store/meter/hub where tests need them. |
| **Clean code** | Duplication, long functions, RPP name | Extract route resolution and TrafficEvent helper; break long Update/proxyHandler; rename RPP → RPS; add constants. |
| **Performance** | Generally good | Consider rate-limiter map eviction/cap and usage slice cap/ring if scaling. |
| **Maintainability** | Good in gateway, securitylog, config | Keep current split; add package/docs and view-based split in tui. |

Overall the project is in good shape: clear packages, async security log, non-blocking hub, and gateway already split into gateway/security/middleware. The largest gains are: slimming down `main.go` and `internal/tui/tui.go` (more files, same package), extracting routing and event helpers in the gateway, and small cleanups (RPS, constants, doc comments). Doing the main and TUI splits first will improve SOLID (SRP) and maintainability the most.
