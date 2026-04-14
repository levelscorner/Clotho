.PHONY: dev-backend dev-frontend build test migrate lint clean

# Development
dev-backend:
	go run ./cmd/clotho

dev-frontend:
	cd web && npm run dev

# Build
build: build-frontend build-backend

build-backend:
	go build -o bin/clotho ./cmd/clotho

build-frontend:
	cd web && npm run build

# Test
test:
	go test -race ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Database
migrate:
	go run ./cmd/clotho migrate

# Lint
lint:
	go vet ./...

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Clean
clean:
	rm -rf bin/ web/dist/ coverage.out coverage.html

# Full local stack (Postgres + Kokoro + ComfyUI + backend + frontend) with NO_AUTH bypass.
# Requires Docker daemon for Postgres, and prior setup of Kokoro-FastAPI + ComfyUI in /Users/level/ws/models/.
dev-full:
	./scripts/dev-full.sh up

dev-status:
	./scripts/dev-full.sh status

dev-logs:
	./scripts/dev-full.sh logs

dev-stop:
	./scripts/dev-stop.sh
