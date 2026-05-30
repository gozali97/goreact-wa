# WA Proxy Platform

Self-hosted WhatsApp gateway built on WhatsApp Web (whatsmeow). A **single Go
binary** serves the REST API, WebSocket realtime feed, media files, and the
embedded React dashboard. Deployment is just `docker compose up -d` with two
services: `wa-proxy` and `postgres`.

---

## Features

- **WhatsApp login** via QR code, with session persistence and auto-reconnect
- **Messaging**: text, image, document, and link-button (CTA) messages
- **Broadcast**: queue-based bulk send with random delay, retry, and
  number-validation (skips numbers not on WhatsApp)
- **Inbox & chat**: live conversation list, full history (text / image / video /
  audio / sticker / document), reply with text + media + emoji, image lightbox,
  file preview & download
- **Start chat**: begin a 1:1 conversation by phone (auto-syncs existing contacts)
- **Contacts**: list, search, automatic name resolution from WhatsApp
- **Realtime**: WebSocket push for QR, device status, new messages, broadcasts;
  in-app notification bell
- **Integration API**: REST endpoints for external apps, authenticated with a
  per-install **API key managed in the database** (generate / copy / regenerate
  from the dashboard)
- **Outbound webhooks**: HMAC-SHA256 signed, with retries (message, message.ack,
  session.status)
- **Anti-ban**: per-recipient rate limiter (default 30s), optional humanize
  (mark-seen + typing), check-number-exists
- **Observability**: activity logs (daily JSON-lines files) with a searchable
  viewer; storage/disk monitor + file manager with auto-retention cleanup
- **Interactive API docs** page with live "Try it" runner

---

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

- Backend: Go 1.24+, Gin, GORM, whatsmeow, gopsutil, gorilla/websocket
- Frontend: React 18, Vite, TailwindCSS v4, FlyonUI, @iconify (Tabler), React
  Router, TanStack Query, Zustand
- Database: PostgreSQL
- Deploy: Docker + Docker Compose

## Project Layout

```
apps/
  backend/        main.go (entrypoint)
  frontend/       React app (Vite)
internal/
  api/            Gin router, handlers, middleware
  apikey/         DB-backed API key service
  config/         env config loader
  database/       connection + auto-migrate + ensure-db
  logsvc/         daily JSON-lines log service + reader
  model/          GORM models
  ratelimit/      anti-ban send limiter
  repository/     data access
  storage/        local media storage + disk monitoring
  webhook/        outbound webhook dispatcher
  webui/          embedded frontend (dist) + SPA serving
  whatsapp/       whatsmeow manager, senders, event handlers
  worker/         message sender, broadcast, session, cleanup
  ws/             WebSocket hub + client
deploy/docker/    Dockerfile
laravel-message/  optional Laravel chat tester (not part of the image)
docker-compose.yml
Makefile
```

---

## Configuration

Copy `.env.example` to `.env` and adjust. Key variables:

| Variable | Default | Description |
| --- | --- | --- |
| `APP_ENV` | `development` | Set to `production` on servers |
| `APP_PORT` | `8080` | HTTP port |
| `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD` | `admin` / `secret123` | Dashboard login |
| `DB_HOST` / `DB_PORT` | `localhost` / `5432` | PostgreSQL host/port |
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `waproxy` / `waproxy` / `waproxy` | PostgreSQL creds/db |
| `DB_SSLMODE` | `disable` | `disable` / `require` / etc. |
| `STORAGE_PATH` | `./storage` | Media directory |
| `LOG_PATH` | `./logs` | Daily log files directory |
| `BROADCAST_MIN_DELAY_MS` / `MAX` | `3000` / `8000` | Delay between broadcast sends |
| `BROADCAST_MAX_RETRY` | `2` | Retries per failed recipient |
| `WEBHOOK_URL` | `` | Outbound webhook target (empty = off) |
| `WEBHOOK_SECRET` | `` | HMAC-SHA256 signing key (optional) |
| `WEBHOOK_MAX_RETRY` | `3` | Webhook delivery retries |
| `WEBHOOK_EVENTS` | `` | Comma list; empty = all events |
| `HUMANIZE_SEND` | `false` | Mark-seen + typing before sending |
| `RATE_LIMIT_ENABLED` | `true` | Per-recipient send rate limiter |
| `RATE_MIN_INTERVAL_SEC` | `30` | Min seconds between msgs to the same number |
| `RATE_GLOBAL_INTERVAL_SEC` | `0` | Min seconds between any sends (0 = off) |
| `MEDIA_RETENTION_DAYS` | `14` | Auto-delete media older than N days (0 = off) |

Notes:
- The app **auto-creates** the target database if it does not exist.
- The **integration API key is NOT an env var** — it lives in the database and
  is generated/copied/regenerated from the Device page in the dashboard.

---

## Local Development

Two terminals from the project root.

**Backend (live reload with Air):**
```bash
air
# or without Air:
go run ./apps/backend
```

**Frontend (Vite hot reload):**
```bash
cd apps/frontend
npm install
npm run dev
```

Open **http://localhost:5173** — the Vite dev server proxies `/api`, `/ws`, and
`/storage` to the Go backend on :8080. (At :8080 the Go binary serves the
*embedded* build; rebuild the frontend to refresh it there.)

## Build a single binary (no Docker)

```bash
cd apps/frontend && npm install && npm run build && cd ../..
go build -o bin/wa-proxy ./apps/backend
./bin/wa-proxy
```

Then open http://localhost:8080 and log in with the Basic Auth credentials.

---

## Production Deployment (Docker Compose) — recommended

This is the easiest, most reproducible path. The multi-stage `Dockerfile` builds
the frontend and backend into one image; Compose runs `wa-proxy` + `postgres`.

### Prerequisites on the server

- Linux host (Ubuntu/Debian recommended), 1 vCPU / 1 GB RAM minimum
- Docker Engine + Docker Compose plugin
- Open inbound port for the dashboard/API (default 8080), ideally behind a
  reverse proxy with TLS

Install Docker (Ubuntu):
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # then log out/in
```

### Step 1 — get the code

```bash
git clone <your-repo-url> wa-proxy
cd wa-proxy
```

### Step 2 — configure environment

```bash
cp .env.example .env
nano .env
```
At minimum, change for production:
```env
APP_ENV=production
BASIC_AUTH_USERNAME=your-admin
BASIC_AUTH_PASSWORD=a-strong-password
DB_PASSWORD=a-strong-db-password
```
Compose reads these via `${VAR}` substitution. You do **not** set `DB_HOST` /
`DB_PORT` for Docker — compose points the app at the `postgres` service on the
internal network automatically.

### Step 3 — build and start

```bash
docker compose up -d --build
```

Check status and logs:
```bash
docker compose ps
docker compose logs -f wa-proxy
```

### Step 4 — connect WhatsApp

1. Open `http://SERVER_IP:8080` and log in (Basic Auth).
2. Go to **Device → Connect**, scan the QR with WhatsApp (Linked Devices).
3. Once connected, the **API key** appears on the Device page — copy it for your
   integrations.

### Data persistence

Named volumes keep data across restarts/upgrades:
- `pgdata` — PostgreSQL (sessions, chats, contacts, API key)
- `storage` — media files (auto-pruned per `MEDIA_RETENTION_DAYS`)
- `logs` — daily activity logs

### Upgrades

```bash
git pull
docker compose up -d --build
```
The app runs DB migrations automatically on start. Volumes are preserved.

### Backups

```bash
# Database
docker compose exec postgres pg_dump -U "$DB_USER" "$DB_NAME" > backup_$(date +%F).sql
# Media (named volume)
docker run --rm -v wa-proxy_storage:/data -v "$PWD":/backup alpine \
  tar czf /backup/storage_$(date +%F).tgz -C /data .
```

---

## Production behind a reverse proxy (TLS)

Run wa-proxy bound to localhost and terminate TLS at Nginx/Caddy. The WebSocket
endpoint (`/ws`) must be proxied with upgrade headers.

**Caddy** (`Caddyfile`) — simplest, auto-HTTPS:
```
wa.example.com {
    reverse_proxy localhost:8080
}
```

**Nginx**:
```nginx
server {
    listen 443 ssl;
    server_name wa.example.com;
    # ssl_certificate / ssl_certificate_key ...

    client_max_body_size 100m;   # allow media uploads

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # WebSocket upgrade
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
    }
}
```

To bind wa-proxy only to localhost, change the compose port mapping to
`"127.0.0.1:8080:8080"`.

---

## Production without Docker (systemd)

If you build the binary on the server (Go 1.24+, Node 18+ required to build):

```bash
cd apps/frontend && npm ci && npm run build && cd ../..
go build -ldflags="-s -w" -o /usr/local/bin/wa-proxy ./apps/backend
sudo mkdir -p /opt/wa-proxy/{storage,logs}
sudo cp .env /opt/wa-proxy/.env   # production values; point DB_* at your Postgres
```

`/etc/systemd/system/wa-proxy.service`:
```ini
[Unit]
Description=WA Proxy
After=network.target postgresql.service

[Service]
Type=simple
WorkingDirectory=/opt/wa-proxy
ExecStart=/usr/local/bin/wa-proxy
Restart=always
RestartSec=5
User=waproxy

[Install]
WantedBy=multi-user.target
```
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now wa-proxy
sudo journalctl -u wa-proxy -f
```
You need a reachable PostgreSQL; set `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME`
in `/opt/wa-proxy/.env`.

---

## API Quick Reference

Dashboard endpoints use **Basic Auth**. Integration endpoints accept the
**`X-Api-Key`** header (key from the Device page) or Basic Auth. Full interactive
docs with live "Try it" are at **`/docs`** in the dashboard.

| Method | Path | Description |
| --- | --- | --- |
| POST | `/api/device/connect` | Start QR login / reconnect |
| GET | `/api/device/qr` | Latest QR code |
| GET | `/api/device/status` | Device status |
| POST | `/api/device/logout` | Logout WhatsApp |
| GET | `/api/device/api-key` | Current API key |
| POST | `/api/device/api-key/generate` | Generate / regenerate key |
| POST | `/api/messages/send` | Send text `{phone, message}` |
| POST | `/api/messages/send-image` | Send image (multipart) |
| POST | `/api/messages/send-file` | Send document (multipart) |
| POST | `/api/messages/send-buttons` | Message with CTA link(s) |
| POST | `/api/messages/check-exists` | Check numbers on WhatsApp |
| POST | `/api/messages/typing` | Typing indicator |
| POST | `/api/messages/seen` | Mark messages read |
| GET | `/api/conversations` | Inbox list |
| GET | `/api/conversations/:id` | Chat history |
| POST | `/api/conversations/:id/reply` | Reply text |
| POST | `/api/conversations/:id/reply-media` | Reply with media |
| POST | `/api/chats/start` | Start a 1:1 chat `{phone, message}` |
| GET | `/api/contacts?search=` | Contacts |
| POST | `/api/broadcasts` | Create broadcast |
| GET | `/api/broadcasts` / `/:id` | History / detail |
| GET | `/api/dashboard/stats` | Dashboard counters |
| GET | `/api/logs` / `/logs/days` | Activity logs |
| GET | `/api/storage-admin/overview` | Disk + media stats |
| GET | `/ws` | WebSocket realtime feed |

### Example: send a message

```bash
curl -X POST 'http://SERVER:8080/api/messages/send' \
  -H 'X-Api-Key: YOUR_API_KEY' \
  -H 'Content-Type: application/json' \
  -d '{"phone":"628123456789","message":"Halo Dunia"}'
```

### WebSocket events

```json
{ "type": "qr",              "data": { "code": "..." } }
{ "type": "device.status",   "data": { "status": "connected" } }
{ "type": "message.new",     "data": { "conversation_id": 1, "message": {...} } }
{ "type": "broadcast.update","data": { ...broadcast } }
```

---

## Security checklist (production)

- [ ] Change `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD` and `DB_PASSWORD`
- [ ] Put wa-proxy behind TLS (reverse proxy); bind to `127.0.0.1` in compose
- [ ] Treat the database as sensitive — it stores the API key and chat history
- [ ] Set `WEBHOOK_SECRET` if you use webhooks, and verify the signature
- [ ] Keep `RATE_LIMIT_ENABLED=true` and consider `HUMANIZE_SEND=true` to reduce
      WhatsApp ban risk
- [ ] Restrict who can reach port 8080 (firewall) if not behind a proxy

---

## Notes & limitations

- **Single device** per instance (Phase 1 scope). For multiple WhatsApp accounts,
  run one isolated instance per account (separate port + DB + volumes).
- **Native button messages** (template/interactive) are not reliably delivered by
  unofficial WhatsApp Web clients; the `send-buttons` endpoint sends a clickable
  link with a rich preview instead. True buttons require the official WhatsApp
  Business Cloud API (roadmap).
- Use responsibly and within WhatsApp's terms; aggressive sending risks bans.
