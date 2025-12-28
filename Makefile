.PHONY: help install-tools dev-up dev-down dev-up-sso migrate-up migrate-down migrate-create seed run run-sso run-frontend test test-unit test-integration test-e2e test-e2e-ui test-e2e-headed check-node test-frontend test-frontend-coverage test-all lint fmt generate build build-linux build-all build-frontend docker-build docker-push clean clean-all pre-commit security-scan

# Variables
APP_NAME=waffles
DOCKER_IMAGE=ghcr.io/waffles/$(APP_NAME)
DOCKER_TAG?=latest
GO_VERSION=1.22
NODE_MIN_VERSION=18
NPM_MIN_VERSION=9

# SSO/OAuth Configuration (override these for your provider)
# Example: make run-sso SSO_ISSUER=https://auth.example.com SSO_CLIENT_ID=my-client
SSO_ENABLED?=true
SSO_ISSUER?=http://localhost:8180/realms/waffles
SSO_CLIENT_ID?=waffles
SSO_CLIENT_SECRET?=change-me
SSO_BASE_URL?=http://localhost:8080

# Detect local OS and architecture
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
BINARY_NAME=$(APP_NAME)-$(GOOS)-$(GOARCH)

# Colors for output
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_BLUE=\033[34m

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)Waffles - Development Commands$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_GREEN)Setup:$(COLOR_RESET)"
	@echo "  make install-tools    - Install development tools (golangci-lint, migrate, etc.)"
	@echo ""
	@echo "$(COLOR_GREEN)Development:$(COLOR_RESET)"
	@echo "  make dev-up           - Start Docker Compose stack (Gateway, Postgres, Mock)"
	@echo "  make dev-up-sso       - Start Docker Compose stack with Keycloak SSO"
	@echo "  make dev-down         - Stop Docker Compose stack"
	@echo "  make docker-test      - Run E2E tests with Docker Compose"
	@echo "  make docker-logs      - Show logs from all services"
	@echo "  make docker-rebuild   - Rebuild and restart all services"
	@echo "  make run              - Run backend server locally"
	@echo "  make run-sso          - Run backend with SSO enabled (uses SSO_* vars)"
	@echo "  make run-frontend     - Run frontend dev server (Vite)"
	@echo "  make seed             - Seed test data"
	@echo ""
	@echo "$(COLOR_GREEN)Database:$(COLOR_RESET)"
	@echo "  make migrate-up       - Apply database migrations"
	@echo "  make migrate-down     - Rollback last migration"
	@echo "  make migrate-create   - Create new migration (NAME=migration_name)"
	@echo ""
	@echo "$(COLOR_GREEN)Testing:$(COLOR_RESET)"
	@echo "  make test             - Run all backend tests"
	@echo "  make test-unit        - Run unit tests only"
	@echo "  make test-integration - Run Go integration tests (requires running server)"
	@echo "  make test-integration-docker - Run Go integration tests with Docker (self-contained)"
	@echo "  make test-e2e         - Run Playwright E2E tests (UI tests)"
	@echo "  make test-e2e-ui      - Run Playwright E2E tests with interactive UI"
	@echo "  make test-e2e-headed  - Run Playwright E2E tests in headed mode"
	@echo "  make test-frontend    - Run frontend tests (Vue app)"
	@echo "  make test-all         - Run all tests (backend + frontend)"
	@echo ""
	@echo "$(COLOR_GREEN)Code Quality:$(COLOR_RESET)"
	@echo "  make lint             - Run golangci-lint"
	@echo "  make fmt              - Format code with gofmt"
	@echo "  make generate         - Run code generators (sqlc, mockgen)"
	@echo ""
	@echo "$(COLOR_GREEN)Build & Deploy:$(COLOR_RESET)"
	@echo "  make build            - Build backend binary"
	@echo "  make build-frontend   - Build frontend (Vue app)"
	@echo "  make docker-build     - Build Docker image (with frontend)"
	@echo "  make docker-push      - Push Docker image"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make clean-all        - Remove all build artifacts (backend + frontend)"

## install-tools: Install required development tools
install-tools:
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	@echo "Installing golangci-lint..."
	@command -v golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2
	@echo "Installing golang-migrate..."
	@command -v migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Installing sqlc..."
	@command -v sqlc > /dev/null || go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@echo "Installing mockgen..."
	@command -v mockgen > /dev/null || go install github.com/golang/mock/mockgen@latest
	@echo "$(COLOR_GREEN)✓ All tools installed$(COLOR_RESET)"

## dev-up: Start Docker Compose development stack
dev-up:
	@echo "$(COLOR_BLUE)Starting development stack...$(COLOR_RESET)"
	docker-compose up -d
	@echo "$(COLOR_GREEN)✓ Development stack running$(COLOR_RESET)"
	@echo "  Gateway:     localhost:8080"
	@echo "  Postgres:    localhost:5432"
	@echo "  Mock Server: localhost:9001"

## dev-up-sso: Start Docker Compose stack with Keycloak for SSO
dev-up-sso:
	@echo "$(COLOR_BLUE)Starting development stack with SSO (Keycloak)...$(COLOR_RESET)"
	docker-compose --profile sso up -d
	@echo "$(COLOR_GREEN)✓ Development stack with SSO running$(COLOR_RESET)"
	@echo "  Gateway:     localhost:8080"
	@echo "  Keycloak:    localhost:8180 (admin/admin)"
	@echo "  Postgres:    localhost:5432"
	@echo "  Mock Server: localhost:9001"
	@echo ""
	@echo "$(COLOR_BLUE)To run gateway with SSO:$(COLOR_RESET)"
	@echo "  make run-sso SSO_CLIENT_SECRET=<your-client-secret>"

## dev-down: Stop Docker Compose development stack
dev-down:
	@echo "$(COLOR_BLUE)Stopping development stack...$(COLOR_RESET)"
	docker-compose down -v
	@echo "$(COLOR_GREEN)✓ Development stack stopped$(COLOR_RESET)"

## docker-up: Start all services with Docker Compose
docker-up: dev-up

## docker-down: Stop all services and remove volumes
docker-down: dev-down

## docker-logs: Show logs from all services
docker-logs:
	@docker-compose logs -f

## docker-test: Run E2E tests with Docker Compose
docker-test:
	@echo "$(COLOR_BLUE)Running Docker-based E2E tests...$(COLOR_RESET)"
	@chmod +x test-docker.sh
	@./test-docker.sh
	@echo "$(COLOR_GREEN)✓ Docker E2E tests complete$(COLOR_RESET)"

## docker-rebuild: Rebuild and restart all services
docker-rebuild:
	@echo "$(COLOR_BLUE)Rebuilding Docker images...$(COLOR_RESET)"
	docker-compose build --no-cache
	docker-compose up -d
	@echo "$(COLOR_GREEN)✓ Services rebuilt and restarted$(COLOR_RESET)"

## migrate-up: Apply database migrations
migrate-up:
	@echo "$(COLOR_BLUE)Applying database migrations...$(COLOR_RESET)"
	migrate -path internal/database/migrations -database "postgresql://postgres:postgres@localhost:5432/mcp_gateway?sslmode=disable" up
	@echo "$(COLOR_GREEN)✓ Migrations applied$(COLOR_RESET)"

## migrate-down: Rollback last database migration
migrate-down:
	@echo "$(COLOR_BLUE)Rolling back last migration...$(COLOR_RESET)"
	migrate -path internal/database/migrations -database "postgresql://postgres:postgres@localhost:5432/mcp_gateway?sslmode=disable" down 1
	@echo "$(COLOR_GREEN)✓ Migration rolled back$(COLOR_RESET)"

## migrate-create: Create new migration file
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "$(COLOR_BLUE)Creating migration: $(NAME)$(COLOR_RESET)"
	migrate create -ext sql -dir internal/database/migrations -seq $(NAME)
	@echo "$(COLOR_GREEN)✓ Migration created$(COLOR_RESET)"

## seed: Seed database with test data
seed:
	@echo "$(COLOR_BLUE)Seeding database...$(COLOR_RESET)"
	@go run scripts/seed/main.go
	@echo "$(COLOR_GREEN)✓ Database seeded$(COLOR_RESET)"

## run: Run server locally
run:
	@echo "$(COLOR_BLUE)Starting backend server...$(COLOR_RESET)"
	@go run cmd/server/main.go

## run-sso: Run server locally with SSO/OAuth enabled
run-sso:
	@echo "$(COLOR_BLUE)Starting backend server with SSO enabled...$(COLOR_RESET)"
	@echo "SSO Config: $(SSO_ISSUER)"
	AUTH_OAUTH_ENABLED=$(SSO_ENABLED) \
	AUTH_OAUTH_ISSUER=$(SSO_ISSUER) \
	AUTH_OAUTH_CLIENT_ID=$(SSO_CLIENT_ID) \
	AUTH_OAUTH_CLIENT_SECRET=$(SSO_CLIENT_SECRET) \
	AUTH_OAUTH_BASE_URL=$(SSO_BASE_URL) \
	go run cmd/server/main.go

## run-frontend: Run frontend development server
run-frontend: check-node
	@echo "$(COLOR_BLUE)Starting frontend dev server...$(COLOR_RESET)"
	@cd web-app && npm install && npm run dev

## test: Run all tests
test:
	@echo "$(COLOR_BLUE)Running all tests...$(COLOR_RESET)"
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "$(COLOR_GREEN)✓ All tests passed$(COLOR_RESET)"
	@echo "Coverage report: coverage.out"

## test-unit: Run unit tests only
test-unit:
	@echo "$(COLOR_BLUE)Running unit tests...$(COLOR_RESET)"
	@go test -v -race -short ./internal/...
	@echo "$(COLOR_GREEN)✓ Unit tests passed$(COLOR_RESET)"

## test-integration: Run integration tests (Go API tests against running server)
test-integration:
	@echo "$(COLOR_BLUE)Running integration tests...$(COLOR_RESET)"
	@INTEGRATION_TEST=1 API_BASE_URL=$${API_BASE_URL:-http://localhost:8080} go test -v -race ./test/integration/...
	@echo "$(COLOR_GREEN)✓ Integration tests passed$(COLOR_RESET)"

## test-integration-docker: Run integration tests with Docker (start services, test, cleanup)
test-integration-docker:
	@echo "$(COLOR_BLUE)Starting Docker services for integration tests...$(COLOR_RESET)"
	@docker-compose up -d --build
	@echo "Waiting for services to be ready..."
	@sleep 5
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if curl -s http://localhost:8080/health > /dev/null 2>&1; then \
			echo "Gateway is ready!"; \
			break; \
		fi; \
		echo "Waiting for gateway... ($$i/10)"; \
		sleep 2; \
	done
	@echo "$(COLOR_BLUE)Running integration tests...$(COLOR_RESET)"
	@INTEGRATION_TEST=1 API_BASE_URL=http://localhost:8080 go test -v -count=1 ./test/integration/... || (docker-compose down -v && exit 1)
	@echo "$(COLOR_BLUE)Cleaning up Docker services...$(COLOR_RESET)"
	@docker-compose down -v
	@echo "$(COLOR_GREEN)✓ Integration tests passed$(COLOR_RESET)"

## test-e2e: Run E2E tests (Playwright UI tests)
test-e2e:
	@echo "$(COLOR_BLUE)Running E2E tests...$(COLOR_RESET)"
	@cd test/e2e && npm install && npm test
	@echo "$(COLOR_GREEN)✓ E2E tests passed$(COLOR_RESET)"

## test-e2e-ui: Run E2E tests with Playwright UI
test-e2e-ui:
	@echo "$(COLOR_BLUE)Running E2E tests with UI...$(COLOR_RESET)"
	@cd test/e2e && npm install && npm run test:ui

## test-e2e-headed: Run E2E tests in headed mode
test-e2e-headed:
	@echo "$(COLOR_BLUE)Running E2E tests (headed)...$(COLOR_RESET)"
	@cd test/e2e && npm install && npm run test:headed

## lint: Run golangci-lint
lint:
	@echo "$(COLOR_BLUE)Running linters...$(COLOR_RESET)"
	@golangci-lint run --timeout 5m ./...
	@echo "$(COLOR_GREEN)✓ Linting passed$(COLOR_RESET)"

## fmt: Format code
fmt:
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	@gofmt -s -w .
	@go mod tidy
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

## generate: Run code generators
generate:
	@echo "$(COLOR_BLUE)Running code generators...$(COLOR_RESET)"
	@go generate ./...
	@if [ -f sqlc.yaml ]; then sqlc generate; fi
	@echo "$(COLOR_GREEN)✓ Code generated$(COLOR_RESET)"

## build: Build binary for local platform
build:
	@echo "$(COLOR_BLUE)Building binary for $(GOOS)/$(GOARCH)...$(COLOR_RESET)"
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-w -s" -o bin/$(BINARY_NAME) cmd/server/main.go
	@ln -sf $(BINARY_NAME) bin/$(APP_NAME)
	@echo "$(COLOR_GREEN)✓ Binary built: bin/$(BINARY_NAME)$(COLOR_RESET)"
	@echo "  Symlink created: bin/$(APP_NAME) -> $(BINARY_NAME)"

## build-linux: Build binary for Linux (for Docker)
build-linux:
	@echo "$(COLOR_BLUE)Building binary for linux/amd64...$(COLOR_RESET)"
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(APP_NAME)-linux-amd64 cmd/server/main.go
	@echo "$(COLOR_GREEN)✓ Binary built: bin/$(APP_NAME)-linux-amd64$(COLOR_RESET)"

## build-all: Build binaries for all platforms
build-all:
	@echo "$(COLOR_BLUE)Building binaries for all platforms...$(COLOR_RESET)"
	@mkdir -p bin
	@echo "Building for darwin/amd64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(APP_NAME)-darwin-amd64 cmd/server/main.go
	@echo "Building for darwin/arm64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o bin/$(APP_NAME)-darwin-arm64 cmd/server/main.go
	@echo "Building for linux/amd64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(APP_NAME)-linux-amd64 cmd/server/main.go
	@echo "Building for linux/arm64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o bin/$(APP_NAME)-linux-arm64 cmd/server/main.go
	@echo "$(COLOR_GREEN)✓ All binaries built$(COLOR_RESET)"
	@ls -lh bin/

## docker-build: Build Docker image
docker-build:
	@echo "$(COLOR_BLUE)Building Docker image...$(COLOR_RESET)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f Dockerfile .
	@echo "$(COLOR_GREEN)✓ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(COLOR_RESET)"

## docker-push: Push Docker image to registry
docker-push:
	@echo "$(COLOR_BLUE)Pushing Docker image...$(COLOR_RESET)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(COLOR_GREEN)✓ Docker image pushed$(COLOR_RESET)"

## check-node: Validate Node.js and npm versions
check-node:
	@echo "$(COLOR_BLUE)Checking Node.js and npm versions...$(COLOR_RESET)"
	@command -v node > /dev/null || (echo "Error: Node.js is not installed. Please install Node.js >= $(NODE_MIN_VERSION)" && exit 1)
	@command -v npm > /dev/null || (echo "Error: npm is not installed. Please install npm >= $(NPM_MIN_VERSION)" && exit 1)
	@NODE_VERSION=$$(node -v | sed 's/v//' | cut -d. -f1); \
	if [ "$$NODE_VERSION" -lt "$(NODE_MIN_VERSION)" ]; then \
		echo "Error: Node.js version $$NODE_VERSION is too old. Required: >= $(NODE_MIN_VERSION)"; \
		exit 1; \
	fi
	@NPM_VERSION=$$(npm -v | cut -d. -f1); \
	if [ "$$NPM_VERSION" -lt "$(NPM_MIN_VERSION)" ]; then \
		echo "Error: npm version $$NPM_VERSION is too old. Required: >= $(NPM_MIN_VERSION)"; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)✓ Node.js $$(node -v) and npm $$(npm -v) OK$(COLOR_RESET)"

## test-frontend: Run frontend unit tests with vitest
test-frontend: check-node
	@echo "$(COLOR_BLUE)Running frontend tests...$(COLOR_RESET)"
	@cd web-app && npm install && npm test
	@echo "$(COLOR_GREEN)✓ Frontend tests passed$(COLOR_RESET)"

## test-frontend-coverage: Run frontend tests with coverage report
test-frontend-coverage: check-node
	@echo "$(COLOR_BLUE)Running frontend tests with coverage...$(COLOR_RESET)"
	@cd web-app && npm install && npm run test:coverage
	@echo "$(COLOR_GREEN)✓ Frontend tests passed with coverage$(COLOR_RESET)"

## test-all: Run all tests (backend + frontend)
test-all: test test-frontend
	@echo "$(COLOR_GREEN)✓ All tests passed (backend + frontend)$(COLOR_RESET)"

## build-frontend: Build Vue.js frontend
build-frontend: check-node
	@echo "$(COLOR_BLUE)Building Vue.js frontend...$(COLOR_RESET)"
	@cd web-app && npm install && npm run build
	@echo "$(COLOR_GREEN)✓ Frontend built in web-app/dist/$(COLOR_RESET)"

## clean: Remove build artifacts
clean:
	@echo "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf bin/
	@rm -f coverage.out
	@go clean
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

## clean-all: Remove all build artifacts (backend + frontend)
clean-all: clean
	@echo "$(COLOR_BLUE)Cleaning frontend artifacts...$(COLOR_RESET)"
	@rm -rf web-app/dist/
	@rm -rf web-app/node_modules/
	@echo "$(COLOR_GREEN)✓ All artifacts cleaned$(COLOR_RESET)"

## pre-commit: Install and run pre-commit hooks
pre-commit:
	@echo "$(COLOR_BLUE)Setting up pre-commit hooks...$(COLOR_RESET)"
	@command -v pre-commit > /dev/null || pip3 install pre-commit
	@pre-commit install
	@echo "$(COLOR_GREEN)✓ Pre-commit hooks installed$(COLOR_RESET)"
	@echo "Run 'pre-commit run --all-files' to run all checks"

## security-scan: Run security scanning tools
security-scan:
	@echo "$(COLOR_BLUE)Running security scans...$(COLOR_RESET)"
	@echo "Scanning for secrets with gitleaks..."
	@command -v gitleaks > /dev/null || (echo "Error: gitleaks not found. Install: https://github.com/gitleaks/gitleaks#installing" && exit 1)
	@gitleaks detect --source . --config .gitleaks.toml -v || true
	@echo ""
	@echo "Scanning Go code with gosec..."
	@command -v gosec > /dev/null || (echo "Error: gosec not found. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest" && exit 1)
	@gosec -exclude-dir=test -exclude-dir=web -exclude-dir=node_modules ./... || true
	@echo ""
	@echo "$(COLOR_GREEN)✓ Security scan complete$(COLOR_RESET)"
