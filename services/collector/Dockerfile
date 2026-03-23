# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go get \
    go.opentelemetry.io/otel@v1.28.0 \
    go.opentelemetry.io/otel/sdk@v1.28.0 \
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.28.0 \
    google.golang.org/grpc@v1.65.0 \
    github.com/segmentio/kafka-go@latest
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o collector .

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /app/collector .
USER app

EXPOSE 8081
ENTRYPOINT ["/app/collector"]
