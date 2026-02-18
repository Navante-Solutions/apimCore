# ApimCore

API Management da solucao Navante em Go: gateway unico, medidor de requisicoes e dev portal.

## Funcoes

- **Gateway**: proxy reverso para EduCore, IntegraCore, IdentityCore e CampusCore com roteamento por path
- **Medidor de requisicoes**: contagem por API, metodo, status e latencia; metricas Prometheus e historico em memoria
- **Dev Portal**: documentacao publica dos produtos/APIs e metricas de uso (ultimas 24h)
- **Admin API**: CRUD de produtos, definicoes de API, assinaturas e chaves de API

## Requisitos

- Go 1.22+

## Configuracao

Copie e edite `config.yaml`. Principais entradas:

- `gateway.listen`: porta do gateway (ex.: `:8080`)
- `server.listen`: porta do servidor (admin, dev portal, metricas) (ex.: `:8081`)
- `backends`: lista de rotas path_prefix -> target_url para cada core

## Build e execucao

```bash
cd apimCore
go mod tidy
go build -o apim ./cmd/apim
```

Ou use o Makefile: `make build` e `make run`.

Variaveis de ambiente (producao):

- `APIM_CONFIG`: caminho do config (default: `config.yaml`)
- `APIM_GATEWAY_LISTEN`: override da porta do gateway (ex.: `:8080`)
- `APIM_SERVER_LISTEN`: override da porta do server (ex.: `:8081`)

## Endpoints

| Onde | Path | Descricao |
|------|------|------------|
| Gateway (:8080) | /educore/*, /integra/*, etc. | Proxy para os backends |
| Server (:8081) | /metrics | Prometheus (apim_requests_total, apim_request_duration_seconds, apim_usage_records_total) |
| Server | /api/admin/products | GET/POST produtos |
| Server | /api/admin/definitions | POST definicoes de API |
| Server | /api/admin/subscriptions | GET/POST assinaturas |
| Server | /api/admin/keys | POST criar chave (retorna a chave em claro uma unica vez) |
| Server | /api/admin/usage | GET uso (query: hours=24) |
| Server | /devportal | Dev Portal (HTML) |
| Server | /devportal/api/products | Lista produtos publicados |
| Server | /devportal/api/apis | Lista APIs (query: product_id opcional) |
| Server | /devportal/api/usage | Resumo de uso 24h |
| Server | /health, /ready | Health check (Kubernetes, load balancers) |

## Uso da API via gateway

Envie o header `X-Api-Key: <sua-chave>` nas requisicoes. A chave e obtida via POST /api/admin/keys (corpo: `{"subscription_id": 1, "name": "minha-chave"}`). O gateway valida a chave, repassa o `X-Tenant-Id` quando houver e registra cada requisicao no medidor.

## Producao

### CI (GitHub Actions)

- **CI** (`.github/workflows/ci.yml`): em todo push/PR em `main`/`master` roda build, testes e golangci-lint.
- **Release** (`.github/workflows/release.yml`): em push de tag `v*` gera binario, builda imagem Docker, publica em GHCR e cria release com o binario.

Para gerar uma release: `git tag v1.0.0 && git push origin v1.0.0`. A imagem fica em `ghcr.io/<owner>/<repo>:v1.0.0`.

### Docker

```bash
docker build -t apimcore:local .
docker run --rm -p 8080:8080 -p 8081:8081 -v $(pwd)/config.yaml:/etc/apim/config.yaml:ro -e APIM_CONFIG=/etc/apim/config.yaml apimcore:local
```

Ou com docker-compose (healthcheck e restart incluidos):

```bash
docker compose up -d
```

### Makefile

- `make build` - compila o binario
- `make test` - executa os testes
- `make lint` - roda golangci-lint
- `make docker` - builda a imagem Docker
- `make run` - compila e executa
- `make clean` - remove o binario

## Estrutura

- `cmd/apim`: main e embed do dev portal
- `config`: carregamento YAML
- `internal/store`: armazenamento em memoria (produtos, APIs, assinaturas, chaves, uso)
- `internal/gateway`: proxy e gravacao de uso
- `internal/meter`: metricas Prometheus e gravacao no store
- `internal/admin`: API administrativa
- `internal/devportal`: API publica do dev portal
- `web/devportal`: UI do dev portal (embed em cmd/apim/web/devportal)
