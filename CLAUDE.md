# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MCP Gateway is a production-ready Model Context Protocol (MCP) gateway platform built with Go. It provides centralized proxy management for MCP servers with features including server registry, health monitoring, audit logging, and Prometheus metrics.

## Common Commands

### Development
```bash
make dev-up           # Start Docker Compose stack (Gateway, PostgreSQL, Mock MCP Server)
make dev-down         # Stop Docker Compose stack and remove volumes
make run              # Run server locally (requires PostgreSQL on localhost:5432)
make docker-logs      # Tail logs from all services
make docker-rebuild   # Rebuild and restart all services
```

### Testing
```bash
make test             # Run all tests with race detection and coverage
make test-unit        # Run unit tests only (./internal/...)
make test-e2e         # Run E2E tests (requires running services)
go test -v ./internal/config/...  # Run tests for a specific package
go test -v -run TestName ./...    # Run a single test by name
```

### Code Quality
```bash
make lint             # Run golangci-lint (40+ linters enabled)
make fmt              # Format code with gofmt and run go mod tidy
```

### Building
```bash
make build            # Build binary for current platform
make build-linux      # Build for Linux (Docker/K8s)
make docker-build     # Build Docker image
```

### Database
```bash
make migrate-up                       # Apply pending migrations
make migrate-down                     # Rollback last migration
make migrate-create NAME=add_users    # Create new migration files
```

### Frontend (Vue.js)
```bash
cd web-app && npm run dev     # Start Vite dev server
cd web-app && npm run build   # Build production bundle
make test-frontend            # Run frontend tests
```

### OAuth/SSO with Keycloak (Optional)
```bash
cd examples/oauth/keycloak && docker-compose up -d   # Start Keycloak
cd examples/oauth/keycloak && ./configure-dcr.sh     # Enable DCR for MCP clients (first time only)
```
See `docs/MCP_CLIENT_AUTH.md` for full OAuth setup instructions.

## Architecture

### Backend Structure

The codebase follows the Go standard project layout:

- **cmd/server/main.go**: Application entry point. Initializes config, database, metrics, and HTTP server. Embeds the Vue.js frontend via `//go:embed all:dist`.

- **internal/server/**: HTTP server and routing setup using Gin framework. Routes are configured in `router.go` with middleware chain: Recovery → Metrics → RequestID → Logger → CORS → Timeout.

- **internal/handler/**: HTTP handlers organized by domain (health, registry, gateway). Gateway handlers use audit middleware for request/response logging.

- **internal/service/**: Business logic layer with three services:
  - `registry`: MCP server CRUD operations and health checks
  - `gateway`: MCP protocol proxying (initialize, tools, resources, prompts)
  - `audit`: Request/response audit trail logging

- **internal/repository/**: Data access layer using pgx for PostgreSQL.

- **internal/domain/**: Domain entities (Server, User, APIKey, Audit).

- **internal/metrics/**: Prometheus metrics registry and collectors (HTTP requests, database stats).

- **pkg/logger/**: Zerolog-based structured logging with context support.

- **pkg/mcp/**: MCP protocol types and error definitions.

### Request Flow

1. Client → Gin Router → Middleware Chain → Handler
2. Handler → Service → Repository → PostgreSQL
3. Gateway requests are proxied: Handler → Gateway Service → External MCP Server

### Key API Routes

- `GET /health`, `GET /ready`: Health check endpoints
- `GET/POST/PUT/DELETE /api/v1/servers/*`: Server registry CRUD
- `POST /api/v1/gateway/:server_id/*`: MCP protocol proxy (initialize, tools/call, tools/list, resources/*, prompts/*)

### Frontend

Vue 3 + Vite + Tailwind CSS SPA embedded in the Go binary. The server serves the SPA for non-API routes with fallback to index.html for client-side routing.

## Configuration

Config is loaded from `configs/config.yaml` with environment variable overrides. Key sections:
- `server`: port, timeouts, environment
- `database`: PostgreSQL connection settings
- `metrics`: Prometheus settings (port 9090)
- `logging`: level and format (json/console)

## Import Ordering

Imports follow gci configuration: standard library → third-party → project packages (github.com/waffles/mcp-gateway).

## Go Best Practices

### Error Handling
- Always wrap errors with context using `fmt.Errorf("operation failed: %w", err)`
- Use domain-specific error types from `internal/domain/errors.go` for business logic errors
- Return errors rather than logging and continuing; let the caller decide how to handle
- Check errors immediately after the call that might produce them

### Naming Conventions
- Use MixedCaps (not underscores) for multi-word names
- Acronyms should be all caps: `serverID`, `httpClient`, `mcpServer`
- Interface names should describe behavior: `Reader`, `Handler`, `Service`
- Avoid stuttering: `registry.Service` not `registry.RegistryService`

### Concurrency
- Pass `context.Context` as the first parameter to functions that do I/O or may block
- Use `defer` for cleanup (closing resources, unlocking mutexes)
- Prefer channels for coordination, mutexes for state protection

### Code Organization
- Keep functions focused and short (target <30 lines)
- Business logic belongs in `internal/service/`, not handlers
- Handlers should only parse requests, call services, and format responses
- Use dependency injection; pass interfaces, not concrete types

### Testing
- Table-driven tests for multiple cases
- Use `testify/assert` and `testify/require` (already in project)
- Name test functions: `TestFunctionName_Scenario_ExpectedBehavior`
- Use `t.Parallel()` for independent tests

## Vue.js / JavaScript Best Practices

### Component Structure
- Use Composition API with `<script setup>` syntax
- Keep components small and focused on a single responsibility
- Extract reusable logic into composables (`src/composables/`)

### State Management
- Use Pinia stores for global state
- Keep component-local state in `ref()` or `reactive()`
- Avoid prop drilling; use provide/inject or stores for deeply nested data

### TypeScript
- Define interfaces for API response types
- Use strict null checks; handle undefined/null explicitly
- Avoid `any`; use `unknown` and narrow the type

### API Calls
- Use Axios (already configured) for HTTP requests
- Handle loading and error states in components
- Create API service modules to encapsulate endpoint calls

### Naming Conventions
- Components: PascalCase (`ServerList.vue`, `HealthStatus.vue`)
- Composables: camelCase with `use` prefix (`useServers`, `useAuth`)
- Props/events: camelCase in script, kebab-case in template
- Constants: SCREAMING_SNAKE_CASE

### Performance
- Use `v-show` for frequently toggled elements, `v-if` for conditional rendering
- Add `key` attributes to `v-for` lists
- Lazy-load routes and heavy components
