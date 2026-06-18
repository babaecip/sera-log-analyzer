# 🔍 Sera Log Analyzer

Sistem analisis log terdistribusi dengan AI-powered analysis, dibangun menggunakan **Go** dan **Docker**.

---

## Arsitektur

```
┌──────────────────────────────────────────┐
│              MASTER (:8080)               │
│  Dashboard │ SQLite │ AI │ Telegram Bot  │
│  Admin kirim command → Agent terima      │
└──────────────────┬───────────────────────┘
                   │  HTTP Polling
┌──────────────────┴───────────────────────┐
│              AGENT                        │
│  Scan file → Chunk → Kirim ke Master     │
│  (bisa jalan di mesin berbeda)           │
└──────────────────────────────────────────┘
```

## Cara Kerja

1. **Scan** — Agent scan path yang dikonfigurasi, cari file `.log`, `.log.2`, custom extension
2. **Select** — User pilih file mana yang mau dipantau lewat dashboard
3. **Chunk** — Agent baca N baris (default 3) dari file, kirim ke master
4. **Analyze** — Master kirim chunk ke AI (Ollama/OpenAI) untuk dianalisis
5. **Report** — Kalau AI deteksi masalah, master kirim alert ke Telegram
6. **Loop** — Master minta chunk berikutnya sampai file selesai
7. **Cleanup** — Chunk yang sudah diproses dihapus dari SQLite (auto cleanup per jam)

---

## ⚡ Quick Start — Full Stack (Master + Agent + Ollama)

```bash
docker compose up -d
```

Lalu pull model AI:
```bash
docker exec sera-ollama ollama pull qwen2.5:0.5b
```

Buka dashboard: **http://localhost:8080** → Login dengan credentials default:
- Username: `admin`
- Password: `sera-admin-2024`

> ⚠️ **Ganti password di `.env`** (`ADMIN_PASS`) sebelum production!

---

## 🖥️ Jalankan Master Saja

### Cara A — Docker CLI (Recommended, langsung pull dari Docker Hub)

```bash
# Pull image
docker pull mrseptian/sera-log-analyzer:master

# Jalankan master
docker run -d \
  --name sera-master \
  --restart unless-stopped \
  -p 8080:8080 \
  -e API_KEY=rahasia-kamu-disini \
  -e ADMIN_USER=admin \
  -e ADMIN_PASS=ganti-password-ini \
  -e MAX_STORAGE_MB=500 \
  -e AI_PROVIDER=ollama \
  -e AI_BASE_URL=http://host.docker.internal:11434 \
  -e AI_MODEL=qwen2.5:0.5b \
  -e TG_ENABLED=false \
  -v sera-data:/app/data \
  mrseptian/sera-log-analyzer:master
```

Buka: **http://localhost:8080**

### Cara B — Docker Compose

Kalau pakai docker-compose (full project), edit `.env` lalu:

```bash
cd sera-log-analyzer
docker compose up -d master
```

### 3. Cek Status

```bash
# Cek container
docker ps --filter "name=sera-master"

# Cek log
docker logs sera-master

# Cek health
curl http://localhost:8080/health
```

### 4. (Opsional) Jalankan Ollama di Mesin yang Sama

Kalau mau pakai AI local:

```bash
# Jalankan ollama
docker compose --profile ai up -d ollama

# Pull model
docker exec sera-ollama ollama pull qwen2.5:0.5b

# Update .env: AI_BASE_URL=http://ollama:11434
```

---

## 🤖 Jalankan Agent Saja

Agent bisa dijalankan di **mesin mana saja** yang bisa akses master. Ada 3 cara:

### Cara A — Agent via Docker CLI (Recommended, langsung pull dari Docker Hub)

```bash
# Pull image
docker pull mrseptian/sera-log-analyzer:agent

# Jalankan agent, hubungkan ke master di IP lain
docker run -d \
  --name sera-agent \
  --restart unless-stopped \
  -e MASTER_URL=http://IP_MASTER:8080 \
  -e MASTER_KEY=rahasia-kamu-disini \
  -e AGENT_NAME=agent-kantor \
  -e POLL_INTERVAL=3 \
  -e SCAN_ROOTS=/var/log,/tmp \
  -e EXTENSIONS=.log,.log.2,.err \
  -v /var/log:/var/log:ro \
  -v /tmp:/tmp:ro \
  mrseptian/sera-log-analyzer:agent
```

> Ganti `IP_MASTER` dengan IP publik/privat mesin yang menjalankan master.

### Cara B — Agent via Docker Compose (Lokal)

Kalau master & agent di mesin yang sama, edit `.env` lalu:

```bash
docker-compose up -d master agent-1
```

### Cara C — Agent via Binary (Tanpa Docker)

Build di mesin yang ada Go:

```bash
cd agent
go build -o sera-agent .
```

Jalankan:

```bash
MASTER_URL=http://IP_MASTER:8080 \
MASTER_KEY=rahasia-kamu-disini \
AGENT_NAME=agent-rumah \
SCAN_ROOTS=/var/log,/opt/app/logs \
EXTENSIONS=.log,.log.2 \
POLL_INTERVAL=3 \
./sera-agent
```

---

## 📁 Struktur Project

```
sera-log-analyzer/
├── docker-compose.yml          # Orchestration semua service
├── .env                        # Konfigurasi environment
├── master/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go                 # HTTP server, DB, handlers
│   ├── ai.go                   # Integrasi AI (Ollama/OpenAI)
│   ├── telegram.go             # Integrasi Telegram bot
│   ├── dashboard.go            # Web dashboard (embedded HTML)
│   ├── types.go                # Type definitions
│   └── data/                   # SQLite database (auto-created)
├── agent/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go                 # Scanner, chunk reader, polling
│   └── types.go                # Type definitions
└── shared/
    ├── go.mod
    └── types.go                # Shared types
```

---

## 🔧 Environment Variables

### Master

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `PORT` | `8080` | Port HTTP master |
| `API_KEY` | `sera-default-key` | **Wajib diganti!** Kunci autentikasi agent |
| `ADMIN_USER` | `admin` | Username login dashboard |
| `ADMIN_PASS` | `sera-admin-2024` | **Wajib diganti!** Password login dashboard (bcrypt hashed) |
| `MAX_STORAGE_MB` | `500` | Maks ukuran SQLite (tolak data kalau penuh) |
| `AI_PROVIDER` | `ollama` | `ollama`, `openai`, atau `openai-compatible` |
| `AI_BASE_URL` | `http://ollama:11434` | URL endpoint AI |
| `AI_MODEL` | `qwen2.5:0.5b` | Nama model AI |
| `AI_API_KEY` | _(kosong)_ | API key (untuk OpenAI) |
| `AI_CHUNK_SIZE` | `3` | Jumlah baris per request ke AI |
| `TG_ENABLED` | `false` | Aktifkan notifikasi Telegram |
| `TG_BOT_TOKEN` | _(kosong)_ | Token bot Telegram |
| `TG_CHAT_ID` | _(kosong)_ | Chat ID Telegram |

### Agent

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `MASTER_URL` | `http://master:8080` | URL master server |
| `MASTER_KEY` | `sera-default-key` | Harus sama dengan `API_KEY` master |
| `AGENT_NAME` | `agent-1` | Nama tampilan agent |
| `AGENT_ID` | _(kosong)_ | Auto-generate kalau kosong |
| `POLL_INTERVAL` | `3` | Detik antar poll ke master |
| `SCAN_ROOTS` | `/var/log,/tmp` | Path scan (koma-pisah) |
| `EXTENSIONS` | `.log` | Extension file (koma-pisah) |

---

## 🌐 API Endpoints

### Login / Auth

| Method | Path | Deskripsi |
|--------|------|-----------|
| GET | `/login` | Halaman login (browser) |
| POST | `/api/login` | Login (username + password + captcha) |
| POST | `/api/logout` | Logout (hapus session) |
| GET | `/api/captcha` | Ambil CAPTCHA baru |
| GET | `/api/session` | Cek status session (authenticated?) |

### Agent API (butuh header `X-API-Key`)

| Method | Path | Deskripsi |
|--------|------|-----------|
| POST | `/api/agent/register` | Daftarkan agent ke master |
| POST | `/api/agent/heartbeat` | Kirim heartbeat |
| POST | `/api/agent/poll` | Ambil command dari master |
| POST | `/api/agent/chunk` | Kirim log chunk ke master |
| POST | `/api/agent/scan-result` | Kirim hasil scan file |
| POST | `/api/agent/report` | Kirim laporan langsung |

### Admin API

| Method | Path | Deskripsi |
|--------|------|-----------|
| GET | `/api/agents` | List semua agent |
| GET | `/api/agents/{id}` | Detail agent |
| POST | `/api/command` | Kirim command ke agent |
| GET | `/api/files` | List file yang ditemukan |
| POST | `/api/files/select` | Pilih file untuk dipantau |
| GET | `/api/reports` | List laporan AI |
| GET | `/api/storage` | Info penggunaan storage |
| GET/PUT | `/api/config/ai` | Lihat/ubah config AI |
| GET/PUT | `/api/config/telegram` | Lihat/ubah config Telegram |

---

## ⚠️ Storage Limits

Ketika SQLite mencapai `MAX_STORAGE_MB`:
- Master **menolak** chunk baru dari agent
- Agent tetap jalan tapi tidak dapat command baru
- Chunk lama (>1 jam) otomatis dihapus
- User harus upgrade `MAX_STORAGE_MB` atau cleanup manual
