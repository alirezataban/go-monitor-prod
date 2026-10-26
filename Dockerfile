# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/monitor-server ./cmd/server

# Runtime stage – use distroless/base (6MB) instead of jammy (1GB+)
# For playwright-go: install only essential dependencies for browser automation
FROM debian:12-slim

WORKDIR /app

# Install playwright-go dependencies (minimal set for chromium/firefox drivers)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libgconf-2-4 \
    libglib2.0-0 \
    libx11-6 \
    libxrender1 \
    libxext6 \
    libfreetype6 \
    libfontconfig1 \
    libdbus-1-3 \
    && rm -rf /var/lib/apt/lists/*

# Copy binary
COPY --from=builder /out/monitor-server /app/monitor-server

ENV LISTEN_ADDR=:8080
ENV HEADLESS=true
ENV DEFAULT_TIMEOUT=30s
ENV MAX_CONCURRENT=4
ENV PLAYWRIGHT_BROWSERS_PATH=/app/playwright

EXPOSE 8080

# Create non-root user
RUN useradd -m -s /sbin/nologin appuser
USER appuser

# Healthcheck to verify server is responsive
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/monitor-server"]
