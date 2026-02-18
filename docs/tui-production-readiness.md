# Plano de implementação: TUI pronta para produção

Este documento lista todos os mocks e dados fictícios na TUI e no pipeline de métricas, e o que implementar para ambiente real.

---

## 1. Métricas de tráfico e latência

### 1.1 AvgLatency zerado (main.go + hub)

**Onde:** `cmd/apim/main.go` envia `AvgLatency: 0` no ticker para a TUI e para `hub.SystemStats`.

**O que fazer:**
- Expor no **meter** uma função que calcule a latência média a partir do uso (ex.: `AvgLatencySince(since time.Time) float64`).
- O store já guarda `ResponseTimeMs` em `RequestUsage`; em `UsageSince` podemos somar e dividir pelo total.
- No ticker de métricas em `main.go`, chamar `m.AvgLatencySince(time.Now().Add(-1*time.Hour))` e preencher `AvgLatency` (em ms) em `MetricsUpdateMsg` e `hub.SystemStats`.

**Arquivos:** `internal/meter/meter.go` (nova função ou uso de store), `internal/store/store.go` (opcional helper), `cmd/apim/main.go`.

---

## 2. CPU e memória (System Vitals)

### 2.1 CPU/RAM fixos no dashboard (tui.go)

**Onde:** `internal/tui/tui.go` usa `cpuUsage = 0.42` e `ramUsage = 0.65` quando `m.CPUUsage`/`m.RAMUsage` são 0.

**O que fazer:**
- Esses valores vêm de `hub.SystemStats` (CPUUsage, MemoryUsageMB). O hub hoje só recebe dados do **Collector**, que está em mock.
- Fonte real: usar **gopsutil** (ou leitura de `/proc` no Linux) no **hub.Collector** para preencher:
  - `CPUUsage`: percentual (0–1) via `cpu.Percent()` ou equivalente.
  - `MemoryUsageMB`: uso em MB via `mem.VirtualMemory()`.
- Iniciar o **Collector** em `main.go` quando TUI estiver ativa e injetar o Hub; chamar `PublishStats` com dados reais no tick do Collector.
- Na TUI, manter o fallback 0.42/0.65 apenas quando nenhum `SystemStats` tiver sido recebido ainda (ex.: antes do primeiro tick).

**Arquivos:** `internal/hub/hub.go` (Collector com gopsutil), `cmd/apim/main.go` (iniciar Collector com Hub), `internal/tui/tui.go` (manter fallback apenas inicial).

---

## 3. Rate limit e bloqueios (Security / Traffic)

### 3.1 RateLimited e Blocked sempre zerados

**Onde:** `hub.SystemStats` tem `RateLimited` e `Blocked`; a TUI mostra esses campos. O gateway não envia eventos quando retorna 429/403.

**O que fazer:**
- **Opção A (recomendada):** No gateway, ao responder 429 (rate limit) ou 403 (blacklist/geo), chamar `Hub.PublishTraffic` com um `TrafficEvent` com `Status` 429 ou 403 e `Action` "RATE_LIMIT" ou "BLOCKED". Manter contadores em memória no gateway (por ex. `atomic` ou `sync` por tipo) e incluir no `SystemStats` (ex.: novo campo no gateway ou no hub que agregue por janela de tempo).
- **Opção B:** Manter apenas eventos de tráfego e agregar na TUI ou num agregador: contar eventos com `Action == "RATE_LIMIT"` e `Action == "BLOCKED"` na janela (ex.: última hora) e enviar em `SystemStats`.
- Para **SystemStats** no main: ou o gateway expõe contadores (ex.: interface `Stats() (rateLimited, blocked int64)`) e o ticker envia para o hub, ou um componente central lê do store/hub e envia `RateLimited`/`Blocked` no `SystemStats`.

**Arquivos:** `internal/gateway/gateway.go` (emitir evento 429/403 + opcionalmente contadores), `internal/gateway/security.go` (passar Hub ou callback para publicar evento), `internal/hub/hub.go` / `cmd/apim/main.go` (preencher SystemStats).

---

## 4. Nó e cluster (Security card)

### 4.1 "US-EAST-1A" e "12 ACTIVE" fixos

**Onde:** `internal/tui/tui.go` (security card): `NODE: US-EAST-1A`, `NODES: 12 ACTIVE`.

**O que fazer:**
- **NODE:** Ler de variável de ambiente (ex.: `APIM_NODE_ID` ou `HOSTNAME`); se vazio, usar `hostname` do OS ou "local".
- **NODES:** Em cenário single-node, mostrar "1" ou "N/A". Para multi-node, exigir uma fonte (ex.: config, API de cluster ou env `APIM_CLUSTER_NODES`); até lá, mostrar "1" ou "single node".
- TUI: receber esses valores por parâmetro no Model (ex.: `NodeID`, `ClusterNodeCount`) preenchidos em `main.go` a partir de config/env.

**Arquivos:** `config/config.go` (opcional: `NodeID`, `ClusterNodes`), `cmd/apim/main.go` (ler env/config e passar para TUI), `internal/tui/tui.go` (usar campos do Model).

---

## 5. Sparkline (Performance trend)

### 5.2 Dados fixos [10, 20, 15, 30, 25]

**Onde:** `internal/tui/tui.go`: `m.renderSparkline([]int64{10, 20, 15, 30, 25}, sparkWidth)`.

**O que fazer:**
- Manter um buffer no Model (ex.: `RequestCountsLastN []int64` ou por minuto) atualizado pelo ticker: a cada intervalo (ex.: 1 min), anotar total de requests no período e adicionar ao buffer (cap ex.: 20 pontos).
- No main, ao enviar métricas para a TUI, enviar também um valor “requests no último intervalo” (ou a TUI usa `TotalRequests` e deriva diferença entre ticks).
- Alternativa: meter/store expõe “total por minuto nos últimos N minutos”; TUI chama ou recebe via msg e usa para o sparkline.

**Arquivos:** `internal/tui/tui.go` (buffer + nova msg ou uso de métricas existentes), `cmd/apim/main.go` (enviar contagem por intervalo se necessário).

---

## 6. Config view

### 6.1 "Loaded from: config.yaml" e "(Editable Console - Coming Soon)"

**Onde:** `internal/tui/configView()`: path fixo e texto "Coming Soon".

**O que fazer:**
- **Path:** Receber o path real de config no Model (ex.: `ConfigPath string` preenchido em main a partir de `configPath`). Exibir: "Loaded from: " + m.ConfigPath (ou "default: config.yaml" se vazio).
- **Edição:** Para produção, definir se haverá editor in-app (arriscado) ou apenas "View-only; edit config file and reload with [R]". Por agora, trocar para "View-only. Edit file and press [R] to reload."

**Arquivos:** `internal/tui/tui.go` (Model.ConfigPath, configView), `cmd/apim/main.go` (passar configPath ao criar Model).

---

## 7. Developer Portal view

### 7.1 "Public APIs: 2", "Documentation: 85%", "Status: LIVE"

**Onde:** `internal/tui/portalView()`.

**O que fazer:**
- **Public APIs:** Usar `len(m.Store.ListDefinitions())` ou contar apenas APIs de produtos publicados (já existe store).
- **Documentation:** Se houver campo ou regra (ex.: definições com `OpenAPISpecURL != ""`), calcular percentual (ex.: X de Y com spec). Caso contrário, exibir "N/A" ou remover até existir dado real.
- **Status:** Manter "LIVE" se o servidor estiver no ar; opcionalmente derivar de health check interno.

**Arquivos:** `internal/tui/tui.go` (portalView usando Store).

---

## 8. Admin view – tenants

### 8.1 "TENANTS: Walmart, Target, Acme" fixo

**Onde:** `internal/tui/adminView()`: `subContent += "TENANTS: Walmart, Target, Acme"`.

**O que fazer:**
- Obter lista real de tenants: por ex. `store.ListSubscriptions()` e extrair `TenantID` únicos, ou adicionar `store.UniqueTenantIDs() []string` que percorre subscriptions (e opcionalmente usage) e retorna IDs únicos.
- Exibir até N tenants (ex.: 5) e ", ..." se houver mais; ou "X tenants" como resumo.

**Arquivos:** `internal/store/store.go` (opcional: UniqueTenants), `internal/tui/tui.go` (adminView usa store).

---

## 9. GeoIP (gateway + tráfego)

### 9.1 Mock em GeoIPMiddleware

**Onde:** `internal/gateway/security.go`: país "Local", "US" (8.8.8.8), ou "BR"/"DE" por paridade do IP.

**O que fazer:**
- Integrar um provedor real: MaxMind GeoLite2 (arquivo .mmdb + lib tipo `github.com/oschwald/geoip2-golang`), ou serviço HTTP de geolocalização.
- Config: ex.: `config.GeoIP.DBPath` ou `GeoIP.Provider` (maxmind/http). No middleware, resolver país pelo IP e setar header + aplicar geo-fencing com o código real.
- Tráfego na TUI já usa `Country` do evento; passará a refletir o país real.

**Arquivos:** `config/config.go` (GeoIP), `internal/gateway/security.go` (GeoIPMiddleware com DB/serviço), documentação de deploy (download de GeoLite2, etc.).

---

## 10. Uptime e SystemStats (hub Collector)

### 10.1 Collector com dados mock

**Onde:** `internal/hub/hub.go`: Collector envia `Uptime: time.Hour`, `CPUUsage: 0.45`, `MemoryUsageMB: 1240`, `ActiveConns: 42`, `GeoThreats` fixo.

**O que fazer:**
- Ver itens 2 (CPU/RAM com gopsutil) e 3 (RateLimited/Blocked).
- **Uptime:** Calcular desde o start do processo (ex.: variável `processStartTime time.Time` em main ou no hub e enviar `time.Since(processStartTime)` no Collector).
- **ActiveConns:** Opcional: usar `net/http` Server ConnState ou contador de conexões no gateway; ou omitir até haver implementação.
- **GeoThreats:** Agregar do tráfego real (países com 4xx/5xx ou bloqueios); ou remover do SystemStats até existir agregador.

**Arquivos:** `internal/hub/hub.go`, `cmd/apim/main.go`.

---

## 11. Resumo de dependências externas sugeridas

| Item              | Dependência sugerida        | Uso                    |
|-------------------|-----------------------------|------------------------|
| CPU / Memória     | `github.com/shirou/gopsutil/v3` | Collector system stats |
| GeoIP             | `github.com/oschwald/geoip2-golang` + MaxMind DB | País real por IP   |
| Config path       | Já existe `APIM_CONFIG`     | Só passar para TUI     |
| Latência média    | Nenhuma (store + meter)      | AvgLatencySince        |
| Sparkline         | Nenhuma (buffer no Model)   | Requests por intervalo |
| Nó / cluster      | Env (APIM_NODE_ID, etc.)    | Security card          |

---

## 12. Ordem sugerida de implementação

1. **Métricas reais (latência, requests):** meter/store + main (AvgLatency, sparkline buffer).
2. **Config path e textos:** Model.ConfigPath, configView e portalView com dados do Store.
3. **Admin tenants:** lista real de tenants a partir do store.
4. **Portal (APIs, docs %):** contar definições e specs do store.
5. **CPU/RAM:** gopsutil + Collector + iniciar Collector no main.
6. **Rate limit / bloqueios:** gateway emite eventos 429/403 e contadores; SystemStats preenchido.
7. **Nó/cluster:** env/config + TUI.
8. **GeoIP:** config + middleware com MaxMind (ou outro).
9. **Uptime/ActiveConns/GeoThreats:** ajustes finos no Collector e agregadores.

Com isso, a TUI deixa de depender de mocks e fica pronta para uso em produção com dados reais.
