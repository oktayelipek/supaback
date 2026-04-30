# ── Stage 1: Frontend ─────────────────────────────────────────────────────────
FROM node:20-alpine AS frontend
WORKDIR /build
COPY apps/web/package*.json ./
RUN npm ci --prefer-offline
COPY apps/web/ ./
RUN npm run build

# ── Stage 2: Go binary ────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder
WORKDIR /build
# Download deps first (cached layer)
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o supaback ./cmd/supaback

# ── Stage 3: Final image ──────────────────────────────────────────────────────
FROM alpine:3.21

# pg_dump for database backups (matches Supabase PostgreSQL 15)
RUN apk add --no-cache postgresql17-client ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/supaback      ./supaback
COPY --from=frontend /build/dist         ./web/
COPY config.docker.yaml                  ./config.yaml

RUN mkdir -p /data /backups

EXPOSE 8080

ENTRYPOINT ["./supaback"]
CMD ["--config", "/app/config.yaml"]
