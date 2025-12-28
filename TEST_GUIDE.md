# How to Test the Waffles Setup

## Quick Start

### Run All Tests
```bash
# Using Makefile (recommended)
make test

# Or directly with go
go test ./...

# With verbose output
go test -v ./...

# With race detection
go test -race ./...
```

## Detailed Testing

### 1. Test Individual Packages

**Test Config Package:**
```bash
go test -v ./internal/config/...
```
Expected output: 6 tests passing, 77.3% coverage

**Test Logger Package:**
```bash
go test -v ./pkg/logger/...
```
Expected output: 8 tests passing, 65.1% coverage

### 2. Test with Coverage

**Generate coverage report:**
```bash
# Generate coverage file
go test -coverprofile=coverage.out ./...

# View coverage summary
go tool cover -func=coverage.out

# Open HTML coverage report in browser
go tool cover -html=coverage.out
```

**What to look for:**
- Green = covered by tests
- Red = not covered
- Gray = not executable

### 3. Test Configuration Loading

**Test with default config:**
```bash
# This uses configs/config.yaml
go run cmd/server/main.go
```
Expected: Server starts, logs show config loaded

**Test with custom config:**
```bash
# Create test config
cat > /tmp/test-config.yaml << 'YAML'
server:
  port: 9999
  environment: development
database:
  host: localhost
  port: 5432
logging:
  level: debug
  format: json
YAML

# Run with custom config
go run cmd/server/main.go -config /tmp/test-config.yaml
```
Expected: Server uses port 9999

**Test with environment variables:**
```bash
# Override with env vars
SERVER_PORT=7777 \
DATABASE_HOST=testdb \
LOGGING_LEVEL=error \
go run cmd/server/main.go
```
Expected: Server uses port 7777, error log level

### 4. Test Logger Output

**Create test script:**
```bash
cat > test-logger.go << 'GO'
package main

import (
    "context"
    "github.com/waffles/waffles/pkg/logger"
)

func main() {
    // Create logger
    log := logger.NewZerolog(logger.Config{
        Level:  logger.InfoLevel,
        Format: "json",
    })

    // Test basic logging
    log.Info().Str("test", "hello").Msg("Basic log message")

    // Test with context
    ctx := context.Background()
    ctx = logger.WithRequestID(ctx, "req-123")
    ctx = logger.WithUserID(ctx, "user-456")

    log.WithContext(ctx).Info().Msg("Log with context")

    // Test different levels
    log.Debug().Msg("Debug message (should not appear)")
    log.Warn().Msg("Warning message")
    log.Error().Msg("Error message")
}
GO

go run test-logger.go
```

Expected output (JSON):
```json
{"level":"info","test":"hello","time":"...","message":"Basic log message"}
{"level":"info","request_id":"req-123","user_id":"user-456","time":"...","message":"Log with context"}
{"level":"warn","time":"...","message":"Warning message"}
{"level":"error","time":"...","message":"Error message"}
```

### 5. Test Building

**Build for current platform:**
```bash
make build
./bin/waffles -h
```
Expected: Binary runs, shows help

**Build for all platforms:**
```bash
make build-all
ls -lh bin/
```
Expected: 4 binaries created (darwin/linux, amd64/arm64)

**Build for Docker:**
```bash
make build-linux
ls -lh bin/waffles-linux-amd64
```

### 6. Test Code Quality

**Run linter:**
```bash
make lint
```
Expected: No errors (some warnings OK for now)

**Format code:**
```bash
make fmt
```

**Check formatting:**
```bash
gofmt -l .
```
Expected: No output (all files formatted)

### 7. Test Docker Setup

**Start dependencies:**
```bash
make dev-up
```
Expected: Postgres and Redis containers running

**Check containers:**
```bash
docker ps
```
Expected output:
```
CONTAINER ID   IMAGE                  STATUS         PORTS
xxx            postgres:15-alpine     Up             0.0.0.0:5432->5432/tcp
xxx            redis:7-alpine         Up             0.0.0.0:6379->6379/tcp
```

**Test Postgres connection:**
```bash
docker exec -it waffles-postgres psql -U postgres -c "SELECT version();"
```

**Test Redis connection:**
```bash
docker exec -it waffles-redis redis-cli ping
```
Expected: PONG

**Stop containers:**
```bash
make dev-down
```

### 8. Integration Test (Manual)

**Full workflow test:**
```bash
# 1. Start dependencies
make dev-up

# 2. Run tests
make test

# 3. Build application
make build

# 4. Run application
./bin/waffles

# Expected output:
# {"level":"info","version":"dev","environment":"development","message":"Starting MCP Gateway"}
# {"level":"info","host":"localhost","port":8080,"message":"MCP Gateway is ready"}

# 5. Stop with Ctrl+C
# Expected: Graceful shutdown message

# 6. Cleanup
make dev-down
```

## Troubleshooting

### Tests Fail

**Check Go version:**
```bash
go version
# Should be 1.22 or higher
```

**Clean and retry:**
```bash
go clean -testcache
make test
```

### Build Fails

**Update dependencies:**
```bash
go mod tidy
go mod download
```

**Clean build cache:**
```bash
go clean -cache
make build
```

### Docker Issues

**Check Docker running:**
```bash
docker ps
```

**View container logs:**
```bash
docker logs waffles-postgres
docker logs waffles-redis
```

**Reset containers:**
```bash
make dev-down
docker system prune -f
make dev-up
```

## Test Checklist

Before committing code, verify:

- [ ] All tests pass: `make test`
- [ ] No race conditions: `go test -race ./...`
- [ ] Code formatted: `make fmt`
- [ ] Linter passes: `make lint`
- [ ] Build succeeds: `make build`
- [ ] Coverage > 50%: `go test -cover ./...`
- [ ] Docker containers start: `make dev-up`

## Performance Testing

**Benchmark tests:**
```bash
go test -bench=. -benchmem ./...
```

**Profile CPU usage:**
```bash
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

**Profile memory usage:**
```bash
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

## Next Steps

Once you implement more features, add:

1. **Database tests** - Use testcontainers
2. **HTTP tests** - Use httptest
3. **Integration tests** - Full request/response cycle
4. **Load tests** - Use k6
5. **E2E tests** - Real user workflows
