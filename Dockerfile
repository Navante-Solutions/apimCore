FROM golang:1.22-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /apim ./cmd/apim

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata curl
COPY --from=builder /app/config.yaml /etc/apim/config.yaml
COPY --from=builder /apim /apim
ENV APIM_CONFIG=/etc/apim/config.yaml
USER nobody
EXPOSE 8080 8081
ENTRYPOINT ["/apim"]
CMD []
