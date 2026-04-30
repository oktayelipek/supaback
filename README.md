# SupaBack

Self-hosted backup tool for Supabase. Backs up your PostgreSQL database and Storage buckets on a schedule, stores results locally, on any SSH server, or in S3-compatible storage — and provides a web UI for everything.

![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)
![License](https://img.shields.io/badge/license-MIT-green?style=flat)

---

## Features

- **Database backup** — runs `pg_dump` (custom format, optional gzip compression)
- **Storage backup** — recursively downloads all files from Supabase Storage buckets
- **Cron scheduling** — multiple schedules with standard cron expressions
- **Retention policy** — auto-delete old backups by count or age
- **Web UI** — dashboard, schedule management, and settings configurable at runtime (no restart needed)
- **Three destination types** — local disk, SFTP (Raspberry Pi, NAS, VPS), S3-compatible (AWS, R2, MinIO)
- **MinIO bundle** — fully self-hosted S3 storage with one extra compose flag
- **Zero-dependency runtime** — single binary + SQLite, no external database needed
- **Docker-first** — one `docker compose up` and you're running

---

## Quick Start

### Docker Compose

```bash
git clone https://github.com/oktayelipek/supaback.git
cd supaback

cp .env.example .env
# edit .env — add your Supabase credentials
```

```bash
docker compose up -d
```

Open **http://localhost:8080** and enter your credentials in **Settings**.

> All configuration can be done through the UI — no config file editing required.

---

## Destinations

### Local disk

Backups are written to a directory on the machine running SupaBack. Default path inside Docker: `/backups` (mounted volume).

```yaml
destination:
  type: local
  local_path: /backups
```

### SFTP

Back up to any SSH-capable machine on your network or the internet. Works with Raspberry Pi, Synology/QNAP NAS, TrueNAS, and any Linux VPS.

```yaml
destination:
  type: sftp
  sftp:
    host: 192.168.1.100       # or hostname
    port: 22
    user: backup-user
    password: ""              # use password OR key_path
    key_path: /root/.ssh/id_rsa
    remote_path: /mnt/backups/supabase
```

All fields are also configurable from the **Settings** UI.

### S3-compatible

Works with AWS S3, Cloudflare R2, MinIO, and any other S3-compatible service.

**AWS S3**
```env
S3_REGION=us-east-1
S3_BUCKET=my-supabase-backups
S3_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
S3_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

**Cloudflare R2**
```env
S3_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com
S3_REGION=auto
S3_BUCKET=my-supabase-backups
S3_ACCESS_KEY_ID=...
S3_SECRET_ACCESS_KEY=...
```

**MinIO (self-hosted, see section below)**
```env
S3_ENDPOINT=http://minio:9000
S3_REGION=us-east-1
S3_BUCKET=supaback
S3_ACCESS_KEY_ID=minioadmin
S3_SECRET_ACCESS_KEY=minioadmin
```

Enable **Force path style** in Settings for MinIO.

---

## Self-Hosted Storage with MinIO

Run SupaBack and MinIO together — no cloud account needed.

```bash
docker compose -f docker-compose.yml -f docker-compose.minio.yml up -d
# shortcut:
make docker-minio
```

| Service | URL |
|---------|-----|
| SupaBack UI | http://localhost:8080 |
| MinIO console | http://localhost:9001 |

Default credentials: `minioadmin / minioadmin`. Change them in `docker-compose.minio.yml` before deploying.

The `minio-init` service automatically creates the `supaback` bucket on first start.

---

## Retention Policy

Automatically delete old backups to prevent storage from filling up.

Configure in **Settings → Retention** or via config:

```yaml
backup:
  retention:
    keep_last: 7   # keep the 7 most recent backup dates
    keep_days: 30  # keep anything from the last 30 days
```

| Rule | Behaviour |
|------|-----------|
| `keep_last: N` | Keeps the N most recent backup dates, deletes the rest |
| `keep_days: N` | Keeps backups newer than N days, deletes the rest |
| Both set | A backup is kept if **either** rule says to keep it |
| Both 0 | Retention is disabled — nothing is ever deleted |

Retention runs automatically after every successful backup job and works on all three destination types.

---

## Backup Schedule

Create schedules from the **Schedules** page. Standard 5-field cron syntax:

| Expression | Meaning |
|------------|---------|
| `0 * * * *` | Every hour |
| `0 2 * * *` | Daily at 02:00 |
| `0 2 * * 0` | Every Sunday at 02:00 |
| `0 2 1 * *` | 1st of every month at 02:00 |
| `0 */6 * * *` | Every 6 hours |

Each schedule can target the full instance, database only, or storage only. Multiple schedules can coexist.

---

## Configuration

Configuration is layered — each level overrides the previous:

```
config.yaml  →  environment variables  →  Settings UI (stored in SQLite)
```

The Settings UI has the highest priority and persists across restarts.

### Environment Variables

**Supabase**

| Variable | Description |
|----------|-------------|
| `SUPABASE_URL` | Project URL |
| `SUPABASE_SERVICE_KEY` | `service_role` API key |
| `SUPABASE_DB_URL` | PostgreSQL connection string (URI) |

**Server / paths**

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `STORE_PATH` | SQLite database path | `./supaback.db` |
| `LOCAL_BACKUP_PATH` | Local backup directory | `./backups` |
| `STATIC_DIR` | Path to built frontend | — |

**S3**

| Variable | Description |
|----------|-------------|
| `S3_ENDPOINT` | Custom endpoint (leave empty for AWS) |
| `S3_REGION` | Region (default `us-east-1`) |
| `S3_BUCKET` | Bucket name |
| `S3_ACCESS_KEY_ID` | Access key |
| `S3_SECRET_ACCESS_KEY` | Secret key |

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
  buckets: []           # empty = all buckets
  retention:
    keep_last: 7
    keep_days: 30

destination:
  type: "local"         # "local", "sftp", or "s3"
  local_path: "./backups"

  # sftp:
  #   host: "192.168.1.100"
  #   port: 22
  #   user: "backup-user"
  #   password: ""
  #   key_path: "/root/.ssh/id_rsa"
  #   remote_path: "/mnt/backups/supabase"

  # s3:
  #   endpoint: ""
  #   region: "us-east-1"
  #   bucket: "my-backups"
  #   prefix: "supabase"
  #   access_key_id: ""
  #   secret_access_key: ""
  #   force_path_style: false

server:
  host: "0.0.0.0"
  port: 8080

store:
  path: "./supaback.db"
```

---

## Backup File Layout

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

The top-level date directory is what retention acts on — removing `2024-05-01/` removes the entire backup for that day.

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

## API Reference

All endpoints return JSON. Base path: `/api`

### Health

```
GET /api/health
→ { "status": "ok", "configured": true }
```

### Settings

```
GET /api/settings       → current configuration
PUT /api/settings       → update and persist configuration
```

### Jobs

```
GET  /api/jobs?limit=50  → list recent backup jobs
POST /api/jobs           → trigger a backup immediately
```

POST body: `{ "type": "full" }` — `"full"` | `"database"` | `"storage"`

Response: `202 Accepted` — backup runs asynchronously.

### Schedules

```
GET    /api/schedules              → list all schedules
POST   /api/schedules              → create a schedule
DELETE /api/schedules/:id          → delete a schedule
PATCH  /api/schedules/:id/toggle   → enable or disable
```

POST body:
```json
{
  "name": "Daily backup",
  "cron_expr": "0 2 * * *",
  "type": "full"
}
```

---

## Development

### Prerequisites

- Go 1.23+
- Node.js 20+
- `pg_dump` (`brew install libpq` on macOS, `apt install postgresql-client` on Linux)

### Running locally

```bash
make web-install        # install npm dependencies

cp config.example.yaml config.yaml
# edit config.yaml or leave empty and configure via UI

make serve              # Go API → http://localhost:8080/api
make web-dev            # Vite dev server → http://localhost:5173
```

The Vite dev server proxies all `/api` requests to `:8080`.

### Building for production

```bash
make web-build          # → apps/web/dist
make build              # → bin/supaback
./bin/supaback --config config.yaml
```

### Project structure

```
supaback/
├── cmd/supaback/           # entry point
├── internal/
│   ├── api/                # HTTP server, routes, handlers
│   ├── appstate/           # thread-safe config + destination holder
│   ├── backup/             # pg_dump runner, storage downloader, orchestrator
│   ├── config/             # structs, env loading, settings keys
│   ├── destination/        # local, SFTP, S3 writers (shared interface)
│   ├── retention/          # retention policy engine
│   ├── scheduler/          # cron engine wrapper
│   └── store/              # SQLite: jobs, schedules, settings
├── apps/web/               # React + Vite frontend
│   └── src/
│       ├── components/     # Navbar, Modal, StatusBadge
│       ├── lib/            # API client, formatters
│       └── pages/          # Dashboard, Schedules, Settings
├── config.example.yaml
├── config.docker.yaml      # defaults baked into Docker image
├── Dockerfile              # multi-stage build
├── docker-compose.yml
└── docker-compose.minio.yml  # MinIO self-hosted storage bundle
```

### Makefile reference

| Target | Description |
|--------|-------------|
| `make build` | Compile Go binary |
| `make serve` | Run API server (dev) |
| `make test` | Run Go tests |
| `make tidy` | Tidy Go modules |
| `make web-install` | Install npm dependencies |
| `make web-dev` | Start Vite dev server |
| `make web-build` | Build frontend for production |
| `make docker-build` | Build Docker image |
| `make docker-up` | Start with Docker Compose |
| `make docker-down` | Stop containers |
| `make docker-logs` | Tail container logs |
| `make docker-minio` | Start SupaBack + MinIO bundle |

---

## Deploying to Coolify

[Coolify](https://coolify.io) is a self-hosted PaaS that can deploy SupaBack directly from GitHub.

### Steps

1. **New Resource → Docker Compose**
   - Source: GitHub → `oktayelipek/supaback`, branch `main`
   - Compose file: `docker-compose.yml`

2. **Environment Variables** — wrap JWT values in double quotes to prevent shell parsing errors:
   ```
   SUPABASE_URL="https://xxxxxxxxxxxx.supabase.co"
   SUPABASE_SERVICE_KEY="eyJhbGci..."
   SUPABASE_DB_URL="postgresql://postgres:[password]@db.xxxxxxxxxxxx.supabase.co:5432/postgres"
   ```

3. **Port configuration** — this is important:

   | Setting | Behaviour |
   |---------|-----------|
   | **Port Mappings** (host:container) | Binds directly to the host — risk of port conflicts |
   | **Ports Exposes** | Only tells Traefik the container listens on that port — no host binding |

   → Leave **Port Mappings empty**, set **Ports Exposes to `8080`**.  
   Traefik routes traffic to the container internally; nothing is exposed directly to the host.

4. **Domain** — set your domain in Coolify. SSL is provisioned automatically via Let's Encrypt.

5. **Deploy** — first build takes ~3–5 minutes (Node + Go compile). Subsequent deploys are faster due to layer caching.

### Volumes

Coolify manages Docker volumes automatically. After the first deploy, confirm these exist under your resource's **Volumes** tab:

- `supaback-data` → SQLite database (settings, job history)
- `supaback-backups` → backup files (when using local destination)

---

## Docker details

Three-stage build:

1. **`node:20-alpine`** — builds the React frontend
2. **`golang:1.23-alpine`** — compiles a static Go binary (CGO disabled)
3. **`alpine:3.21`** — final image with `postgresql17-client` for `pg_dump`

Final image size: ~**70–90 MB**

### Volumes

| Volume | Mount | Contents |
|--------|-------|----------|
| `supaback-data` | `/data` | SQLite DB (settings, job history) |
| `supaback-backups` | `/backups` | Backup files (local destination) |

### Updating

```bash
docker compose build     # rebuild from source
docker compose up -d
```

---

## Roadmap

- [ ] Email / webhook notifications on failure
- [ ] Restore UI — browse and download backup files from the UI
- [ ] Multiple Supabase project support
- [ ] Backup encryption at rest

---

## Contributing

Pull requests are welcome. For major changes please open an issue first.

```bash
git checkout -b feature/my-feature
# make changes
go test ./...
git commit -m "feat: my feature"
git push origin feature/my-feature
```

---

## License

MIT — see [LICENSE](LICENSE) file.
