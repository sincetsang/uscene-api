# ============================================
# uscene-api-main Dockerfile
# Multi-stage build for Go API + Cron
# ============================================

# ---- Build Stage ----
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
COPY pkg/ pkg/
RUN go mod download

COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/uscene ./cmd/uscene

# Build Cron binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/cron ./cmd/cron

# ---- Runtime Stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl
ENV TZ=Asia/Shanghai

# Copy binaries
COPY --from=builder /out/uscene /app/uscene
COPY --from=builder /out/cron  /app/cron

# Copy static assets
COPY static/ /app/static/

WORKDIR /app

EXPOSE 8050

# Default: run API server
# Config file is mounted via K8s ConfigMap at /app/config/config.toml
CMD ["./uscene", "-c", "/app/config/config.toml"]
