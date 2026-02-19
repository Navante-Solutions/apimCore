FROM golang:1.24-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /apimcore ./cmd/apimcore

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata curl
COPY --from=builder /app/config.yaml /etc/apimcore/config.yaml
COPY --from=builder /apimcore /apimcore
ENV APIM_CONFIG=/etc/apimcore/config.yaml
USER nobody
EXPOSE 8080 8081
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=2 \
	CMD curl -f http://localhost:8081/health || exit 1
ENTRYPOINT ["/apimcore"]
CMD []
