# WA Proxy Platform

Self-hosted WhatsApp Gateway built on WhatsApp Web (whatsmeow). A single Go
binary serves the REST API, WebSocket realtime feed, media files, and the
embedded React dashboard. Deployment is just `docker compose up -d` with two
services: `wa-proxy` and `postgres`.

## Features

- WhatsApp login via QR code with session persistence and auto-reconnect
- Send text, image, and document messages
- Queue-based broadcast with random delay and automatic retry
- Inbox monitoring, chat history, and reply from the dashboard
- Contact management with search
- REST API for external integrations (HTTP Basic Auth)
- Realtime updates over WebSocket (QR, device status, new messages)
- All data in PostgreSQL; media on local filesystem (swappable for S3/MinIO/R2)

## Architecture

```
React (Vite) ──build──> embed.FS ──┐
                                    ▼
            ┌──────────────  Go single binary  ──────────────┐
            │  Gin REST API   WebSocket Hub   Static SPA       │
            │  whatsmeow client   Background workers           │
            └───────────────────────┬──────────────────────────┘
                                     ▼
                                PostgreSQL
```

The frontend builds into `internal/webui/dist` and is embedded into the binary
via `//go:embed`, so production runs a single process.

## Tech Stack

- Backend: Go 1.24+, Gin, GORM, whatsmeow, gorilla/websocket
- Frontend: React 18, Vite, TailwindCSS, FlyonUI, React Router, TanStack Query, Zustand
- Database: PostgreSQL
- Deploy: Docker + Docker Compose

## Project Layout

```
apps/
  backend/        main.go (entrypoint)
  frontend/       React app (Vite)
internal/
  api/            Gin router, handlers, middleware
  config/         env config loader
  database/       connection + auto-migrate + ensure-db
  model/          GORM models
  repository/     data access
  storage/        local media storage
  webui/          embedded frontend (dist) + SPA serving
  whatsapp/       whatsmeow manager, senders, event handlers
  worker/         message sender, broadcast, session monitor
  ws/             WebSocket hub + client
deploy/docker/    Dockerfile
docker-compose.yml
Makefile
```

## Configuration

Copy `.env.example` to `.env` and adjust:

| Variable | Default | Description |
| --- | --- | --- |
| `APP_PORT` | `8080` | HTTP port |
| `BASIC_AUTH_USERNAME` | `admin` | Dashboard/API user |
| `BASIC_AUTH_PASSWORD` | `secret123` | Dashboard/API password |
| `DB_HOST` / `DB_PORT` | `localhost` / `5432` | PostgreSQL host/port |
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `waproxy` | PostgreSQL creds/db |
| `STORAGE_PATH` | `./storage` | Media directory |
| `BROADCAST_MIN_DELAY_MS` | `3000` | Min delay between broadcast sends |
| `BROADCAST_MAX_DELAY_MS` | `8000` | Max delay between broadcast sends |
| `BROADCAST_MAX_RETRY` | `2` | Retries per failed broadcast recipient |

The app creates the target database automatically if it does not exist.

## Development

Backend (terminal 1):

```bash
go run ./apps/backend
```

Frontend dev server with hot reload (terminal 2):

```bash
cd apps/frontend
npm install
npm run dev
```

The Vite dev server (http://localhost:5173) proxies `/api`, `/storage`, and
`/ws` to the Go backend on :8080.

## Build (single binary)

```bash
# 1. Build the frontend into the embed directory
cd apps/frontend && npm run build && cd ../..

# 2. Build the Go binary (embeds the frontend)
go build -o bin/wa-proxy ./apps/backend

# 3. Run
./bin/wa-proxy
```

Or with make: `make build && make run`.

Then open http://localhost:8080 and log in with the Basic Auth credentials.

## Production (Docker)

```bash
docker compose up -d
```

This builds the multi-stage image (frontend + backend) and starts `wa-proxy`
and `postgres`. Media and database use named volumes for persistence.

## API Quick Reference

All endpoints require HTTP Basic Auth.

| Method | Path | Description |
| --- | --- | --- |
| POST | `/api/device/connect` | Start QR login / reconnect |
| GET | `/api/device/qr` | Latest QR code |
| GET | `/api/device/status` | Device status |
| POST | `/api/device/logout` | Logout WhatsApp |
| POST | `/api/messages/send` | Send text `{phone, message}` |
| POST | `/api/messages/send-image` | Send image (multipart: phone, caption, file) |
| POST | `/api/messages/send-file` | Send document (multipart: phone, caption, file) |
| GET | `/api/conversations` | Inbox list |
| GET | `/api/conversations/:id` | Chat history |
| POST | `/api/conversations/:id/reply` | Reply `{message}` |
| GET | `/api/contacts?search=` | Contacts |
| GET | `/api/contacts/:id` | Contact detail |
| POST | `/api/broadcasts` | Create broadcast `{name, message, phones[]}` |
| GET | `/api/broadcasts` | Broadcast history |
| GET | `/api/broadcasts/:id` | Broadcast detail |
| GET | `/api/dashboard/stats` | Dashboard counters |
| GET | `/ws` | WebSocket realtime feed |

### WebSocket events

```json
{ "type": "qr",             "data": { "code": "..." } }
{ "type": "device.status",  "data": { "status": "connected" } }
{ "type": "message.new",    "data": { "conversation_id": 1, "message": {...} } }
{ "type": "broadcast.update","data": { ...broadcast } }
```

## Notes

- Single-device scope (Phase 1 per the PRD). Multi-device/session, webhooks,
  API keys, and rate limiting are on the future roadmap.
- Media files are stored under `STORAGE_PATH` and served (auth-protected) under
  `/storage`. The DB stores only relative paths, so the backend can later be
  pointed at object storage without schema changes.
