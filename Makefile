.PHONY: build run test test-coverage clean deps lint fmt vet \
	migrate-up migrate-down migrate-status migrate-create \
	sqlc-generate swagger generate-keys \
	docker-dev-up docker-dev-down docker-dev-logs docker-dev-build \
	docker-prod-up docker-prod-down docker-prod-logs docker-prod-build \
	redis-cli psql air dev local all \
	mocks test-unit test-integration

BINARY_NAME=api
MAIN_PATH=./cmd/api

# ==================== Building ====================

build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run:
	go run $(MAIN_PATH)

# ==================== Testing ====================

test:
	go test -v -race -coverprofile=coverage.out ./...

test-unit: ## Run unit tests only (short tests without integration)
	go test -v -short -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests (requires Docker)
	go test -v -run Integration -race -coverprofile=coverage.out ./...

test-coverage: test
	go tool cover -html=coverage.out

# ==================== Code Quality ====================

clean:
	rm -rf bin/
	rm -rf tmp/
	rm -f coverage.out

deps:
	go mod download
	go mod tidy

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

# ==================== Database Migrations ====================

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir migrations postgres "$(DATABASE_URL)" status

migrate-create:
	@read -p "Enter migration name: " name; \
	goose -dir migrations create $$name sql

# ==================== Code Generation ====================

generate-keys:
	@./scripts/generate_keys.sh keys
	@echo "Generating refresh key pair..."
	@./scripts/generate_keys.sh keys/refresh
	@mv keys/refresh/private.pem keys/refresh_private.pem
	@mv keys/refresh/public.pem keys/refresh_public.pem
	@rmdir keys/refresh
	@mv keys/private.pem keys/access_private.pem
	@mv keys/public.pem keys/access_public.pem

sqlc-generate:
	sqlc generate

swagger:
	swag init -g cmd/api/main.go -o ./docs --parseInternal --parseDependency --parseDepth 2

mocks: ## Generate mocks for testing
	go generate ./internal/service/...
	go generate ./internal/repository/...

# ==================== Development (with Air locally) ====================

air:
	air -c .air.toml

local: ## Run only postgres and redis for local development with air
	docker compose -f docker-compose-dev.yml up -d postgres redis
	@echo "Database and Redis started. Run 'make air' to start the app with hot reload."

local-down: ## Stop local development services
	docker compose -f docker-compose-dev.yml down

local-logs: ## View local development logs
	docker compose -f docker-compose-dev.yml logs -f

# ==================== Development (Docker with Air) ====================

docker-dev-up: ## Start all development services (with Air hot reload)
	docker compose -f docker-compose-dev.yml up -d

docker-dev-down: ## Stop development services
	docker compose -f docker-compose-dev.yml down

docker-dev-logs: ## View development logs
	docker compose -f docker-compose-dev.yml logs -f app

docker-dev-build: ## Build development images
	docker compose -f docker-compose-dev.yml build

docker-dev-restart: ## Restart development app
	docker compose -f docker-compose-dev.yml restart app

# ==================== Production ====================

docker-prod-up: ## Start production services
	docker compose -f docker-compose.yml up -d

docker-prod-down: ## Stop production services
	docker compose -f docker-compose.yml down

docker-prod-logs: ## View production logs
	docker compose -f docker-compose.yml logs -f

docker-prod-build: ## Build production images
	docker compose -f docker-compose.yml build

docker-prod-restart: ## Restart production services
	docker compose -f docker-compose.yml restart

# Legacy aliases for backward compatibility
docker-up: docker-prod-up
docker-down: docker-prod-down
docker-logs: docker-prod-logs
docker-build: docker-prod-build
docker-restart: docker-prod-restart

# ==================== Database Utilities ====================

redis-cli:
	docker compose -f docker-compose-dev.yml exec redis redis-cli

psql:
	docker compose -f docker-compose-dev.yml exec postgres psql -U postgres -d app

# ==================== Convenience ====================

dev: ## Quick start: start services and run app locally
	docker compose -f docker-compose-dev.yml up -d postgres redis
	@sleep 3
	go run $(MAIN_PATH)

all: deps sqlc-generate swagger build

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
