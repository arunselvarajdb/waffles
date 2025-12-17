#!/bin/bash

echo "=== MCP Gateway Test Script ==="
echo ""

# Kill any existing processes
echo "1. Cleaning up old processes..."
pkill -f "test_mock_mcp_server" 2>/dev/null || true
pkill -f "cmd/server/main" 2>/dev/null || true
sleep 2

# Start mock MCP server
echo "2. Starting Mock MCP Server on port 9001..."
nohup go run test_mock_mcp_server.go > /tmp/mock_mcp.log 2>&1 &
MOCK_PID=$!
sleep 2

# Verify mock server
if curl -s http://localhost:9001/initialize > /dev/null; then
    echo "   ✅ Mock MCP Server running (PID: $MOCK_PID)"
else
    echo "   ❌ Mock MCP Server failed to start"
    exit 1
fi

# Start gateway server
echo "3. Starting Gateway Server on port 8080..."
nohup go run cmd/server/main.go > /tmp/gateway.log 2>&1 &
GATEWAY_PID=$!
sleep 4

# Verify gateway
if curl -s http://localhost:8080/health > /dev/null; then
    echo "   ✅ Gateway Server running (PID: $GATEWAY_PID)"
else
    echo "   ❌ Gateway Server failed to start"
    tail -20 /tmp/gateway.log
    exit 1
fi

echo ""
echo "=== Running Tests ==="
echo ""

# Clean up any existing test server
echo "0. Cleaning up old test data..."
LIST_RESP=$(curl -s http://localhost:8080/api/v1/servers)
if echo "$LIST_RESP" | grep -q 'test-mock-server'; then
    EXISTING_ID=$(echo "$LIST_RESP" | grep -o '"id":"[^"]*","name":"test-mock-server"' | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$EXISTING_ID" ]; then
        curl -s -X DELETE "http://localhost:8080/api/v1/servers/$EXISTING_ID" > /dev/null
        echo "   ✅ Deleted existing test server"
    fi
fi

echo ""

# Test 1: Register Mock MCP Server
echo "TEST 1: Register Mock MCP Server"
REGISTER_RESP=$(curl -s -X POST http://localhost:8080/api/v1/servers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-mock-server",
    "description": "Mock MCP server for testing",
    "url": "http://localhost:9001",
    "auth_type": "none"
  }')

if echo "$REGISTER_RESP" | grep -q '"id"'; then
    SERVER_ID=$(echo "$REGISTER_RESP" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "✅ Server registered successfully"
    echo "   Server ID: $SERVER_ID"
else
    echo "❌ Failed to register server"
    echo "   Response: $REGISTER_RESP"
    exit 1
fi

echo ""

# Test 2: List Servers
echo "TEST 2: List Registered Servers"
LIST_RESP=$(curl -s http://localhost:8080/api/v1/servers)
if echo "$LIST_RESP" | grep -q 'test-mock-server'; then
    echo "✅ Server appears in list"
else
    echo "❌ Server not found in list"
fi

echo ""

# Test 3: Gateway Proxy - Initialize
echo "TEST 3: Gateway Proxy - Initialize"
INIT_RESP=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$SERVER_ID/initialize")
if echo "$INIT_RESP" | grep -q 'initialized'; then
    echo "✅ Initialize request successful"
    echo "   Response: $INIT_RESP"
else
    echo "❌ Initialize request failed"
    echo "   Response: $INIT_RESP"
fi

echo ""

# Test 4: Gateway Proxy - List Tools
echo "TEST 4: Gateway Proxy - List Tools"
TOOLS_RESP=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$SERVER_ID/tools/list")
if echo "$TOOLS_RESP" | grep -q 'calculator'; then
    echo "✅ Tools/list proxied successfully"
    echo "   Tools found: calculator, echo"
else
    echo "❌ Tools/list proxy failed"
    echo "   Response: $TOOLS_RESP"
fi

echo ""

# Test 5: Gateway Proxy - Call Tool
echo "TEST 5: Gateway Proxy - Call Tool"
CALL_RESP=$(curl -s -X POST "http://localhost:8080/api/v1/gateway/$SERVER_ID/tools/call" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "calculator",
    "arguments": {"operation": "add", "a": 5, "b": 3}
  }')
if echo "$CALL_RESP" | grep -q 'Mock result'; then
    echo "✅ Tools/call proxied successfully"
    echo "   Response: $CALL_RESP"
else
    echo "❌ Tools/call proxy failed"
    echo "   Response: $CALL_RESP"
fi

echo ""
echo "=== Test Summary ==="
echo "All tests completed!"
echo ""
echo "To view logs:"
echo "  Mock MCP: tail -f /tmp/mock_mcp.log"
echo "  Gateway:  tail -f /tmp/gateway.log"
echo ""
echo "To stop servers:"
echo "  kill $MOCK_PID $GATEWAY_PID"
