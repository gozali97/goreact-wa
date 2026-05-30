# PRD - WA Proxy Platform

## 1. Overview

### Nama Project

WA Proxy Platform

### Tujuan

Membangun platform WhatsApp Gateway berbasis WhatsApp Web yang memungkinkan:

* Login WhatsApp menggunakan QR Code
* Mengirim pesan ke nomor tujuan
* Mengirim gambar
* Mengirim dokumen/file
* Broadcast pesan
* Monitoring seluruh percakapan
* Membalas chat melalui dashboard
* Menyediakan API untuk aplikasi eksternal
* Menyimpan seluruh histori chat dalam PostgreSQL

Platform ditujukan sebagai self-hosted solution dan dapat dijalankan dalam satu paket deployment menggunakan Docker.

---

# 2. Technology Stack

## Backend

* Golang 1.24+
* Gin Framework
* GORM
* PostgreSQL
* Redis (opsional untuk queue)
* WhatsApp Web Client Library

  * go.mau.fi/whatsmeow

## Frontend

* React JS
* Vite
* React Router
* TanStack Query
* Zustand
* FlyonUI
* TailwindCSS

## Database

* PostgreSQL

## Deployment

* Docker
* Docker Compose

---

# 3. High Level Architecture

```text
┌────────────────────┐
│     React UI       │
│    Dashboard       │
└─────────┬──────────┘
          │
          ▼
┌────────────────────┐
│     Golang API     │
│     Gin Server     │
└─────────┬──────────┘
          │
          ▼
┌────────────────────┐
│ WhatsApp Session   │
│     Whatsmeow      │
└─────────┬──────────┘
          │
          ▼
┌────────────────────┐
│    PostgreSQL      │
└────────────────────┘
```

---

# 4. Authentication

## Dashboard Authentication

Tidak menggunakan user management.

Menggunakan Basic Authentication.

### Environment Variable

```env
BASIC_AUTH_USERNAME=admin
BASIC_AUTH_PASSWORD=secret123
```

### Flow

1. User membuka dashboard
2. Browser meminta Basic Auth
3. Jika valid:

   * akses dashboard
4. Jika tidak valid:

   * 401 Unauthorized

---

# 5. Core Features

## F01 - WhatsApp Login

### Deskripsi

Menghubungkan akun WhatsApp melalui QR Code.

### Flow

1. User membuka halaman Device
2. Klik Connect
3. Backend generate QR
4. QR ditampilkan
5. User scan QR menggunakan WhatsApp
6. Session tersimpan di database

### Acceptance Criteria

* QR dapat dibuat
* Session tersimpan
* Session otomatis reconnect

---

## F02 - Device Status

### Status

* Connected
* Connecting
* Disconnected
* Logged Out

### Dashboard

Menampilkan:

* Nama akun
* Nomor WA
* Status
* Last Seen

---

## F03 - Send Message

### Endpoint

```http
POST /api/messages/send
```

### Request

```json
{
  "phone": "628123456789",
  "message": "Halo Dunia"
}
```

### Response

```json
{
  "success": true,
  "message_id": "xxx"
}
```

---

## F04 - Send Image

### Endpoint

```http
POST /api/messages/send-image
```

### Request

multipart/form-data

```text
phone
caption
file
```

---

## F05 - Send Document

### Endpoint

```http
POST /api/messages/send-file
```

### Request

multipart/form-data

```text
phone
caption
file
```

---

## F06 - Broadcast

### Deskripsi

Mengirim pesan ke banyak nomor.

### Request

```json
{
  "phones": [
    "62811111111",
    "62822222222"
  ],
  "message": "Promo Hari Ini"
}
```

### Requirement

* Queue Based
* Delay Random
* Retry Failed

### Status

* Pending
* Sending
* Success
* Failed

---

## F07 - Inbox Monitoring

### Deskripsi

Semua pesan masuk disimpan.

### Data Ditampilkan

* Nama kontak
* Nomor
* Pesan terakhir
* Waktu terakhir

---

## F08 - Chat Detail

### Deskripsi

Melihat histori percakapan.

### Data

* Incoming Message
* Outgoing Message
* Attachment
* Timestamp

---

## F09 - Reply Message

### Deskripsi

Membalas chat langsung dari dashboard.

### Flow

1. Buka chat
2. Ketik pesan
3. Klik send
4. Pesan terkirim melalui WhatsApp

---

## F10 - Contact Management

### Data

* Nama
* Nomor
* Last Chat

### Fitur

* Search
* Filter
* Detail Contact

---

## F11 - API Documentation

### Halaman

/dashboard/docs

### Isi

* Authentication
* Send Message
* Send Image
* Send File
* Broadcast
* Webhook

---

# 6. Database Design

## Table sessions

```sql
id
phone
push_name
status
session_data
created_at
updated_at
```

---

## Table contacts

```sql
id
phone
name
avatar
last_message_at
created_at
updated_at
```

---

## Table conversations

```sql
id
contact_id
last_message
last_message_at
created_at
updated_at
```

---

## Table messages

```sql
id
conversation_id
message_id
direction
message_type
content
media_url
status
sent_at
created_at
```

direction:

* incoming
* outgoing

message_type:

* text
* image
* document

status:

* pending
* sent
* delivered
* read
* failed

---

## Table broadcasts

```sql
id
name
message
status
total_target
total_success
total_failed
created_at
updated_at
```

---

## Table broadcast_details

```sql
id
broadcast_id
phone
status
error_message
created_at
```

---

# 7. Storage Strategy

Semua data menggunakan PostgreSQL.

Media file disimpan:

```text
/storage
```

Database hanya menyimpan path.

Contoh:

```text
/storage/images/xxx.jpg
/storage/docs/xxx.pdf
```

Catatan:

Untuk skala besar nanti dapat dipindahkan ke:

* S3
* MinIO
* Cloudflare R2

Tanpa mengubah struktur aplikasi.

---

# 8. Frontend Pages

## Login

Basic Auth Browser

---

## Dashboard

Menampilkan:

* Device Status
* Total Contact
* Total Chat
* Total Broadcast

---

## Device

* QR Code
* Connection Status
* Logout

---

## Chats

Sidebar:

* List Contact

Content:

* Chat History
* Reply Form

---

## Broadcast

* Create Broadcast
* Broadcast History

---

## API Docs

* Endpoint List
* Sample Request
* Sample Response

---

## Settings

* App Configuration
* Device Information

---

# 9. Background Workers

## Message Sender

Menangani:

* Send Message
* Send Image
* Send File

---

## Broadcast Worker

Menangani:

* Queue Broadcast
* Retry
* Delay Random

---

## Session Worker

Menangani:

* Auto Reconnect
* Session Monitoring

---

# 10. Deployment

## Development

```bash
make dev
```

Menjalankan:

```bash
backend
frontend
postgres
```

secara bersamaan.

---

## Production

Docker Compose

```text
app
postgres
```

### Command

```bash
docker compose up -d
```

---

# 11. Monorepo Structure

```text
wa-proxy/

apps/
 ├── backend/
 └── frontend/

internal/
 ├── api/
 ├── service/
 ├── repository/
 ├── worker/
 └── whatsapp/

storage/

deploy/
 ├── docker/
 └── compose/

docs/

Makefile
docker-compose.yml
```

---

# 12. Future Roadmap

Phase 2

* Multi Device
* Multi Session
* User Management
* Role Permission
* Webhook
* REST API Key
* Rate Limiter

Phase 3

* Official WhatsApp Cloud API Support
* Hybrid WA Web + Cloud API
* SaaS Version
* Multi Tenant
* Billing System
