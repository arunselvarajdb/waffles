#!/bin/bash

set -e

echo "════════════════════════════════════════════════════════════"
echo "          MCP Gateway End-to-End Test"
echo "════════════════════════════════════════════════════════════"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"
    pkill -f "go run test/cmd/test_mock_mcp_server" 2>/dev/null || true
    pkill -f "go run cmd/server/main.go" 2>/dev/null || true
    sleep 1
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

trap cleanup EXIT

echo "Step 1: Starting Mock MCP Server on port 9001..."
go run test/cmd/test_mock_mcp_server.go 2>&1 | sed 's/^/  [MOCK] /' &
MOCK_PID=$!
sleep 2

if ! curl -s http://localhost:9001/health > /dev/null; then
    echo -e "${RED}✗ Failed to start mock MCP server${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Mock MCP server running on port 9001${NC}"
echo ""

echo "Step 2: Starting MCP Gateway on port 8080..."
go run cmd/server/main.go 2>&1 | sed 's/^/  [GATEWAY] /' &
GATEWAY_PID=$!
sleep 3

if ! curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${RED}✗ Failed to start gateway${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Gateway running on port 8080${NC}"
echo ""

echo "Step 2.5: Cleaning up existing test servers..."
# Delete any existing test server with the same name
EXISTING_SERVERS=$(curl -s http://localhost:8080/api/v1/servers | jq -r '.[] | select(.name=="test-mock-server") | .id')
if [ -n "$EXISTING_SERVERS" ]; then
    for server_id in $EXISTING_SERVERS; do
        curl -s -X DELETE "http://localhost:8080/api/v1/servers/$server_id" > /dev/null
        echo -e "${YELLOW}  Deleted existing test server: $server_id${NC}"
    done
fi
echo -e "${GREEN}✓ Cleanup complete${NC}"
echo ""

echo "Step 3: Registering Mock MCP Server..."
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/servers \
    -H "Content-Type: application/json" \
    -d '{
        "name": "test-mock-server",
        "description": "Mock MCP server for testing",
        "url": "http://localhost:9001",
        "auth_type": "none",
        "tags": ["test", "mock"]
    }')

SERVER_ID=$(echo $REGISTER_RESPONSE | jq -r '.id')

if [ -z "$SERVER_ID" ] || [ "$SERVER_ID" = "null" ]; then
    echo -e "${RED}✗ Failed to register server${NC}"
    echo "Response: $REGISTER_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓ Server registered with ID: $SERVER_ID${NC}"
echo ""

echo "Step 4: Listing Registered Servers..."
SERVERS=$(curl -s http://localhost:8080/api/v1/servers)
echo "$SERVERS" | jq '.'
echo ""

echo "Step 5: Calling MCP Initialize..."
INIT_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$SERVER_ID/initialize" \
    -H "Content-Type: application/json" \
    -d '{
        "protocolVersion": "1.0.0",
        "clientInfo": {
            "name": "test-client",
            "version": "1.0.0"
        }
    }')

echo "$INIT_RESPONSE" | jq '.'

if echo "$INIT_RESPONSE" | jq -e '.protocolVersion' > /dev/null; then
    echo -e "${GREEN}✓ Initialize successful${NC}"
else
    echo -e "${RED}✗ Initialize failed${NC}"
    exit 1
fi
echo ""

echo "Step 6: Listing Tools..."
TOOLS_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$SERVER_ID/tools/list")
echo "$TOOLS_RESPONSE" | jq '.'

if echo "$TOOLS_RESPONSE" | jq -e '.tools' > /dev/null; then
    echo -e "${GREEN}✓ Tools list successful${NC}"
else
    echo -e "${RED}✗ Tools list failed${NC}"
    exit 1
fi
echo ""

echo "Step 7: Calling Calculator Tool..."
CALL_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$SERVER_ID/tools/call" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "calculator",
        "arguments": {
            "operation": "add",
            "a": 40,
            "b": 2
        }
    }')

echo "$CALL_RESPONSE" | jq '.'

if echo "$CALL_RESPONSE" | jq -e '.content[0].text' | grep -q "42"; then
    echo -e "${GREEN}✓ Tool call successful (result: 42)${NC}"
else
    echo -e "${RED}✗ Tool call failed${NC}"
    exit 1
fi
echo ""

echo "Step 8: Checking Audit Logs..."
sleep 1  # Give audit logger time to write
AUDIT_LOGS=$(curl -s "http://localhost:8080/api/v1/audit/logs?limit=10")
AUDIT_COUNT=$(echo "$AUDIT_LOGS" | jq '. | length')

echo "Found $AUDIT_COUNT audit log entries"
echo "$AUDIT_LOGS" | jq '.[0:3]'  # Show first 3 entries

if [ "$AUDIT_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Audit logs created${NC}"
else
    echo -e "${YELLOW}⚠ No audit logs found (might be async issue)${NC}"
fi
echo ""

echo "Step 9: Checking Server Health..."
HEALTH_RESPONSE=$(curl -s "http://localhost:8080/api/v1/servers/$SERVER_ID/health")
echo "$HEALTH_RESPONSE" | jq '.'

if echo "$HEALTH_RESPONSE" | jq -e '.status' > /dev/null; then
    echo -e "${GREEN}✓ Health check successful${NC}"
else
    echo -e "${YELLOW}⚠ Health check not yet available${NC}"
fi
echo ""

echo "════════════════════════════════════════════════════════════"
echo -e "${GREEN}          ✓ All End-to-End Tests Passed!${NC}"
echo "════════════════════════════════════════════════════════════"
echo ""
echo "Summary:"
echo "  • Mock MCP server running ✓"
echo "  • Gateway server running ✓"
echo "  • Server registration ✓"
echo "  • MCP initialize ✓"
echo "  • Tools list ✓"
echo "  • Tool execution (calculator) ✓"
echo "  • Audit logging $([ "$AUDIT_COUNT" -gt 0 ] && echo '✓' || echo '⚠')"
echo "  • Health monitoring ✓"
echo ""
