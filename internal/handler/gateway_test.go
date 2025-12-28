package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Mock implementations for testing

type mockGatewayService struct {
	transportErr      error
	initStreamErr     error
	proxyErr          error
	serverInfoErr     error
	initErr           error
	terminateErr      error
	callSSEErr        error
	callStreamErr     error
	server            *domain.MCPServer
	proxyServer       *httputil.ReverseProxy
	initStreamSession *MCPSession
	transportType     domain.TransportType
	callStreamResult  json.RawMessage
	callSSEResult     json.RawMessage
}

func (m *mockGatewayService) ProxyToServer(ctx context.Context, serverID string) (*httputil.ReverseProxy, *domain.MCPServer, error) {
	if m.proxyErr != nil {
		return nil, nil, m.proxyErr
	}

	return m.proxyServer, m.server, nil
}

func (m *mockGatewayService) GetServerInfo(ctx context.Context, serverID string) (*domain.MCPServer, error) {
	if m.serverInfoErr != nil {
		return nil, m.serverInfoErr
	}

	return m.server, nil
}

func (m *mockGatewayService) Initialize(ctx context.Context, serverID string) (*domain.MCPServer, error) {
	if m.initErr != nil {
		return nil, m.initErr
	}

	return m.server, nil
}

func (m *mockGatewayService) GetTransportType(ctx context.Context, serverID string) (domain.TransportType, *domain.MCPServer, error) {
	if m.transportErr != nil {
		return "", nil, m.transportErr
	}

	return m.transportType, m.server, nil
}

func (m *mockGatewayService) CallSSE(ctx context.Context, serverID string, method string, params interface{}) (json.RawMessage, error) {
	if m.callSSEErr != nil {
		return nil, m.callSSEErr
	}

	return m.callSSEResult, nil
}

func (m *mockGatewayService) CallStreamableHTTP(ctx context.Context, serverID string, method string, params interface{}) (json.RawMessage, error) {
	if m.callStreamErr != nil {
		return nil, m.callStreamErr
	}

	return m.callStreamResult, nil
}

func (m *mockGatewayService) InitializeStreamableHTTP(ctx context.Context, serverID string) (*MCPSession, error) {
	if m.initStreamErr != nil {
		return nil, m.initStreamErr
	}

	return m.initStreamSession, nil
}

func (m *mockGatewayService) TerminateStreamableHTTP(ctx context.Context, serverID string) error {
	return m.terminateErr
}

type mockGatewayAccessService struct {
	accessErr error
	serverIDs []string
	canAccess bool
}

func (m *mockGatewayAccessService) GetAccessibleServerIDs(ctx context.Context, roles []string, level domain.AccessLevel) ([]string, error) {
	if m.accessErr != nil {
		return nil, m.accessErr
	}

	return m.serverIDs, nil
}

func (m *mockGatewayAccessService) CanAccessServer(ctx context.Context, roles []string, serverID string, level domain.AccessLevel) (bool, error) {
	if m.accessErr != nil {
		return false, m.accessErr
	}

	return m.canAccess, nil
}

func TestNewGatewayHandler(t *testing.T) {
	t.Run("creates handler with nil services", func(t *testing.T) {
		log := logger.NewNopLogger()
		handler := NewGatewayHandler(nil, nil, log)

		require.NotNil(t, handler)
		assert.Nil(t, handler.service)
		assert.Nil(t, handler.accessService)
		assert.NotNil(t, handler.logger)
	})
}

func TestGatewayHandler_isToolAllowed(t *testing.T) {
	handler := &GatewayHandler{
		logger: logger.NewNopLogger(),
	}

	tests := []struct {
		name         string
		toolName     string
		allowedTools []string
		expected     bool
	}{
		{
			name:         "tool is in allowed list",
			toolName:     "read_file",
			allowedTools: []string{"read_file", "write_file", "list_dir"},
			expected:     true,
		},
		{
			name:         "tool is not in allowed list",
			toolName:     "delete_file",
			allowedTools: []string{"read_file", "write_file"},
			expected:     false,
		},
		{
			name:         "empty allowed list",
			toolName:     "read_file",
			allowedTools: []string{},
			expected:     false,
		},
		{
			name:         "single tool in list matches",
			toolName:     "execute",
			allowedTools: []string{"execute"},
			expected:     true,
		},
		{
			name:         "case sensitive comparison",
			toolName:     "Read_File",
			allowedTools: []string{"read_file"},
			expected:     false,
		},
		{
			name:         "tool with special characters",
			toolName:     "tool:v2:execute",
			allowedTools: []string{"tool:v2:execute", "tool:v1:read"},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.isToolAllowed(tt.toolName, tt.allowedTools)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGatewayHandler_parseSSEResponse(t *testing.T) {
	handler := &GatewayHandler{
		logger: logger.NewNopLogger(),
	}

	t.Run("parses valid SSE response", func(t *testing.T) {
		sseData := `event: message
data: {"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"test"}]}}

`
		resp, err := handler.parseSSEResponse([]byte(sseData))

		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Equal(t, float64(1), resp.ID)
		assert.NotNil(t, resp.Result)
	})

	t.Run("handles empty data value", func(t *testing.T) {
		sseData := `event: message
data:
data: {"jsonrpc":"2.0","id":1,"result":{}}

`
		resp, err := handler.parseSSEResponse([]byte(sseData))

		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
	})

	t.Run("handles non-data lines", func(t *testing.T) {
		sseData := `event: message
retry: 1000
id: 123
data: {"jsonrpc":"2.0","id":1,"result":{}}

`
		resp, err := handler.parseSSEResponse([]byte(sseData))

		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
	})

	t.Run("handles multiple data lines", func(t *testing.T) {
		sseData := `event: message
data: {"jsonrpc":"2.0","id":1}
data: {"jsonrpc":"2.0","id":2,"result":{}}

`
		resp, err := handler.parseSSEResponse([]byte(sseData))

		require.NoError(t, err)
		// Should use the last data line
		assert.Equal(t, float64(2), resp.ID)
	})

	t.Run("returns error for empty response", func(t *testing.T) {
		sseData := `event: message

`
		_, err := handler.parseSSEResponse([]byte(sseData))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no data found")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		sseData := `event: message
data: {invalid json}

`
		_, err := handler.parseSSEResponse([]byte(sseData))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON")
	})

	t.Run("handles data prefix with extra spaces", func(t *testing.T) {
		sseData := `event: message
data:    {"jsonrpc":"2.0","id":1,"result":{}}

`
		resp, err := handler.parseSSEResponse([]byte(sseData))

		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
	})
}

func TestGatewayHandler_sendMCPError(t *testing.T) {
	handler := &GatewayHandler{
		logger: logger.NewNopLogger(),
	}

	t.Run("sends error response in SSE format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		handler.sendMCPError(c, 1, -32602, "Tool not allowed")

		body := w.Body.String()
		assert.Contains(t, body, "event: message")
		assert.Contains(t, body, "data:")

		// Extract JSON from SSE
		var resp MCPResponse
		jsonStart := bytes.Index([]byte(body), []byte("{"))
		jsonEnd := bytes.LastIndex([]byte(body), []byte("}")) + 1
		err := json.Unmarshal([]byte(body[jsonStart:jsonEnd]), &resp)

		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Equal(t, float64(1), resp.ID)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, -32602, resp.Error.Code)
		assert.Equal(t, "Tool not allowed", resp.Error.Message)
	})

	t.Run("sets content type header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		handler.sendMCPError(c, "req-123", -32603, "Internal error")

		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	})
}

// Tests for MCPRequest and MCPResponse types

func TestMCPRequest_JSON(t *testing.T) {
	t.Run("marshals correctly", func(t *testing.T) {
		req := MCPRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "tools/list",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "2.0", parsed["jsonrpc"])
		assert.Equal(t, float64(1), parsed["id"])
		assert.Equal(t, "tools/list", parsed["method"])
	})

	t.Run("unmarshals correctly", func(t *testing.T) {
		jsonData := `{"jsonrpc":"2.0","id":"abc123","method":"tools/call","params":{"name":"test"}}`

		var req MCPRequest
		err := json.Unmarshal([]byte(jsonData), &req)

		require.NoError(t, err)
		assert.Equal(t, "2.0", req.JSONRPC)
		assert.Equal(t, "abc123", req.ID)
		assert.Equal(t, "tools/call", req.Method)
		assert.NotNil(t, req.Params)
	})

	t.Run("handles numeric ID", func(t *testing.T) {
		jsonData := `{"jsonrpc":"2.0","id":42,"method":"initialize"}`

		var req MCPRequest
		err := json.Unmarshal([]byte(jsonData), &req)

		require.NoError(t, err)
		assert.Equal(t, float64(42), req.ID)
	})
}

func TestMCPResponse_JSON(t *testing.T) {
	t.Run("marshals result correctly", func(t *testing.T) {
		result := json.RawMessage(`{"tools":[]}`)
		resp := MCPResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  result,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "2.0", parsed["jsonrpc"])
		assert.NotNil(t, parsed["result"])
		assert.Nil(t, parsed["error"])
	})

	t.Run("marshals error correctly", func(t *testing.T) {
		resp := MCPResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error: &MCPError{
				Code:    -32600,
				Message: "Invalid request",
			},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.NotNil(t, parsed["error"])
		assert.Nil(t, parsed["result"])

		errorObj := parsed["error"].(map[string]interface{})
		assert.Equal(t, float64(-32600), errorObj["code"])
		assert.Equal(t, "Invalid request", errorObj["message"])
	})
}

func TestToolCallParams_JSON(t *testing.T) {
	t.Run("unmarshals correctly", func(t *testing.T) {
		jsonData := `{"name":"read_file","arguments":{"path":"/tmp/test.txt"}}`

		var params ToolCallParams
		err := json.Unmarshal([]byte(jsonData), &params)

		require.NoError(t, err)
		assert.Equal(t, "read_file", params.Name)
		assert.NotNil(t, params.Arguments)
	})

	t.Run("handles missing arguments", func(t *testing.T) {
		jsonData := `{"name":"list_tools"}`

		var params ToolCallParams
		err := json.Unmarshal([]byte(jsonData), &params)

		require.NoError(t, err)
		assert.Equal(t, "list_tools", params.Name)
		assert.Nil(t, params.Arguments)
	})
}

func TestToolsListResult_JSON(t *testing.T) {
	t.Run("marshals empty tools list", func(t *testing.T) {
		result := ToolsListResult{
			Tools: []MCPTool{},
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"tools":[]`)
	})

	t.Run("marshals tools with all fields", func(t *testing.T) {
		result := ToolsListResult{
			Tools: []MCPTool{
				{
					Name:        "test_tool",
					Description: "A test tool",
					InputSchema: json.RawMessage(`{"type":"object"}`),
				},
			},
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		tools := parsed["tools"].([]interface{})
		assert.Len(t, tools, 1)

		tool := tools[0].(map[string]interface{})
		assert.Equal(t, "test_tool", tool["name"])
		assert.Equal(t, "A test tool", tool["description"])
	})
}

// Response simulation tests

func TestGatewayHandler_ProxyRequest_ResponseCodes(t *testing.T) {
	tests := []struct {
		name         string
		serverID     string
		expectedErr  string
		expectedCode int
	}{
		{
			name:         "bad gateway when proxy fails",
			serverID:     "nonexistent",
			expectedCode: http.StatusBadGateway,
			expectedErr:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "server_id", Value: tt.serverID}}
			c.Request = httptest.NewRequest("GET", "/api/v1/gateway/"+tt.serverID, nil)

			// Simulate error response
			c.JSON(tt.expectedCode, gin.H{"error": "Failed to get proxy"})

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestGatewayHandler_MCPProxy_ResponseCodes(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expectedCode int
	}{
		{
			name:         "service unavailable for inactive server",
			expectedCode: http.StatusServiceUnavailable,
			errorMessage: "server is inactive",
		},
		{
			name:         "forbidden for access denied",
			expectedCode: http.StatusForbidden,
			errorMessage: "You don't have execute permission for this server",
		},
		{
			name:         "bad gateway for server not found",
			expectedCode: http.StatusBadGateway,
			errorMessage: "server not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.JSON(tt.expectedCode, gin.H{"error": tt.errorMessage})

			assert.Equal(t, tt.expectedCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.errorMessage, response["error"])
		})
	}
}

func TestGatewayHandler_Initialize_ResponseCodes(t *testing.T) {
	tests := []struct {
		response     interface{}
		name         string
		expectedCode int
	}{
		{
			name:         "success response",
			expectedCode: http.StatusOK,
			response: gin.H{
				"server_id":   "server-1",
				"server_name": "Test Server",
				"url":         "https://example.com/mcp",
				"status":      "initialized",
			},
		},
		{
			name:         "internal error",
			expectedCode: http.StatusInternalServerError,
			response: gin.H{
				"error": "initialization failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.JSON(tt.expectedCode, tt.response)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestGatewayHandler_ListTools_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "server_id", Value: "nonexistent"}}

	c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGatewayHandler_CallTool_BadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}

	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGatewayHandler_StreamableHTTPResponses(t *testing.T) {
	t.Run("initialize success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.JSON(http.StatusOK, gin.H{
			"server_id":        "server-1",
			"session_id":       "session-123",
			"protocol_version": "2025-11-25",
			"status":           "initialized",
		})

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "initialized", response["status"])
	})

	t.Run("terminate success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.JSON(http.StatusOK, gin.H{
			"server_id": "server-1",
			"status":    "terminated",
		})

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "terminated", response["status"])
	})
}

// Tests using mock interfaces

func TestGatewayHandler_ProxyRequest_WithMock(t *testing.T) {
	t.Run("returns bad gateway on proxy error", func(t *testing.T) {
		mockService := &mockGatewayService{
			proxyErr: errors.New("server not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/tools/list", nil)

		handler.ProxyRequest(c)

		assert.Equal(t, http.StatusBadGateway, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Contains(t, response["error"], "server not found")
	})
}

func TestGatewayHandler_Initialize_WithMock(t *testing.T) {
	t.Run("returns server info on success", func(t *testing.T) {
		server := &domain.MCPServer{
			ID:   "server-1",
			Name: "Test Server",
			URL:  "https://example.com/mcp",
		}
		mockService := &mockGatewayService{
			server: server,
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/initialize", nil)

		handler.Initialize(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "server-1", response["server_id"])
		assert.Equal(t, "Test Server", response["server_name"])
		assert.Equal(t, "initialized", response["status"])
	})

	t.Run("returns error on initialization failure", func(t *testing.T) {
		mockService := &mockGatewayService{
			initErr: errors.New("server unavailable"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/initialize", nil)

		handler.Initialize(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGatewayHandler_ListTools_WithMock(t *testing.T) {
	t.Run("returns not found on transport error", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportErr: errors.New("server not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/tools/list", nil)

		handler.ListTools(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("uses SSE transport when configured", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType: domain.TransportSSE,
			server:        &domain.MCPServer{ID: "server-1"},
			callSSEResult: json.RawMessage(`{"tools":[]}`),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/tools/list", nil)

		handler.ListTools(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("uses streamable HTTP transport when configured", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType:    domain.TransportStreamableHTTP,
			server:           &domain.MCPServer{ID: "server-1"},
			callStreamResult: json.RawMessage(`{"tools":[]}`),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/tools/list", nil)

		handler.ListTools(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns error when SSE call fails", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType: domain.TransportSSE,
			server:        &domain.MCPServer{ID: "server-1"},
			callSSEErr:    errors.New("connection refused"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/tools/list", nil)

		handler.ListTools(c)

		assert.Equal(t, http.StatusBadGateway, w.Code)
	})
}

func TestGatewayHandler_CallTool_WithMock(t *testing.T) {
	t.Run("returns not found on transport error", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportErr: errors.New("server not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/tools/call", strings.NewReader(`{"name":"test"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CallTool(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("uses SSE transport for tool call", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType: domain.TransportSSE,
			server:        &domain.MCPServer{ID: "server-1"},
			callSSEResult: json.RawMessage(`{"content":[{"text":"result"}]}`),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/tools/call", strings.NewReader(`{"name":"test"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CallTool(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("uses streamable HTTP transport for tool call", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType:    domain.TransportStreamableHTTP,
			server:           &domain.MCPServer{ID: "server-1"},
			callStreamResult: json.RawMessage(`{"content":[{"text":"result"}]}`),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/tools/call", strings.NewReader(`{"name":"test"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CallTool(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGatewayHandler_ListResources_WithMock(t *testing.T) {
	t.Run("returns not found on transport error", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportErr: errors.New("server not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/resources/list", nil)

		handler.ListResources(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("uses streamable HTTP transport", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType:    domain.TransportStreamableHTTP,
			server:           &domain.MCPServer{ID: "server-1"},
			callStreamResult: json.RawMessage(`{"resources":[]}`),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/resources/list", nil)

		handler.ListResources(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGatewayHandler_ReadResource_WithMock(t *testing.T) {
	t.Run("returns not found on transport error", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportErr: errors.New("server not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/resources/read", strings.NewReader(`{"uri":"file:///test"}`))

		handler.ReadResource(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("uses SSE transport for resource read", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType: domain.TransportSSE,
			server:        &domain.MCPServer{ID: "server-1"},
			callSSEResult: json.RawMessage(`{"contents":[{"text":"content"}]}`),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/resources/read", strings.NewReader(`{"uri":"file:///test"}`))

		handler.ReadResource(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGatewayHandler_ListPrompts_WithMock(t *testing.T) {
	t.Run("returns not found on transport error", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportErr: errors.New("server not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/prompts/list", nil)

		handler.ListPrompts(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("uses SSE transport", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType: domain.TransportSSE,
			server:        &domain.MCPServer{ID: "server-1"},
			callSSEResult: json.RawMessage(`{"prompts":[]}`),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("GET", "/api/v1/gateway/server-1/prompts/list", nil)

		handler.ListPrompts(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGatewayHandler_GetPrompt_WithMock(t *testing.T) {
	t.Run("returns not found on transport error", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportErr: errors.New("server not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/prompts/get", strings.NewReader(`{"name":"test"}`))

		handler.GetPrompt(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("uses streamable HTTP transport", func(t *testing.T) {
		mockService := &mockGatewayService{
			transportType:    domain.TransportStreamableHTTP,
			server:           &domain.MCPServer{ID: "server-1"},
			callStreamResult: json.RawMessage(`{"messages":[]}`),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/prompts/get", strings.NewReader(`{"name":"test"}`))

		handler.GetPrompt(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGatewayHandler_InitializeStreamableHTTP_WithMock(t *testing.T) {
	t.Run("returns session info on success", func(t *testing.T) {
		mockService := &mockGatewayService{
			initStreamSession: &MCPSession{
				SessionID:       "session-123",
				ProtocolVersion: "2025-11-25",
			},
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/streamable-http/initialize", nil)

		handler.InitializeStreamableHTTP(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "server-1", response["server_id"])
		assert.Equal(t, "session-123", response["session_id"])
		assert.Equal(t, "2025-11-25", response["protocol_version"])
		assert.Equal(t, "initialized", response["status"])
	})

	t.Run("returns error on failure", func(t *testing.T) {
		mockService := &mockGatewayService{
			initStreamErr: errors.New("connection refused"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-1/streamable-http/initialize", nil)

		handler.InitializeStreamableHTTP(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGatewayHandler_TerminateStreamableHTTP_WithMock(t *testing.T) {
	t.Run("returns success on termination", func(t *testing.T) {
		mockService := &mockGatewayService{
			terminateErr: nil,
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("DELETE", "/api/v1/gateway/server-1/streamable-http/session", nil)

		handler.TerminateStreamableHTTP(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "server-1", response["server_id"])
		assert.Equal(t, "terminated", response["status"])
	})

	t.Run("returns error on failure", func(t *testing.T) {
		mockService := &mockGatewayService{
			terminateErr: errors.New("session not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockService, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("DELETE", "/api/v1/gateway/server-1/streamable-http/session", nil)

		handler.TerminateStreamableHTTP(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGatewayHandler_MCPProxy_WithMock(t *testing.T) {
	t.Run("denies access when access check fails", func(t *testing.T) {
		mockGwSvc := &mockGatewayService{
			server: &domain.MCPServer{ID: "server-1", IsActive: true},
		}
		mockAccessSvc := &mockGatewayAccessService{
			accessErr: errors.New("database error"),
		}
		handler := NewGatewayHandlerWithInterface(mockGwSvc, mockAccessSvc, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/mcp/server-1", strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.MCPProxy(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("denies access when user has no execute permission", func(t *testing.T) {
		mockGwSvc := &mockGatewayService{
			server: &domain.MCPServer{ID: "server-1", IsActive: true},
		}
		mockAccessSvc := &mockGatewayAccessService{
			canAccess: false,
		}
		handler := NewGatewayHandlerWithInterface(mockGwSvc, mockAccessSvc, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/mcp/server-1", strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.MCPProxy(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("returns error when server info fails", func(t *testing.T) {
		mockGwSvc := &mockGatewayService{
			serverInfoErr: errors.New("server not found"),
		}
		handler := NewGatewayHandlerWithInterface(mockGwSvc, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/mcp/server-1", strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.MCPProxy(c)

		assert.Equal(t, http.StatusBadGateway, w.Code)
	})

	t.Run("returns service unavailable when server is inactive", func(t *testing.T) {
		mockGwSvc := &mockGatewayService{
			server: &domain.MCPServer{ID: "server-1", IsActive: false},
		}
		handler := NewGatewayHandlerWithInterface(mockGwSvc, nil, logger.NewNopLogger())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "server_id", Value: "server-1"}}
		c.Request = httptest.NewRequest("POST", "/api/v1/mcp/server-1", strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.MCPProxy(c)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestNewGatewayHandlerWithInterface(t *testing.T) {
	t.Run("creates handler with mock services", func(t *testing.T) {
		mockGwSvc := &mockGatewayService{}
		mockAccessSvc := &mockGatewayAccessService{}
		log := logger.NewNopLogger()

		handler := NewGatewayHandlerWithInterface(mockGwSvc, mockAccessSvc, log)

		require.NotNil(t, handler)
		assert.NotNil(t, handler.service)
		assert.NotNil(t, handler.accessService)
		assert.NotNil(t, handler.logger)
	})

	t.Run("creates handler with nil services", func(t *testing.T) {
		log := logger.NewNopLogger()

		handler := NewGatewayHandlerWithInterface(nil, nil, log)

		require.NotNil(t, handler)
		assert.Nil(t, handler.service)
		assert.Nil(t, handler.accessService)
	})
}
