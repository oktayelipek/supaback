.PHONY: build run serve tidy test web-install web-dev web-build

build:
	go build -o bin/supaback ./cmd/supaback

run:
	go run ./cmd/supaback --run-now --config config.yaml

serve:
	go run ./cmd/supaback --config config.yaml

tidy:
	go mod tidy

test:
	go test ./...

web-install:
	cd apps/web && npm install

web-dev:
	cd apps/web && npm run dev

web-build:
	cd apps/web && npm run build

docker-build:
	docker build -t supaback:latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-minio:
	docker compose -f docker-compose.yml -f docker-compose.minio.yml up -d
