# Waffles

[![CI](https://github.com/arunselvarajdb/waffles/actions/workflows/ci.yml/badge.svg)](https://github.com/arunselvarajdb/waffles/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/arunselvarajdb/waffles)](https://goreportcard.com/report/github.com/arunselvarajdb/waffles)
[![codecov](https://codecov.io/gh/arunselvarajdb/waffles/branch/main/graph/badge.svg)](https://app.codecov.io/github/arunselvarajdb/waffles)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Production-ready MCP (Model Context Protocol) gateway platform built with Go, featuring comprehensive security, observability, and DevOps best practices.

## Overview

Waffles is a centralized proxy and management platform for MCP servers, providing:

- **Centralized Gateway**: Proxy MCP protocol requests with circuit breakers, retries, and connection pooling
- **Server Registry**: Catalog and discovery of MCP servers with health monitoring
- **Security**: JWT/API key authentication, OAuth 2.0, LDAP/AD integration, and RBAC
- **Audit & Analytics**: Complete request logging and usage analytics
- **Production-Ready**: Full CI/CD, monitoring, and deployment automation

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   MCP Clients                           │
│     (Claude Desktop, VSCode, Custom Applications)      │
└────────────────┬────────────────────────────────────────┘
                 │ HTTP/SSE (MCP Protocol)
                 │
┌────────────────▼────────────────────────────────────────┐
│                 MCP Gateway API (Go)                    │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Middleware: Auth, CORS, Rate Limit, Logging    │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Services: Gateway, Registry, Auth, Audit       │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────┬────────────────────────────────────────┘
                 │
        ┌────────┴──────────┐
        │                   │
┌───────▼──────┐   ┌────────▼──────┐
│  PostgreSQL  │   │     Redis     │
│  (Metadata)  │   │  (Cache/Rate) │
└──────────────┘   └───────────────┘
                 │
        ┌────────┴────────┐
        │                 │
┌───────▼──────┐   ┌──────▼────────┐
│ MCP Server 1 │   │ MCP Server N  │
└──────────────┘   └───────────────┘
```

## Features

### Core Gateway
- ✅ **MCP Protocol Support**: Initialize, tools/call, tools/list, resources, prompts
- ✅ **High Availability**: Connection pooling, circuit breakers, automatic retries
- ✅ **Request Routing**: Intelligent routing to registered MCP servers
- ✅ **Health Monitoring**: Automated health checks for all registered servers

### Security
- ✅ **Authentication**: JWT tokens, API keys, OAuth 2.0 (planned), LDAP/AD (planned)
- ✅ **Authorization**: Role-Based Access Control (RBAC) with 4 built-in roles
- ✅ **Rate Limiting**: Redis-based sliding window (planned)
- ✅ **Audit Logging**: Complete request/response audit trail

### Observability
- ✅ **Structured Logging**: JSON logging with Zerolog (request ID, user ID tracking)
- ✅ **Metrics**: Prometheus-compatible metrics (planned)
- ✅ **Health Checks**: `/health` and `/ready` endpoints
- ✅ **Distributed Tracing**: OpenTelemetry support (planned)

### DevOps
- ✅ **CI/CD**: GitHub Actions (lint, security scan, test, multi-platform build)
- ✅ **Containerization**: Multi-stage Docker builds with distroless base
- ✅ **Orchestration**: Kubernetes Helm charts (planned)
- ✅ **Database Migrations**: Automated migrations with golang-migrate

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make (optional, for convenience commands)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/waffles/waffles.git
   cd waffles
   ```

2. **Start all services with Docker Compose** (Recommended)
   ```bash
   # Start PostgreSQL, Gateway, and Mock MCP Server
   make dev-up

   # Services will be available at:
   # - Gateway API: http://localhost:8080
   # - PostgreSQL: localhost:5432
   # - Mock MCP Server: http://localhost:9001
   ```

3. **Verify it's running**
   ```bash
   curl http://localhost:8080/health
   # Expected: {"service":"waffles","status":"healthy"}

   curl http://localhost:8080/ready
   # Expected: {"status":"ready","checks":{"database":{"healthy":true...}}}
   ```

4. **View logs**
   ```bash
   make docker-logs
   ```

5. **Stop all services**
   ```bash
   make dev-down
   ```

### Local Development (Without Docker)

If you prefer to run the gateway locally:

```bash
# 1. Start PostgreSQL
docker run -d \
  --name waffles-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=mcp_gateway \
  -p 5432:5432 \
  postgres:15-alpine

# 2. Run the server
go run cmd/server/main.go

# 3. Server will start on http://localhost:8080
```

### Configuration

Configuration can be provided via:
1. **Config file**: `configs/config.yaml` (default) or `-config path/to/config.yaml`
2. **Environment variables**: Override any config value (e.g., `SERVER_PORT=9000`)

Example config:
```yaml
server:
  port: 8080
  environment: development

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: mcp_gateway
  max_connections: 25

logging:
  level: info
  format: json
```

See [`.env.example`](.env.example) for all available environment variables.

## API Endpoints

### Health & Status
```
GET  /health              # Health check (database connectivity)
GET  /ready               # Readiness probe
GET  /api/v1/status       # API status and version
```

### MCP Server Registry ✅
```
GET    /api/v1/servers              # List all servers (with filtering)
POST   /api/v1/servers              # Register new MCP server
GET    /api/v1/servers/:id          # Get server details
PUT    /api/v1/servers/:id          # Update server
DELETE /api/v1/servers/:id          # Delete server
PATCH  /api/v1/servers/:id/toggle   # Enable/disable server

GET    /api/v1/servers/:id/health   # Get latest health status
POST   /api/v1/servers/:id/health   # Trigger immediate health check
```

### Gateway Proxy ✅
```
POST /api/v1/gateway/:server_id/initialize       # Initialize MCP connection
POST /api/v1/gateway/:server_id/tools/list       # List available tools
POST /api/v1/gateway/:server_id/tools/call       # Execute tool
GET  /api/v1/gateway/:server_id/resources/list   # List resources
GET  /api/v1/gateway/:server_id/resources/read   # Read resource
POST /api/v1/gateway/:server_id/prompts/list     # List prompts
POST /api/v1/gateway/:server_id/prompts/get      # Get prompt
```

### Authentication (Planned - Phase 3)
```
POST /api/v1/auth/register       # Register new user
POST /api/v1/auth/login          # Login (email/password)
POST /api/v1/auth/refresh        # Refresh JWT token
GET  /api/v1/auth/me             # Get current user info
```

See [API Documentation](docs/api/openapi.yaml) for full OpenAPI 3.0 specification (coming soon).

## Development

### Available Make Commands

```bash
# Development
make run              # Run server locally
make dev-up           # Start Docker Compose (PostgreSQL, Gateway, Mock MCP)
make dev-down         # Stop Docker Compose and remove volumes
make docker-logs      # Show logs from all services
make docker-rebuild   # Rebuild and restart all services

# Testing
make test             # Run all tests with race detection
make test-unit        # Run unit tests only
make test-integration # Run integration tests
make test-e2e         # Run E2E tests
make docker-test      # Run E2E tests with Docker Compose
make lint             # Run golangci-lint
make fmt              # Format code with gofmt

# Building
make build            # Build for current platform (auto-detect)
make build-all        # Cross-compile for all platforms
make build-linux      # Build for Linux (Docker/K8s)

# Database
make migrate-up       # Apply pending migrations
make migrate-down     # Rollback last migration
make migrate-create   # Create new migration (NAME=migration_name)

# Docker
make docker-build     # Build Docker image
make docker-push      # Push to registry (coming soon)

# Cleanup
make clean            # Remove build artifacts
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./internal/config/...
go test -v ./pkg/logger/...

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

See [TEST_GUIDE.md](test/TEST_GUIDE.md) for comprehensive testing guide.

### Project Structure

```
waffles/
├── cmd/
│   ├── server/          # API server entry point
│   └── migrate/         # Database migration tool
│
├── internal/
│   ├── config/          # Configuration loading and validation
│   ├── database/        # Database connection and migrations
│   ├── handler/         # HTTP handlers and middleware
│   ├── server/          # HTTP server setup and routing
│   ├── domain/          # Domain entities (planned)
│   ├── repository/      # Data access layer (planned)
│   └── service/         # Business logic (planned)
│
├── pkg/
│   ├── logger/          # Structured logging interface
│   ├── mcp/             # MCP protocol client (planned)
│   └── errors/          # Error types and handling (planned)
│
├── deployments/
│   ├── docker/          # Dockerfile and docker-compose
│   └── k8s/             # Kubernetes manifests (planned)
│
├── test/
│   ├── fixtures/        # Test data fixtures
│   ├── testutil/        # Test utilities
│   └── integration/     # Integration tests (planned)
│
└── configs/             # Configuration files
```

Following [Go Standard Project Layout](https://github.com/golang-standards/project-layout).

## Tech Stack

### Backend
- **Language**: Go 1.22+
- **HTTP Framework**: [Gin](https://github.com/gin-gonic/gin) (high-performance, feature-rich)
- **Database**: PostgreSQL 15+ with [pgx](https://github.com/jackc/pgx) (fastest Go driver)
- **Database Layer**: [sqlc](https://sqlc.dev/) (type-safe SQL, zero reflection) - planned
- **Cache**: Redis 7+ with [go-redis](https://github.com/redis/go-redis)
- **Config**: [Viper](https://github.com/spf13/viper) (12-factor app config)
- **Logging**: [Zerolog](https://github.com/rs/zerolog) (structured JSON, zero allocation)
- **Migrations**: [golang-migrate](https://github.com/golang-migrate/migrate)

### Security
- **JWT**: [golang-jwt](https://github.com/golang-jwt/jwt) (planned)
- **OAuth 2.0**: Integration planned
- **LDAP/AD**: Integration planned
- **Secrets**: AWS Secrets Manager (production) / env vars (local dev) - planned

### DevOps
- **CI/CD**: GitHub Actions (lint, security scan, test, build)
- **Container**: Docker multi-stage builds
- **Orchestration**: Kubernetes + Helm (planned)
- **Monitoring**: Prometheus + Grafana (planned)
- **Linting**: [golangci-lint](https://golangci-lint.run/) (40+ linters)

## Database Schema

The platform uses PostgreSQL with the following core tables:

- **users**: User accounts (email, password hash, auth provider)
- **api_keys**: User API keys for programmatic access
- **roles**: RBAC roles (admin, operator, user, readonly)
- **permissions**: Fine-grained permissions (resource + action)
- **mcp_servers**: Registered MCP server catalog
- **server_health**: Health check results and metrics
- **audit_logs**: Complete request/response audit trail
- **usage_stats**: Usage analytics (aggregated)

Built-in roles:
- **admin**: Full system access
- **operator**: Manage servers, view logs
- **user**: Use gateway, discover servers
- **readonly**: View-only access

See [migrations](internal/database/migrations/) for full schema.

## Security

### Authentication Methods
- ✅ JWT tokens (short-lived access + refresh tokens) - planned
- ✅ API keys (SHA-256 hashed, user-scoped) - planned
- 🔄 OAuth 2.0 (Google, GitHub, Microsoft) - planned
- 🔄 LDAP/Active Directory - planned

### Authorization
- ✅ Role-Based Access Control (RBAC)
- ✅ Server-level permissions
- ✅ Fine-grained resource/action permissions

### Security Hardening
- ✅ Rate limiting (Redis-based sliding window) - planned
- ✅ Input validation on all endpoints
- ✅ SQL injection prevention (parameterized queries)
- ✅ TLS 1.3 enforcement - planned
- ✅ CORS configuration
- ✅ Security headers (CSP, HSTS, etc.) - planned

Security scanning:
- **gosec**: Go security checker (runs in CI)
- **Trivy**: Vulnerability scanner for dependencies (runs in CI)
- **golangci-lint**: Includes security-focused linters

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines (coming soon).

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Run linter (`make lint`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

All PRs must:
- ✅ Pass all CI checks (lint, security, test, build)
- ✅ Include tests for new functionality
- ✅ Maintain or improve code coverage (target: 80%+)
- ✅ Follow Go best practices and project conventions

## Roadmap

### ✅ Phase 1: Foundation (Completed)
- [x] Project structure and tooling
- [x] Configuration management
- [x] Database setup with migrations
- [x] Structured logging
- [x] HTTP server with Gin
- [x] Middleware stack (CORS, logging, recovery, request ID, timeout)
- [x] CI/CD pipeline (GitHub Actions)

### ✅ Phase 2: Core Services (Completed)
- [x] Registry service (server CRUD)
- [x] Gateway proxy (MCP protocol)
- [x] Health monitoring (automated checks)
- [x] Audit logging (basic request/response tracking)
- [x] Unit tests for all services
- [x] Docker Compose setup for easy deployment
- [x] Makefile commands for development workflow

### 📋 Phase 3: Advanced Features (Planned)
- [ ] OAuth 2.0 integration
- [ ] LDAP/AD authentication
- [ ] AWS Secrets Manager integration
- [ ] Advanced RBAC
- [ ] Rate limiting
- [ ] Circuit breakers and retries

### 📋 Phase 4: Observability (Planned)
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Alert rules

### 📋 Phase 5: Production Readiness (Planned)
- [ ] Kubernetes Helm charts
- [ ] Production deployment guides
- [ ] Load testing
- [ ] Security audit
- [ ] Performance optimization

### 📋 Phase 6: Advanced Features (Future)
- [ ] Admin UI (web dashboard)
- [ ] Multi-tenancy support
- [ ] Advanced analytics
- [ ] Plugin system
- [ ] Custom MCP server templates

## Performance

**Current targets** (to be benchmarked):
- P95 latency: < 500ms
- Throughput: 1000+ RPS per instance
- Database connections: Configurable pooling (default: 25 max)
- Horizontal scaling: Stateless design for unlimited scale

Load testing with [k6](https://k6.io/) planned.

## Monitoring

### Health Endpoints

```bash
# Basic health check (returns 200 if database is reachable)
curl http://localhost:8080/health

# Readiness probe (for K8s)
curl http://localhost:8080/ready

# Detailed status
curl http://localhost:8080/api/v1/status
```

### Logs

All logs are structured JSON with Zerolog:

```json
{
  "level": "info",
  "time": "2025-12-10T17:22:48-08:00",
  "message": "Starting MCP Gateway",
  "version": "dev",
  "build_time": "unknown",
  "environment": "development"
}
```

Logs include:
- Request ID tracking (for request correlation)
- User ID (when authenticated)
- Latency measurements
- Error stack traces

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built following Go best practices from the community
- Thanks to all contributors and the Go ecosystem

## Support

- **Issues**: [GitHub Issues](https://github.com/waffles/waffles/issues)
- **Discussions**: [GitHub Discussions](https://github.com/waffles/waffles/discussions)
- **Documentation**: [docs/](docs/)

## Status

✅ **Phase 2 Complete**: Core MCP gateway functionality is implemented and ready for use.

Current version: **v0.2.0-beta** (Core Services Complete)

**What's Working:**
- ✅ Server Registry (CRUD operations)
- ✅ Gateway Proxy (MCP protocol forwarding)
- ✅ Health Monitoring (automated checks)
- ✅ Audit Logging (request/response tracking)
- ✅ Docker Compose deployment
- ✅ PostgreSQL integration

**Coming in Phase 3:**
- 🔄 Authentication (JWT, OAuth, LDAP)
- 🔄 Authorization (RBAC)
- 🔄 Rate Limiting
- 🔄 Advanced observability (Prometheus, Grafana)

---

**Built with ❤️ using Go and best practices**
