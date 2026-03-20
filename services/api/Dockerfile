# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api .

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /app/api .
USER app

EXPOSE 8080
ENTRYPOINT ["/app/api"]
