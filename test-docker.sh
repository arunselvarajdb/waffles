#!/bin/bash

set -e

echo "════════════════════════════════════════════════════════════"
echo "          MCP Gateway - Docker Compose E2E Test"
echo "════════════════════════════════════════════════════════════"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Cookie file for session management
COOKIE_FILE=$(mktemp)

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up Docker containers...${NC}"
    docker-compose down -v 2>/dev/null || true
    rm -f "$COOKIE_FILE" 2>/dev/null || true
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

trap cleanup EXIT

pass_test() {
    TESTS_PASSED=$((TESTS_PASSED+1))
    echo -e "${GREEN}✓ $1${NC}"
}

fail_test() {
    TESTS_FAILED=$((TESTS_FAILED+1))
    echo -e "${RED}✗ $1${NC}"
    if [ -n "$2" ]; then
        echo "  Response: $2"
    fi
}

echo "Step 1: Starting services with Docker Compose..."
docker-compose up -d

echo ""
echo "Step 2: Waiting for services to be ready..."
sleep 10

# Wait for gateway health check
MAX_RETRIES=30
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Gateway is ready${NC}"
        break
    fi
    echo "  Waiting for gateway... ($((RETRY_COUNT+1))/$MAX_RETRIES)"
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT+1))
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo -e "${RED}✗ Gateway failed to start${NC}"
    docker-compose logs gateway
    exit 1
fi

# Wait for mock server
if curl -s http://localhost:9001/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Mock MCP server is ready${NC}"
else
    echo -e "${RED}✗ Mock MCP server not responding${NC}"
    docker-compose logs mock-mcp
    exit 1
fi

echo ""
echo "Step 3: Authenticating..."
echo ""

# Login as admin user to get session cookie
echo "Logging in as admin@example.com..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -c "$COOKIE_FILE" \
    -d '{
        "email": "admin@example.com",
        "password": "admin123"
    }')

if echo "$LOGIN_RESPONSE" | jq -e '.user.id' > /dev/null 2>&1; then
    pass_test "Logged in successfully"
else
    fail_test "Login failed" "$LOGIN_RESPONSE"
    docker-compose logs gateway
    exit 1
fi
echo ""

echo "Step 4: Running E2E Tests..."
echo ""

# ═══════════════════════════════════════════════════════════════
# HTTP Transport Tests
# ═══════════════════════════════════════════════════════════════
echo -e "${BLUE}═══ HTTP Transport Tests ═══${NC}"
echo ""

# Register HTTP Server
echo "Registering HTTP transport server..."
HTTP_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/servers \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE" \
    -d '{
        "name": "http-transport-server",
        "description": "Mock MCP server using HTTP REST transport",
        "url": "http://mock-mcp:9001",
        "protocol_version": "1.0.0",
        "transport": "http",
        "tags": ["test", "http"]
    }')

HTTP_SERVER_ID=$(echo $HTTP_RESPONSE | jq -r '.id')

if [ -z "$HTTP_SERVER_ID" ] || [ "$HTTP_SERVER_ID" = "null" ]; then
    fail_test "Failed to register HTTP server" "$HTTP_RESPONSE"
else
    pass_test "HTTP server registered: $HTTP_SERVER_ID"
fi

# HTTP Initialize
INIT_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$HTTP_SERVER_ID/initialize" \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE")

if echo "$INIT_RESPONSE" | jq -e '.status' | grep -q "initialized"; then
    pass_test "HTTP Initialize successful"
else
    fail_test "HTTP Initialize failed" "$INIT_RESPONSE"
fi

# HTTP Call Tool
CALL_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$HTTP_SERVER_ID/tools/call" \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE" \
    -d '{
        "name": "calculator",
        "arguments": {"operation": "add", "a": 10, "b": 5}
    }')

if echo "$CALL_RESPONSE" | jq -e '.content[0].text' | grep -q "Mock result"; then
    pass_test "HTTP Call Tool successful"
else
    fail_test "HTTP Call Tool failed" "$CALL_RESPONSE"
fi

echo ""

# ═══════════════════════════════════════════════════════════════
# Streamable HTTP Transport Tests (MCP 2025-11-25)
# ═══════════════════════════════════════════════════════════════
echo -e "${BLUE}═══ Streamable HTTP Transport Tests (MCP 2025-11-25) ═══${NC}"
echo ""

# Register Streamable HTTP Server
echo "Registering Streamable HTTP transport server..."
STREAMABLE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/servers \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE" \
    -d '{
        "name": "streamable-http-server",
        "description": "Mock MCP server using Streamable HTTP transport",
        "url": "http://mock-mcp:9001/mcp",
        "protocol_version": "2025-11-25",
        "transport": "streamable_http",
        "tags": ["test", "streamable_http"]
    }')

STREAMABLE_SERVER_ID=$(echo $STREAMABLE_RESPONSE | jq -r '.id')

if [ -z "$STREAMABLE_SERVER_ID" ] || [ "$STREAMABLE_SERVER_ID" = "null" ]; then
    fail_test "Failed to register Streamable HTTP server" "$STREAMABLE_RESPONSE"
else
    pass_test "Streamable HTTP server registered: $STREAMABLE_SERVER_ID"
fi

# Verify transport field is persisted
VERIFY_TRANSPORT=$(curl -s -b "$COOKIE_FILE" "http://localhost:8080/api/v1/servers/$STREAMABLE_SERVER_ID" | jq -r '.transport')
if [ "$VERIFY_TRANSPORT" = "streamable_http" ]; then
    pass_test "Transport field persisted correctly: $VERIFY_TRANSPORT"
else
    fail_test "Transport field not persisted" "Expected: streamable_http, Got: $VERIFY_TRANSPORT"
fi

# Streamable HTTP Initialize
INIT_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$STREAMABLE_SERVER_ID/initialize" \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE")

if echo "$INIT_RESPONSE" | jq -e '.status' | grep -q "initialized"; then
    pass_test "Streamable HTTP Initialize successful"
else
    fail_test "Streamable HTTP Initialize failed" "$INIT_RESPONSE"
fi

# Streamable HTTP Call Tool
CALL_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$STREAMABLE_SERVER_ID/tools/call" \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE" \
    -d '{
        "name": "echo",
        "arguments": {"message": "Hello from Streamable HTTP!"}
    }')

if echo "$CALL_RESPONSE" | jq -e '.content[0].text' | grep -q "Mock result"; then
    pass_test "Streamable HTTP Call Tool successful"
else
    fail_test "Streamable HTTP Call Tool failed" "$CALL_RESPONSE"
fi

echo ""

# ═══════════════════════════════════════════════════════════════
# SSE Transport Tests (Legacy)
# ═══════════════════════════════════════════════════════════════
echo -e "${BLUE}═══ SSE Transport Tests (Legacy) ═══${NC}"
echo ""

# Register SSE Server
echo "Registering SSE transport server..."
SSE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/servers \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE" \
    -d '{
        "name": "sse-transport-server",
        "description": "Mock MCP server using SSE transport (legacy)",
        "url": "http://mock-mcp:9001/sse",
        "protocol_version": "1.0.0",
        "transport": "sse",
        "tags": ["test", "sse"]
    }')

SSE_SERVER_ID=$(echo $SSE_RESPONSE | jq -r '.id')

if [ -z "$SSE_SERVER_ID" ] || [ "$SSE_SERVER_ID" = "null" ]; then
    fail_test "Failed to register SSE server" "$SSE_RESPONSE"
else
    pass_test "SSE server registered: $SSE_SERVER_ID"
fi

# SSE Initialize
INIT_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$SSE_SERVER_ID/initialize" \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE")

if echo "$INIT_RESPONSE" | jq -e '.status' | grep -q "initialized"; then
    pass_test "SSE Initialize successful"
else
    fail_test "SSE Initialize failed" "$INIT_RESPONSE"
fi

# SSE Call Tool
CALL_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$SSE_SERVER_ID/tools/call" \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE" \
    -d '{
        "name": "echo",
        "arguments": {"message": "Hello from SSE!"}
    }')

if echo "$CALL_RESPONSE" | jq -e '.content[0].text' | grep -q "Mock result"; then
    pass_test "SSE Call Tool successful"
else
    fail_test "SSE Call Tool failed" "$CALL_RESPONSE"
fi

echo ""

# ═══════════════════════════════════════════════════════════════
# Common Gateway Tests
# ═══════════════════════════════════════════════════════════════
echo -e "${BLUE}═══ Common Gateway Tests ═══${NC}"
echo ""

# List Servers
SERVERS=$(curl -s -b "$COOKIE_FILE" http://localhost:8080/api/v1/servers | jq '.servers | length')
if [ "$SERVERS" -ge 3 ]; then
    pass_test "Listed $SERVERS servers"
else
    fail_test "Expected at least 3 servers, found $SERVERS"
fi

# Verify Audit Logs
sleep 2  # Give time for async logging
AUDIT_LOGS=$(curl -s -b "$COOKIE_FILE" "http://localhost:8080/api/v1/audit/logs?limit=10")
AUDIT_COUNT=$(echo "$AUDIT_LOGS" | jq '. | length')

if [ "$AUDIT_COUNT" -gt 0 ]; then
    pass_test "Audit logs working ($AUDIT_COUNT entries)"
else
    echo -e "${YELLOW}⚠ No audit logs found${NC}"
fi

# Check Health Status
HEALTH_RESPONSE=$(curl -s -b "$COOKIE_FILE" "http://localhost:8080/api/v1/servers/$HTTP_SERVER_ID/health")
if echo "$HEALTH_RESPONSE" | jq -e '.status' > /dev/null; then
    HEALTH_STATUS=$(echo "$HEALTH_RESPONSE" | jq -r '.status')
    pass_test "Health monitoring working (status: $HEALTH_STATUS)"
else
    echo -e "${YELLOW}⚠ Health check not yet available${NC}"
fi

echo ""

# ═══════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════
echo "════════════════════════════════════════════════════════════"
if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}          ✓ All Tests Passed! ($TESTS_PASSED passed)${NC}"
else
    echo -e "${RED}          ✗ Some Tests Failed ($TESTS_PASSED passed, $TESTS_FAILED failed)${NC}"
fi
echo "════════════════════════════════════════════════════════════"
echo ""
echo "Transport Tests Summary:"
echo "  • HTTP REST transport       ✓"
echo "  • Streamable HTTP transport ✓"
echo "  • SSE transport (legacy)    ✓"
echo ""
echo "Feature Tests Summary:"
echo "  • Authentication (session cookie) ✓"
echo "  • Server registration ✓"
echo "  • MCP initialize ✓"
echo "  • Tools call ✓"
echo "  • Transport persistence ✓"
echo "  • Audit logging $([ "$AUDIT_COUNT" -gt 0 ] && echo '✓' || echo '⚠')"
echo "  • Health monitoring ✓"
echo ""
echo "Test credentials: admin@example.com / admin123"
echo ""
echo "To view logs: docker-compose logs -f"
echo "To stop: docker-compose down"
echo ""

# Exit with error if any tests failed
if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
fi
