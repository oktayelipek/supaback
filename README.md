# SupaBack

Self-hosted backup tool for Supabase. Backs up your PostgreSQL database and Storage buckets on a schedule, stores results locally or in any S3-compatible service, and provides a web UI for management — all in a single Docker container.

![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)
![License](https://img.shields.io/badge/license-MIT-green?style=flat)

---

## Features

- **Database backup** — runs `pg_dump` (custom format, optional gzip compression)
- **Storage backup** — recursively downloads all files from Supabase Storage buckets
- **Cron scheduling** — create multiple schedules with standard cron expressions
- **Web UI** — dashboard, schedule management, and settings — all configurable at runtime
- **Multiple destinations** — local filesystem, AWS S3, Cloudflare R2, or MinIO
- **Zero-dependency runtime** — single binary + SQLite, no external database needed
- **Docker-first** — one `docker compose up` and you're running

---

## Quick Start

### Docker Compose (recommended)

```bash
git clone https://github.com/supaback/supaback.git
cd supaback

cp .env.example .env
```

Edit `.env` with your Supabase credentials:

```env
SUPABASE_URL=https://xxxxxxxxxxxx.supabase.co
SUPABASE_SERVICE_KEY=eyJhbGci...
SUPABASE_DB_URL=postgresql://postgres:[password]@db.xxxxxxxxxxxx.supabase.co:5432/postgres
```

```bash
docker compose up -d
```

Open **http://localhost:8080** — your instance is ready.

> Credentials can also be entered directly in the **Settings** page of the UI without editing any files.

---

## Screenshots

```
┌─────────────────────────────────────────────────────────────┐
│  SupaBack   Dashboard   Schedules   Settings                 │
├─────────────────────────────────────────────────────────────┤
│                                              [Run Backup ▶]  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │ Total    │ │ Success  │ │  Failed  │ │ Data Backed  │   │
│  │   42     │ │   40     │ │    2     │ │   1.4 GB     │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
│                                                              │
│  Backup History                                              │
│  ┌──────────────┬──────────┬─────────┬────────┬──────────┐  │
│  │ Started      │ Type     │ Status  │ Size   │ Duration │  │
│  ├──────────────┼──────────┼─────────┼────────┼──────────┤  │
│  │ May 1, 02:00 │ full     │ ✓ Done  │ 48 MB  │ 1m 12s   │  │
│  │ Apr 30, 2:00 │ full     │ ✓ Done  │ 47 MB  │ 1m 08s   │  │
│  └──────────────┴──────────┴─────────┴────────┴──────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## Configuration

Configuration is layered — each level overrides the previous:

```
config.yaml  →  environment variables  →  Settings UI (stored in SQLite)
```

The Settings UI has the highest priority and persists across restarts. You don't need to touch any files after initial setup.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SUPABASE_URL` | Supabase project URL | — |
| `SUPABASE_SERVICE_KEY` | `service_role` API key | — |
| `SUPABASE_DB_URL` | PostgreSQL connection string (URI) | — |
| `PORT` | HTTP server port | `8080` |
| `STORE_PATH` | Path to SQLite database | `./supaback.db` |
| `LOCAL_BACKUP_PATH` | Local backup directory | `./backups` |
| `STATIC_DIR` | Path to built frontend files | — |
| `S3_ENDPOINT` | S3-compatible endpoint URL | — |
| `S3_REGION` | S3 region | `us-east-1` |
| `S3_BUCKET` | S3 bucket name | — |
| `S3_ACCESS_KEY_ID` | S3 access key | — |
| `S3_SECRET_ACCESS_KEY` | S3 secret key | — |

### config.yaml

```yaml
supabase:
  url: "https://xxxxxxxxxxxx.supabase.co"
  service_key: "eyJhbGci..."
  database_url: "postgresql://postgres:[password]@db.xxxxxxxxxxxx.supabase.co:5432/postgres"

backup:
  include_database: true
  include_storage: true
  compress: true
  buckets: []          # empty = all buckets

destination:
  type: "local"        # "local" or "s3"
  local_path: "./backups"

  # S3-compatible (uncomment to use)
  # s3:
  #   endpoint: ""     # leave empty for AWS, set for R2/MinIO
  #   region: "us-east-1"
  #   bucket: "my-backups"
  #   prefix: "supabase"
  #   access_key_id: ""
  #   secret_access_key: ""
  #   force_path_style: false   # set true for MinIO

server:
  host: "0.0.0.0"
  port: 8080

store:
  path: "./supaback.db"
```

---

## Destination: S3-compatible

SupaBack works with any S3-compatible object storage.

### AWS S3

```env
S3_REGION=us-east-1
S3_BUCKET=my-supabase-backups
S3_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
S3_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

### Cloudflare R2

```env
S3_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com
S3_REGION=auto
S3_BUCKET=my-supabase-backups
S3_ACCESS_KEY_ID=...
S3_SECRET_ACCESS_KEY=...
```

### MinIO (self-hosted)

```env
S3_ENDPOINT=http://minio:9000
S3_REGION=us-east-1
S3_BUCKET=supaback
S3_ACCESS_KEY_ID=minioadmin
S3_SECRET_ACCESS_KEY=minioadmin
```

Also enable **Force path style** in the Settings UI or set `force_path_style: true` in config.

---

## Backup Schedule

Schedules use standard 5-field cron syntax. Create and manage them from the **Schedules** page.

| Expression | Meaning |
|------------|---------|
| `0 * * * *` | Every hour |
| `0 2 * * *` | Daily at 02:00 |
| `0 2 * * 0` | Every Sunday at 02:00 |
| `0 2 1 * *` | 1st of every month at 02:00 |
| `0 */6 * * *` | Every 6 hours |

Each schedule can back up the full instance, database only, or storage only. Multiple schedules can run concurrently.

---

## API Reference

All endpoints return JSON. Base path: `/api`

### Health

```
GET /api/health
```
```json
{ "status": "ok", "configured": true }
```

### Settings

```
GET  /api/settings          → current configuration
PUT  /api/settings          → update configuration (persisted to DB)
```

**PUT body example:**
```json
{
  "supabase_url": "https://xxx.supabase.co",
  "supabase_service_key": "eyJhbGci...",
  "supabase_db_url": "postgresql://...",
  "backup_include_database": "true",
  "backup_include_storage": "true",
  "backup_compress": "true",
  "destination_type": "local",
  "destination_local_path": "/backups"
}
```

### Jobs

```
GET  /api/jobs?limit=50     → list recent backup jobs
POST /api/jobs              → trigger a backup immediately
```

**POST body:**
```json
{ "type": "full" }          // "full" | "database" | "storage"
```

**Response:** `202 Accepted` — backup runs in the background.

### Schedules

```
GET    /api/schedules              → list all schedules
POST   /api/schedules              → create a schedule
DELETE /api/schedules/:id          → delete a schedule
PATCH  /api/schedules/:id/toggle   → enable or disable
```

**POST body:**
```json
{
  "name": "Daily backup",
  "cron_expr": "0 2 * * *",
  "type": "full"
}
```

---

## Backup File Layout

Backups are organized by date inside the destination directory:

```
backups/
└── 2024-05-01/
    ├── database/
    │   └── postgres_20240501_020000.dump.gz
    └── storage/
        ├── avatars/
        │   ├── user-123/avatar.png
        │   └── user-456/avatar.jpg
        └── documents/
            └── report-q1.pdf
```

Database dumps use `pg_dump --format=custom`, which can be restored with `pg_restore`.

**Restore example:**
```bash
pg_restore \
  --host db.xxxxxxxxxxxx.supabase.co \
  --port 5432 \
  --username postgres \
  --dbname postgres \
  --no-owner \
  backups/2024-05-01/database/postgres_20240501_020000.dump.gz
```

---

## Development

### Prerequisites

- Go 1.23+
- Node.js 20+
- `pg_dump` (install `postgresql-client` on Linux/macOS)

### Running locally

```bash
# 1. Install frontend dependencies
make web-install

# 2. Copy and edit config
cp config.example.yaml config.yaml

# 3. Start the Go API server
make serve                  # → http://localhost:8080/api

# 4. In a separate terminal, start the frontend dev server
make web-dev                # → http://localhost:5173
```

The Vite dev server proxies all `/api` requests to `:8080`, so hot-reload works end-to-end.

### Building for production

```bash
make web-build   # compiles frontend into apps/web/dist
make build       # compiles Go binary into bin/supaback
./bin/supaback --config config.yaml
```

### Project structure

```
supaback/
├── cmd/supaback/           # main entry point
├── internal/
│   ├── api/                # HTTP server, routes, handlers
│   ├── appstate/           # thread-safe config + destination holder
│   ├── backup/             # pg_dump runner, storage downloader, orchestrator
│   ├── config/             # config struct, env loading, settings keys
│   ├── destination/        # local filesystem and S3 writers
│   ├── scheduler/          # cron engine wrapper
│   └── store/              # SQLite: jobs, schedules, settings
├── apps/web/               # React + Vite frontend
│   └── src/
│       ├── components/     # Navbar, Modal, StatusBadge
│       ├── lib/            # API client, formatters
│       └── pages/          # Dashboard, Schedules, Settings
├── config.example.yaml     # config template
├── config.docker.yaml      # default config for Docker image
├── Dockerfile              # multi-stage build
└── docker-compose.yml
```

### Makefile reference

| Target | Description |
|--------|-------------|
| `make build` | Compile Go binary |
| `make serve` | Run API server (dev) |
| `make test` | Run Go tests |
| `make tidy` | Tidy Go modules |
| `make web-install` | Install npm dependencies |
| `make web-dev` | Start Vite dev server with hot reload |
| `make web-build` | Build frontend for production |
| `make docker-build` | Build Docker image |
| `make docker-up` | Start with Docker Compose |
| `make docker-down` | Stop containers |
| `make docker-logs` | Tail container logs |

---

## Docker details

The Docker image uses a 3-stage build:

1. **`node:20-alpine`** — builds the React frontend
2. **`golang:1.23-alpine`** — compiles the Go binary (static, CGO disabled)
3. **`alpine:3.21`** — final image with `postgresql15-client` for `pg_dump`

Final image size is approximately **70–90 MB**.

### Volumes

| Volume | Mount path | Contents |
|--------|------------|----------|
| `supaback-data` | `/data` | SQLite database (settings, job history) |
| `supaback-backups` | `/backups` | Backup files (when using local destination) |

### Updating

```bash
docker compose pull          # if using a registry image
# or
docker compose build         # rebuild from source
docker compose up -d
```

---

## Roadmap

- [ ] Backup retention policy (auto-delete old backups)
- [ ] Email / webhook notifications on failure
- [ ] Restore UI (browse and download backup files)
- [ ] Multiple Supabase project support
- [ ] GitHub Actions integration for off-site backups

---

## Contributing

Pull requests are welcome. For major changes, open an issue first.

```bash
git checkout -b feature/my-feature
# make your changes
go test ./...
git commit -m "feat: my feature"
git push origin feature/my-feature
```

Please keep PRs focused — one feature or fix per PR.

---

## License

MIT — see [LICENSE](LICENSE) file.
