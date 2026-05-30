# ─── Stage 1: build the React frontend ───────────────────
# Debian-based node (glibc), NOT alpine: Tailwind v4 uses a native Rust engine
# (@tailwindcss/oxide) whose prebuilt binary is libc-specific; glibc avoids the
# musl resolution issues that break `npm ci` on alpine.
FROM node:22-slim AS frontend
WORKDIR /app/apps/frontend

# Install deps first (better layer caching).
COPY apps/frontend/package.json apps/frontend/package-lock.json ./
RUN npm ci

# Build into the Go embed directory (vite outDir -> ../../internal/webui/dist).
COPY apps/frontend/ ./
RUN npm run build
# Output lands at /app/internal/webui/dist

# ─── Stage 2: build the Go binary (embeds frontend) ───────
FROM golang:1.25 AS backend
WORKDIR /app

# Cache modules.
COPY go.mod go.sum ./
RUN go mod download

# Copy source, then overlay the freshly built frontend so //go:embed picks it up.
COPY . .
COPY --from=frontend /app/internal/webui/dist ./internal/webui/dist

# Pure-Go build (pgx + whatsmeow need no cgo) -> static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /out/wa-proxy ./apps/backend

# ─── Stage 3: minimal runtime image ──────────────────────
FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 waproxy && \
    mkdir -p /app/storage/images /app/storage/docs /app/storage/media /app/logs && \
    chown -R waproxy:waproxy /app

COPY --from=backend /out/wa-proxy /app/wa-proxy

USER waproxy
EXPOSE 8080

ENV APP_ENV=production \
    APP_PORT=8080 \
    STORAGE_PATH=/app/storage \
    LOG_PATH=/app/logs

# Lightweight healthcheck against the unauthenticated /healthz endpoint.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/app/wa-proxy"]
