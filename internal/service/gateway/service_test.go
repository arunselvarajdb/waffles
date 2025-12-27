package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

func TestNewService(t *testing.T) {
	t.Run("creates service with nil dependencies", func(t *testing.T) {
		log := logger.NewNopLogger()
		svc := NewService(nil, log, nil)

		require.NotNil(t, svc)
		assert.Nil(t, svc.repo)
		assert.NotNil(t, svc.logger)
		assert.Nil(t, svc.metrics)
		assert.NotNil(t, svc.sseClient)
		assert.NotNil(t, svc.streamableHTTPClient)
	})
}

func TestNewSSEClient(t *testing.T) {
	t.Run("creates client with default timeout", func(t *testing.T) {
		log := logger.NewNopLogger()
		client := NewSSEClient(log, 30*time.Second)

		require.NotNil(t, client)
		assert.NotNil(t, client.httpClient)
		assert.NotNil(t, client.logger)
		assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	})

	t.Run("creates client with custom timeout", func(t *testing.T) {
		log := logger.NewNopLogger()
		client := NewSSEClient(log, 60*time.Second)

		require.NotNil(t, client)
		assert.Equal(t, 60*time.Second, client.httpClient.Timeout)
	})
}

func TestNewStreamableHTTPClient(t *testing.T) {
	t.Run("creates client with timeout and empty sessions", func(t *testing.T) {
		log := logger.NewNopLogger()
		client := NewStreamableHTTPClient(log, 30*time.Second)

		require.NotNil(t, client)
		assert.NotNil(t, client.httpClient)
		assert.NotNil(t, client.logger)
		assert.NotNil(t, client.sessions)
		assert.Len(t, client.sessions, 0)
	})
}

func TestIsSSEServer(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "URL ending with /mcp is SSE",
			url:      "http://localhost:8080/mcp",
			expected: true,
		},
		{
			name:     "URL without /mcp is not SSE",
			url:      "http://localhost:8080/api",
			expected: false,
		},
		{
			name:     "URL with /mcp in path but not at end",
			url:      "http://localhost:8080/mcp/tools",
			expected: false,
		},
		{
			name:     "HTTPS URL ending with /mcp",
			url:      "https://server.example.com/mcp",
			expected: true,
		},
		{
			name:     "URL with port ending in /mcp",
			url:      "http://127.0.0.1:3000/mcp",
			expected: true,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &domain.MCPServer{URL: tt.url}
			result := IsSSEServer(server)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStreamableHTTPServer(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "URL ending with /mcp is Streamable HTTP",
			url:      "http://localhost:8080/mcp",
			expected: true,
		},
		{
			name:     "URL without /mcp is not Streamable HTTP",
			url:      "http://localhost:8080/api/v1",
			expected: false,
		},
		{
			name:     "URL with /mcp in middle is not Streamable HTTP",
			url:      "http://localhost:8080/mcp/stream",
			expected: false,
		},
		{
			name:     "HTTPS URL with /mcp suffix",
			url:      "https://mcp-server.example.com/mcp",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &domain.MCPServer{URL: tt.url}
			result := IsStreamableHTTPServer(server)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONRPCRequest_Marshal(t *testing.T) {
	t.Run("marshals correctly with params", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  map[string]string{"name": "test_tool"},
			ID:      42,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "2.0", parsed["jsonrpc"])
		assert.Equal(t, "tools/call", parsed["method"])
		assert.Equal(t, float64(42), parsed["id"])
		assert.NotNil(t, parsed["params"])
	})

	t.Run("omits empty params", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      1,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		_, hasParams := parsed["params"]
		assert.False(t, hasParams, "params should be omitted when nil")
	})
}

func TestJSONRPCResponse_Unmarshal(t *testing.T) {
	t.Run("unmarshals success response", func(t *testing.T) {
		jsonData := `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`

		var resp JSONRPCResponse
		err := json.Unmarshal([]byte(jsonData), &resp)

		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Equal(t, float64(1), resp.ID)
		assert.NotNil(t, resp.Result)
		assert.Nil(t, resp.Error)
	})

	t.Run("unmarshals error response", func(t *testing.T) {
		jsonData := `{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"Invalid request"}}`

		var resp JSONRPCResponse
		err := json.Unmarshal([]byte(jsonData), &resp)

		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Nil(t, resp.Result)
		require.NotNil(t, resp.Error)
		assert.Equal(t, -32600, resp.Error.Code)
		assert.Equal(t, "Invalid request", resp.Error.Message)
	})

	t.Run("handles string ID", func(t *testing.T) {
		jsonData := `{"jsonrpc":"2.0","id":"req-abc","result":{}}`

		var resp JSONRPCResponse
		err := json.Unmarshal([]byte(jsonData), &resp)

		require.NoError(t, err)
		assert.Equal(t, "req-abc", resp.ID)
	})
}

func TestJSONRPCError_Marshal(t *testing.T) {
	t.Run("marshals standard error", func(t *testing.T) {
		rpcErr := JSONRPCError{
			Code:    -32602,
			Message: "Invalid params",
		}

		data, err := json.Marshal(rpcErr)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, float64(-32602), parsed["code"])
		assert.Equal(t, "Invalid params", parsed["message"])
	})

	t.Run("marshals error with data", func(t *testing.T) {
		rpcErr := JSONRPCError{
			Code:    -32000,
			Message: "Server error",
			Data:    map[string]string{"detail": "database connection failed"},
		}

		data, err := json.Marshal(rpcErr)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.NotNil(t, parsed["data"])
		dataMap := parsed["data"].(map[string]interface{})
		assert.Equal(t, "database connection failed", dataMap["detail"])
	})
}

func TestMCPSession(t *testing.T) {
	t.Run("creates session with all fields", func(t *testing.T) {
		now := time.Now()
		session := &MCPSession{
			SessionID:       "sess-123",
			ServerID:        "server-456",
			ServerURL:       "http://localhost:8080/mcp",
			Initialized:     true,
			ProtocolVersion: "2025-11-25",
			LastEventID:     "evt-789",
			CreatedAt:       now,
		}

		assert.Equal(t, "sess-123", session.SessionID)
		assert.Equal(t, "server-456", session.ServerID)
		assert.True(t, session.Initialized)
		assert.Equal(t, "2025-11-25", session.ProtocolVersion)
	})
}

func TestInitializeParams(t *testing.T) {
	t.Run("marshals correctly", func(t *testing.T) {
		params := InitializeParams{
			ProtocolVersion: "2025-11-25",
			ClientInfo: ClientInfo{
				Name:    "waffles",
				Version: "1.0.0",
			},
		}

		data, err := json.Marshal(params)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "2025-11-25", parsed["protocolVersion"])

		clientInfo := parsed["clientInfo"].(map[string]interface{})
		assert.Equal(t, "waffles", clientInfo["name"])
		assert.Equal(t, "1.0.0", clientInfo["version"])
	})
}

func TestToolsCallParams(t *testing.T) {
	t.Run("marshals with arguments", func(t *testing.T) {
		params := ToolsCallParams{
			Name: "read_file",
			Arguments: map[string]interface{}{
				"path": "/tmp/test.txt",
			},
		}

		data, err := json.Marshal(params)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "read_file", parsed["name"])
		assert.NotNil(t, parsed["arguments"])
	})

	t.Run("omits empty arguments", func(t *testing.T) {
		params := ToolsCallParams{
			Name: "list_files",
		}

		data, err := json.Marshal(params)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		_, hasArgs := parsed["arguments"]
		assert.False(t, hasArgs)
	})
}

func TestResourcesReadParams(t *testing.T) {
	t.Run("marshals correctly", func(t *testing.T) {
		params := ResourcesReadParams{
			URI: "file:///tmp/example.txt",
		}

		data, err := json.Marshal(params)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "file:///tmp/example.txt", parsed["uri"])
	})
}

func TestPromptsGetParams(t *testing.T) {
	t.Run("marshals with arguments", func(t *testing.T) {
		params := PromptsGetParams{
			Name: "code_review",
			Arguments: map[string]interface{}{
				"language": "go",
			},
		}

		data, err := json.Marshal(params)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "code_review", parsed["name"])
		args := parsed["arguments"].(map[string]interface{})
		assert.Equal(t, "go", args["language"])
	})
}

func TestStreamableHTTPClient_SessionManagement(t *testing.T) {
	log := logger.NewNopLogger()
	client := NewStreamableHTTPClient(log, 30*time.Second)

	t.Run("getSession returns nil for unknown server", func(t *testing.T) {
		session := client.getSession("unknown-server")
		assert.Nil(t, session)
	})

	t.Run("clearSession handles unknown server gracefully", func(t *testing.T) {
		// Should not panic
		client.clearSession("nonexistent-server")
	})
}

func TestConstants(t *testing.T) {
	t.Run("MCP protocol version is correct", func(t *testing.T) {
		assert.Equal(t, "2025-11-25", MCPProtocolVersion)
	})

	t.Run("header names are correct", func(t *testing.T) {
		assert.Equal(t, "MCP-Protocol-Version", HeaderMCPProtocolVersion)
		assert.Equal(t, "MCP-Session-Id", HeaderMCPSessionID)
		assert.Equal(t, "Accept", HeaderAccept)
		assert.Equal(t, "Content-Type", HeaderContentType)
	})

	t.Run("content types are correct", func(t *testing.T) {
		assert.Equal(t, "application/json", ContentTypeJSON)
		assert.Equal(t, "text/event-stream", ContentTypeEventStream)
	})
}

func TestRewriteProxyPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		serverID string
		wantPath string
	}{
		{
			name:     "tools/list endpoint",
			path:     "/api/v1/gateway/server-123/tools/list",
			serverID: "server-123",
			wantPath: "/tools/list",
		},
		{
			name:     "tools/call endpoint",
			path:     "/api/v1/gateway/abc-def/tools/call",
			serverID: "abc-def",
			wantPath: "/tools/call",
		},
		{
			name:     "initialize endpoint",
			path:     "/api/v1/gateway/test-id/initialize",
			serverID: "test-id",
			wantPath: "/initialize",
		},
		{
			name:     "resources/list endpoint",
			path:     "/api/v1/gateway/my-server/resources/list",
			serverID: "my-server",
			wantPath: "/resources/list",
		},
		{
			name:     "path with query params",
			path:     "/api/v1/gateway/srv1/tools/list?filter=calc",
			serverID: "srv1",
			wantPath: "/tools/list?filter=calc",
		},
		{
			name:     "prompts/list endpoint",
			path:     "/api/v1/gateway/server-xyz/prompts/list",
			serverID: "server-xyz",
			wantPath: "/prompts/list",
		},
		{
			name:     "resources/read endpoint",
			path:     "/api/v1/gateway/srv-abc/resources/read",
			serverID: "srv-abc",
			wantPath: "/resources/read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath := rewriteProxyPath(tt.path, tt.serverID)
			assert.Equal(t, tt.wantPath, gotPath, "path should be correctly rewritten")
		})
	}
}

func TestRewriteProxyPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		serverID string
		wantPath string
	}{
		{
			name:     "server ID with dashes",
			path:     "/api/v1/gateway/my-test-server-123/tools/call",
			serverID: "my-test-server-123",
			wantPath: "/tools/call",
		},
		{
			name:     "server ID with underscores",
			path:     "/api/v1/gateway/test_server_1/initialize",
			serverID: "test_server_1",
			wantPath: "/initialize",
		},
		{
			name:     "nested path",
			path:     "/api/v1/gateway/server-123/some/nested/path",
			serverID: "server-123",
			wantPath: "/some/nested/path",
		},
		{
			name:     "path with multiple query params",
			path:     "/api/v1/gateway/srv1/tools/list?filter=calc&limit=10&offset=0",
			serverID: "srv1",
			wantPath: "/tools/list?filter=calc&limit=10&offset=0",
		},
		{
			name:     "path without gateway prefix returns unchanged",
			path:     "/some/other/path",
			serverID: "server-123",
			wantPath: "/some/other/path",
		},
		{
			name:     "path with different server ID returns unchanged",
			path:     "/api/v1/gateway/other-server/tools/list",
			serverID: "server-123",
			wantPath: "/api/v1/gateway/other-server/tools/list",
		},
		{
			name:     "empty path",
			path:     "",
			serverID: "server-123",
			wantPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath := rewriteProxyPath(tt.path, tt.serverID)
			assert.Equal(t, tt.wantPath, gotPath, "path should handle edge cases")
		})
	}
}

func TestIsSSEServer_PackageLevel(t *testing.T) {
	tests := []struct {
		server   *domain.MCPServer
		name     string
		expected bool
	}{
		{
			name: "SSE server with /mcp suffix",
			server: &domain.MCPServer{
				URL: "http://localhost:8080/mcp",
			},
			expected: true,
		},
		{
			name: "SSE server with /mcp/ suffix",
			server: &domain.MCPServer{
				URL: "http://localhost:8080/mcp/",
			},
			expected: false, // Needs exact /mcp suffix
		},
		{
			name: "non-SSE server",
			server: &domain.MCPServer{
				URL: "http://localhost:8080/api/v1",
			},
			expected: false,
		},
		{
			name: "empty URL",
			server: &domain.MCPServer{
				URL: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSSEServer(tt.server)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStreamableHTTPServer_PackageLevel(t *testing.T) {
	tests := []struct {
		server   *domain.MCPServer
		name     string
		expected bool
	}{
		{
			name:     "Streamable HTTP server with /mcp suffix",
			server:   &domain.MCPServer{URL: "http://localhost:8080/mcp"},
			expected: true,
		},
		{
			name:     "non-Streamable HTTP server",
			server:   &domain.MCPServer{URL: "http://localhost:8080/api"},
			expected: false,
		},
		{
			name:     "Streamable HTTP server with subpath",
			server:   &domain.MCPServer{URL: "http://localhost:8080/servers/123/mcp"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStreamableHTTPServer(tt.server)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewService_NilDependencies(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("creates service with nil repository", func(t *testing.T) {
		svc := NewService(nil, log, nil)
		require.NotNil(t, svc)
	})

	t.Run("creates service with nil logger", func(t *testing.T) {
		svc := NewService(nil, nil, nil)
		require.NotNil(t, svc)
	})
}

func TestSSEClient_ParseJSONResponse(t *testing.T) {
	log := logger.NewNopLogger()
	client := NewSSEClient(log, 30*time.Second)

	tests := []struct {
		name        string
		body        string
		errContains string
		wantResult  bool
		wantErr     bool
	}{
		{
			name:       "valid JSON-RPC response with result",
			body:       `{"jsonrpc":"2.0","result":{"tools":[]},"id":1}`,
			wantResult: true,
			wantErr:    false,
		},
		{
			name:        "JSON-RPC error response",
			body:        `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":1}`,
			wantResult:  false,
			wantErr:     true,
			errContains: "MCP error -32600",
		},
		{
			name:        "invalid JSON",
			body:        `{invalid json}`,
			wantResult:  false,
			wantErr:     true,
			errContains: "failed to parse JSON-RPC response",
		},
		{
			name:       "null result",
			body:       `{"jsonrpc":"2.0","result":null,"id":1}`,
			wantResult: false, // null result is valid but empty
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseJSONResponse(strings.NewReader(tt.body))
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				if tt.wantResult {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

func TestSSEClient_ParseSSEResponse(t *testing.T) {
	log := logger.NewNopLogger()
	client := NewSSEClient(log, 30*time.Second)

	tests := []struct {
		name        string
		body        string
		errContains string
		wantResult  bool
		wantErr     bool
	}{
		{
			name:       "valid SSE response with data",
			body:       "event: message\ndata: {\"jsonrpc\":\"2.0\",\"result\":{\"tools\":[]},\"id\":1}\n\n",
			wantResult: true,
			wantErr:    false,
		},
		{
			name:        "SSE response with no data",
			body:        "event: message\n\n",
			wantResult:  false,
			wantErr:     true,
			errContains: "no data received",
		},
		{
			name:        "SSE response with JSON-RPC error",
			body:        "data: {\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32601,\"message\":\"Method not found\"},\"id\":1}\n\n",
			wantResult:  false,
			wantErr:     true,
			errContains: "MCP error -32601",
		},
		{
			name:        "SSE response with invalid JSON",
			body:        "data: {not valid json}\n\n",
			wantResult:  false,
			wantErr:     true,
			errContains: "failed to parse JSON-RPC response",
		},
		{
			name:       "SSE response with multiple events",
			body:       "event: message\ndata: {\"jsonrpc\":\"2.0\",\"result\":{\"first\":true},\"id\":1}\n\nevent: message\ndata: {\"jsonrpc\":\"2.0\",\"result\":{\"last\":true},\"id\":2}\n\n",
			wantResult: true, // Should parse the last one
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseSSEResponse(strings.NewReader(tt.body))
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				if tt.wantResult {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

func TestSSEClient_InjectAuth(t *testing.T) {
	log := logger.NewNopLogger()
	client := NewSSEClient(log, 30*time.Second)

	tests := []struct {
		name           string
		server         *domain.MCPServer
		expectedHeader string
		expectedValue  string
	}{
		{
			name: "no auth config",
			server: &domain.MCPServer{
				ID:         "server-1",
				AuthConfig: nil,
			},
			expectedHeader: "",
			expectedValue:  "",
		},
		{
			name: "bearer token auth",
			server: &domain.MCPServer{
				ID:         "server-2",
				AuthType:   domain.ServerAuthBearer,
				AuthConfig: json.RawMessage(`{"token":"test-token-123"}`),
			},
			expectedHeader: "Authorization",
			expectedValue:  "Bearer test-token-123",
		},
		{
			name: "basic auth",
			server: &domain.MCPServer{
				ID:         "server-3",
				AuthType:   domain.ServerAuthBasic,
				AuthConfig: json.RawMessage(`{"username":"user","password":"pass"}`),
			},
			expectedHeader: "Authorization",
			expectedValue:  "Basic dXNlcjpwYXNz", // base64(user:pass)
		},
		{
			name: "invalid auth config JSON",
			server: &domain.MCPServer{
				ID:         "server-4",
				AuthType:   domain.ServerAuthBearer,
				AuthConfig: json.RawMessage(`{invalid json}`),
			},
			expectedHeader: "",
			expectedValue:  "",
		},
		{
			name: "empty auth config",
			server: &domain.MCPServer{
				ID:         "server-5",
				AuthConfig: json.RawMessage(`{}`),
			},
			expectedHeader: "",
			expectedValue:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "http://example.com", nil)
			client.injectAuth(req, tt.server)
			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedValue, req.Header.Get(tt.expectedHeader))
			} else {
				assert.Empty(t, req.Header.Get("Authorization"))
			}
		})
	}
}

func TestStreamableHTTPClient_ParseJSONResponse(t *testing.T) {
	log := logger.NewNopLogger()
	client := NewStreamableHTTPClient(log, 30*time.Second)

	tests := []struct {
		name        string
		body        string
		errContains string
		wantResult  bool
		wantErr     bool
	}{
		{
			name:       "valid JSON-RPC response",
			body:       `{"jsonrpc":"2.0","result":{"capabilities":{}},"id":1}`,
			wantResult: true,
			wantErr:    false,
		},
		{
			name:        "JSON-RPC error response",
			body:        `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":1}`,
			wantResult:  false,
			wantErr:     true,
			errContains: "MCP error -32700",
		},
		{
			name:        "invalid JSON",
			body:        `not valid json`,
			wantResult:  false,
			wantErr:     true,
			errContains: "failed to parse JSON-RPC response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := client.parseJSONResponse(strings.NewReader(tt.body))
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				if tt.wantResult {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

func TestStreamableHTTPClient_ParseSSEStream(t *testing.T) {
	log := logger.NewNopLogger()
	client := NewStreamableHTTPClient(log, 30*time.Second)

	tests := []struct {
		name        string
		body        string
		errContains string
		wantResult  bool
		wantErr     bool
	}{
		{
			name:       "valid SSE stream with data",
			body:       "event: message\ndata: {\"jsonrpc\":\"2.0\",\"result\":{\"tools\":[]},\"id\":1}\n\n",
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "SSE stream with event ID",
			body:       "id: event-123\ndata: {\"jsonrpc\":\"2.0\",\"result\":{},\"id\":1}\n\n",
			wantResult: true,
			wantErr:    false,
		},
		{
			name:        "SSE stream with no data",
			body:        "event: message\n\n",
			wantResult:  false,
			wantErr:     true,
			errContains: "no data received",
		},
		{
			name:        "SSE stream with JSON-RPC error",
			body:        "data: {\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32602,\"message\":\"Invalid params\"},\"id\":1}\n\n",
			wantResult:  false,
			wantErr:     true,
			errContains: "MCP error -32602",
		},
		{
			name:       "SSE stream with multiple events uses last",
			body:       "data: {\"jsonrpc\":\"2.0\",\"result\":{\"first\":true},\"id\":1}\n\ndata: {\"jsonrpc\":\"2.0\",\"result\":{\"last\":true},\"id\":2}\n\n",
			wantResult: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := client.parseSSEStream(strings.NewReader(tt.body))
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				if tt.wantResult {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

func TestStreamableHTTPClient_InjectAuth(t *testing.T) {
	log := logger.NewNopLogger()
	client := NewStreamableHTTPClient(log, 30*time.Second)

	tests := []struct {
		name           string
		server         *domain.MCPServer
		expectedHeader string
		expectedValue  string
	}{
		{
			name: "no auth config",
			server: &domain.MCPServer{
				ID:         "server-1",
				AuthConfig: nil,
			},
			expectedHeader: "",
			expectedValue:  "",
		},
		{
			name: "bearer token auth",
			server: &domain.MCPServer{
				ID:         "server-2",
				AuthType:   domain.ServerAuthBearer,
				AuthConfig: json.RawMessage(`{"token":"streamable-token-456"}`),
			},
			expectedHeader: "Authorization",
			expectedValue:  "Bearer streamable-token-456",
		},
		{
			name: "basic auth",
			server: &domain.MCPServer{
				ID:         "server-3",
				AuthType:   domain.ServerAuthBasic,
				AuthConfig: json.RawMessage(`{"username":"admin","password":"secret"}`),
			},
			expectedHeader: "Authorization",
			expectedValue:  "Basic YWRtaW46c2VjcmV0", // base64(admin:secret)
		},
		{
			name: "empty token",
			server: &domain.MCPServer{
				ID:         "server-4",
				AuthType:   domain.ServerAuthBearer,
				AuthConfig: json.RawMessage(`{"token":""}`),
			},
			expectedHeader: "",
			expectedValue:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "http://example.com", nil)
			client.injectAuth(req, tt.server)
			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedValue, req.Header.Get(tt.expectedHeader))
			} else {
				assert.Empty(t, req.Header.Get("Authorization"))
			}
		})
	}
}

// Tests using httptest for HTTP-based methods

func TestSSEClient_Call(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("successful call with JSON response", func(t *testing.T) {
		// Create mock SSE server that returns JSON
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/message", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"tools":[]},"id":1}`))
		}))
		defer ts.Close()

		client := NewSSEClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL, // URL without /message suffix
		}

		result, err := client.Call(context.Background(), server, "tools/list", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("call with authentication", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check that auth header is present
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		}))
		defer ts.Close()

		client := NewSSEClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:         "test-server",
			URL:        ts.URL,
			AuthType:   domain.ServerAuthBearer,
			AuthConfig: json.RawMessage(`{"token":"test-token"}`),
		}

		_, err := client.Call(context.Background(), server, "tools/list", nil)
		require.NoError(t, err)
	})

	t.Run("call returns server error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer ts.Close()

		client := NewSSEClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Call(context.Background(), server, "tools/list", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("call returns JSON-RPC error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}`))
		}))
		defer ts.Close()

		client := NewSSEClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Call(context.Background(), server, "unknown/method", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MCP error -32601")
	})

	t.Run("call with params", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			assert.Contains(t, string(body), `"params"`)
			assert.Contains(t, string(body), `"name":"test_tool"`)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":[]},"id":1}`))
		}))
		defer ts.Close()

		client := NewSSEClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		params := ToolsCallParams{
			Name:      "test_tool",
			Arguments: map[string]interface{}{"arg1": "value1"},
		}

		_, err := client.Call(context.Background(), server, "tools/call", params)
		require.NoError(t, err)
	})
}

func TestStreamableHTTPClient_Initialize(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("successful initialization", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				body, _ := io.ReadAll(r.Body)
				if strings.Contains(string(body), "initialize") {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("MCP-Session-Id", "session-123")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"jsonrpc":"2.0","result":{"protocolVersion":"2025-11-25","capabilities":{}},"id":1}`))
				} else if strings.Contains(string(body), "notifications/initialized") {
					w.WriteHeader(http.StatusAccepted)
				}
			}
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		session, err := client.Initialize(context.Background(), server)
		require.NoError(t, err)
		require.NotNil(t, session)
		// Note: Session ID from header is captured only for 202 responses in current implementation
		assert.True(t, session.Initialized)
		assert.Equal(t, server.ID, session.ServerID)
		assert.Equal(t, ts.URL, session.ServerURL)
	})

	t.Run("initialization failure", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Initialize(context.Background(), server)
		require.Error(t, err)
	})
}

func TestStreamableHTTPClient_TerminateSession(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("terminate existing session", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "DELETE" {
				assert.Equal(t, "session-123", r.Header.Get("MCP-Session-Id"))
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		// Add a session manually
		client.sessionsMu.Lock()
		client.sessions["test-server"] = &MCPSession{
			SessionID: "session-123",
			ServerID:  "test-server",
		}
		client.sessionsMu.Unlock()

		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		err := client.TerminateSession(context.Background(), server)
		require.NoError(t, err)

		// Verify session was cleared
		assert.Nil(t, client.getSession("test-server"))
	})

	t.Run("terminate non-existent session returns nil", func(t *testing.T) {
		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "non-existent-server",
			URL: "http://localhost:9999",
		}

		err := client.TerminateSession(context.Background(), server)
		require.NoError(t, err)
	})

	t.Run("server returns 405 method not allowed", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "DELETE" {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		client.sessionsMu.Lock()
		client.sessions["test-server"] = &MCPSession{
			SessionID: "session-456",
			ServerID:  "test-server",
		}
		client.sessionsMu.Unlock()

		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		err := client.TerminateSession(context.Background(), server)
		require.NoError(t, err) // 405 is acceptable
	})

	t.Run("server returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "DELETE" {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
			}
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		client.sessionsMu.Lock()
		client.sessions["test-server"] = &MCPSession{
			SessionID: "session-789",
			ServerID:  "test-server",
		}
		client.sessionsMu.Unlock()

		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		err := client.TerminateSession(context.Background(), server)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestStreamableHTTPClient_CallWithSSEResponse(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("call with SSE response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"result\":{\"tools\":[]},\"id\":1}\n\n"))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		// Add a session so Call doesn't try to reinitialize
		client.sessionsMu.Lock()
		client.sessions["test-server"] = &MCPSession{
			SessionID:   "session-test",
			ServerID:    "test-server",
			Initialized: true,
		}
		client.sessionsMu.Unlock()

		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		result, err := client.Call(context.Background(), server, "tools/list", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestStreamableHTTPClient_Call(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("call with JSON response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"data":"test"},"id":1}`))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		client.sessionsMu.Lock()
		client.sessions["test-server"] = &MCPSession{
			SessionID:   "session-abc",
			ServerID:    "test-server",
			Initialized: true,
		}
		client.sessionsMu.Unlock()

		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		result, err := client.Call(context.Background(), server, "test/method", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("call without existing session", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "new-server",
			URL: ts.URL,
		}

		// No session exists, Call should work without session
		result, err := client.Call(context.Background(), server, "test/method", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("call with session ID update", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("MCP-Session-Id", "new-session-id")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		client.sessionsMu.Lock()
		client.sessions["test-server"] = &MCPSession{
			SessionID:   "old-session-id",
			ServerID:    "test-server",
			Initialized: true,
		}
		client.sessionsMu.Unlock()

		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Call(context.Background(), server, "test/method", nil)
		require.NoError(t, err)
	})
}

func TestStreamableHTTPClient_CallWithSessionHandling(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("400 bad request response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing session ID"))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Call(context.Background(), server, "test/method", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("404 not found response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Session not found"))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Call(context.Background(), server, "test/method", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})

	t.Run("500 internal server error response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Call(context.Background(), server, "test/method", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("202 accepted response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("MCP-Session-Id", "session-456")
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		// 202 Accepted returns nil result which is valid for notifications
		result, err := client.Call(context.Background(), server, "notifications/canceled", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("call with authentication", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:         "test-server",
			URL:        ts.URL,
			AuthType:   domain.ServerAuthBearer,
			AuthConfig: json.RawMessage(`{"token":"my-token"}`),
		}

		_, err := client.Call(context.Background(), server, "test/method", nil)
		require.NoError(t, err)
	})

	t.Run("call with basic authentication", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "user", username)
			assert.Equal(t, "pass", password)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:         "test-server",
			URL:        ts.URL,
			AuthType:   domain.ServerAuthBasic,
			AuthConfig: json.RawMessage(`{"username":"user","password":"pass"}`),
		}

		_, err := client.Call(context.Background(), server, "test/method", nil)
		require.NoError(t, err)
	})

	t.Run("verifies MCP headers are set", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Contains(t, r.Header.Get("Accept"), "application/json")
			assert.Contains(t, r.Header.Get("Accept"), "text/event-stream")
			assert.Equal(t, "2025-11-25", r.Header.Get("MCP-Protocol-Version"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Call(context.Background(), server, "test/method", nil)
		require.NoError(t, err)
	})

	t.Run("includes session ID in header when available", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "my-session-id", r.Header.Get("MCP-Session-Id"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		}))
		defer ts.Close()

		client := NewStreamableHTTPClient(log, 30*time.Second)
		client.sessionsMu.Lock()
		client.sessions["test-server"] = &MCPSession{
			SessionID:   "my-session-id",
			ServerID:    "test-server",
			Initialized: true,
		}
		client.sessionsMu.Unlock()

		server := &domain.MCPServer{
			ID:  "test-server",
			URL: ts.URL,
		}

		_, err := client.Call(context.Background(), server, "test/method", nil)
		require.NoError(t, err)
	})
}

// Mock implementations for testing

type mockServerRepository struct {
	server *domain.MCPServer
	err    error
}

func (m *mockServerRepository) Get(ctx context.Context, id string) (*domain.MCPServer, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.server, nil
}

type mockSSEClient struct {
	err    error
	result json.RawMessage
}

func (m *mockSSEClient) Call(ctx context.Context, server *domain.MCPServer, method string, params interface{}) (json.RawMessage, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.result, nil
}

type mockStreamableHTTPClient struct {
	callErr         error
	initErr         error
	terminateErr    error
	initSession     *MCPSession
	callResult      json.RawMessage
	terminateCalled bool
}

func (m *mockStreamableHTTPClient) Call(ctx context.Context, server *domain.MCPServer, method string, params interface{}) (json.RawMessage, error) {
	if m.callErr != nil {
		return nil, m.callErr
	}

	return m.callResult, nil
}

func (m *mockStreamableHTTPClient) Initialize(ctx context.Context, server *domain.MCPServer) (*MCPSession, error) {
	if m.initErr != nil {
		return nil, m.initErr
	}

	return m.initSession, nil
}

func (m *mockStreamableHTTPClient) TerminateSession(ctx context.Context, server *domain.MCPServer) error {
	m.terminateCalled = true

	return m.terminateErr
}

// Service method tests using mocks

func TestNewServiceWithClients(t *testing.T) {
	mockRepo := &mockServerRepository{}
	mockSSE := &mockSSEClient{}
	mockStreamable := &mockStreamableHTTPClient{}
	log := logger.NewNopLogger()

	svc := NewServiceWithClients(mockRepo, log, nil, mockSSE, mockStreamable)

	require.NotNil(t, svc)
	assert.NotNil(t, svc.repo)
	assert.NotNil(t, svc.sseClient)
	assert.NotNil(t, svc.streamableHTTPClient)
}

func TestService_ProxyToServer(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		proxy, server, err := svc.ProxyToServer(context.Background(), "unknown-server")

		assert.Error(t, err)
		assert.Nil(t, proxy)
		assert.Nil(t, server)
	})

	t.Run("returns error when server is inactive", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				Name:     "Test Server",
				URL:      "http://localhost:8080",
				IsActive: false,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		proxy, server, err := svc.ProxyToServer(context.Background(), "server-123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
		assert.Nil(t, proxy)
		assert.Nil(t, server)
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				Name:     "Test Server",
				URL:      "://invalid-url",
				IsActive: true,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		proxy, server, err := svc.ProxyToServer(context.Background(), "server-123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server URL")
		assert.Nil(t, proxy)
		assert.Nil(t, server)
	})

	t.Run("creates proxy for active server", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:             "server-123",
				Name:           "Test Server",
				URL:            "http://localhost:8080",
				IsActive:       true,
				MaxConnections: 10,
				TimeoutSeconds: 30,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		proxy, server, err := svc.ProxyToServer(context.Background(), "server-123")

		require.NoError(t, err)
		assert.NotNil(t, proxy)
		assert.NotNil(t, server)
		assert.Equal(t, "server-123", server.ID)
	})
}

func TestService_Initialize(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		server, err := svc.Initialize(context.Background(), "unknown-server")

		assert.Error(t, err)
		assert.Nil(t, server)
	})

	t.Run("returns error when server is inactive", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				IsActive: false,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		server, err := svc.Initialize(context.Background(), "server-123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
		assert.Nil(t, server)
	})

	t.Run("returns server for active server", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				Name:     "Test Server",
				URL:      "http://localhost:8080",
				IsActive: true,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		server, err := svc.Initialize(context.Background(), "server-123")

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "server-123", server.ID)
	})
}

func TestService_GetServerInfo(t *testing.T) {
	t.Run("returns server info", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:   "server-123",
				Name: "Test Server",
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		server, err := svc.GetServerInfo(context.Background(), "server-123")

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "server-123", server.ID)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		server, err := svc.GetServerInfo(context.Background(), "unknown")

		assert.Error(t, err)
		assert.Nil(t, server)
	})
}

func TestService_CallSSE(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		result, err := svc.CallSSE(context.Background(), "unknown", "tools/list", nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when server is inactive", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				IsActive: false,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		result, err := svc.CallSSE(context.Background(), "server-123", "tools/list", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
		assert.Nil(t, result)
	})

	t.Run("calls SSE client for active server", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				IsActive: true,
			},
		}
		expectedResult := json.RawMessage(`{"tools":[]}`)
		mockSSE := &mockSSEClient{
			result: expectedResult,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, mockSSE, nil)

		result, err := svc.CallSSE(context.Background(), "server-123", "tools/list", nil)

		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestService_IsSSEServer(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		isSSE, server, err := svc.IsSSEServer(context.Background(), "unknown")

		assert.Error(t, err)
		assert.False(t, isSSE)
		assert.Nil(t, server)
	})

	t.Run("returns true for SSE server", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:  "server-123",
				URL: "http://localhost:8080/mcp",
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		isSSE, server, err := svc.IsSSEServer(context.Background(), "server-123")

		require.NoError(t, err)
		assert.True(t, isSSE)
		assert.NotNil(t, server)
	})

	t.Run("returns false for non-SSE server", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:  "server-123",
				URL: "http://localhost:8080/api",
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		isSSE, server, err := svc.IsSSEServer(context.Background(), "server-123")

		require.NoError(t, err)
		assert.False(t, isSSE)
		assert.NotNil(t, server)
	})
}

func TestService_IsStreamableHTTPServer(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		isStreamable, server, err := svc.IsStreamableHTTPServer(context.Background(), "unknown")

		assert.Error(t, err)
		assert.False(t, isStreamable)
		assert.Nil(t, server)
	})

	t.Run("returns true for Streamable HTTP server", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:  "server-123",
				URL: "http://localhost:8080/mcp",
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		isStreamable, server, err := svc.IsStreamableHTTPServer(context.Background(), "server-123")

		require.NoError(t, err)
		assert.True(t, isStreamable)
		assert.NotNil(t, server)
	})
}

func TestService_CallStreamableHTTP(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		result, err := svc.CallStreamableHTTP(context.Background(), "unknown", "tools/list", nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when server is inactive", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				IsActive: false,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		result, err := svc.CallStreamableHTTP(context.Background(), "server-123", "tools/list", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
		assert.Nil(t, result)
	})

	t.Run("calls Streamable HTTP client for active server", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				IsActive: true,
			},
		}
		expectedResult := json.RawMessage(`{"tools":[]}`)
		mockStreamable := &mockStreamableHTTPClient{
			callResult: expectedResult,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, mockStreamable)

		result, err := svc.CallStreamableHTTP(context.Background(), "server-123", "tools/list", nil)

		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestService_InitializeStreamableHTTP(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		session, err := svc.InitializeStreamableHTTP(context.Background(), "unknown")

		assert.Error(t, err)
		assert.Nil(t, session)
	})

	t.Run("returns error when server is inactive", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				IsActive: false,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		session, err := svc.InitializeStreamableHTTP(context.Background(), "server-123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
		assert.Nil(t, session)
	})

	t.Run("initializes session for active server", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:       "server-123",
				IsActive: true,
				URL:      "http://localhost:8080/mcp",
			},
		}
		expectedSession := &MCPSession{
			SessionID: "sess-123",
			ServerID:  "server-123",
		}
		mockStreamable := &mockStreamableHTTPClient{
			initSession: expectedSession,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, mockStreamable)

		session, err := svc.InitializeStreamableHTTP(context.Background(), "server-123")

		require.NoError(t, err)
		assert.Equal(t, expectedSession, session)
	})
}

func TestService_TerminateStreamableHTTP(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		err := svc.TerminateStreamableHTTP(context.Background(), "unknown")

		assert.Error(t, err)
	})

	t.Run("terminates session successfully", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:  "server-123",
				URL: "http://localhost:8080/mcp",
			},
		}
		mockStreamable := &mockStreamableHTTPClient{}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, mockStreamable)

		err := svc.TerminateStreamableHTTP(context.Background(), "server-123")

		require.NoError(t, err)
		assert.True(t, mockStreamable.terminateCalled)
	})
}

func TestService_GetTransportType(t *testing.T) {
	t.Run("returns error when server not found", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			err: domain.ErrNotFound,
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		transport, server, err := svc.GetTransportType(context.Background(), "unknown")

		assert.Error(t, err)
		assert.Empty(t, transport)
		assert.Nil(t, server)
	})

	t.Run("returns explicit transport type", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:        "server-123",
				URL:       "http://localhost:8080/api",
				Transport: domain.TransportSSE,
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		transport, server, err := svc.GetTransportType(context.Background(), "server-123")

		require.NoError(t, err)
		assert.Equal(t, domain.TransportSSE, transport)
		assert.NotNil(t, server)
	})

	t.Run("auto-detects Streamable HTTP transport", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:  "server-123",
				URL: "http://localhost:8080/mcp",
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		transport, server, err := svc.GetTransportType(context.Background(), "server-123")

		require.NoError(t, err)
		assert.Equal(t, domain.TransportStreamableHTTP, transport)
		assert.NotNil(t, server)
	})

	t.Run("defaults to HTTP transport", func(t *testing.T) {
		mockRepo := &mockServerRepository{
			server: &domain.MCPServer{
				ID:  "server-123",
				URL: "http://localhost:8080/api",
			},
		}
		svc := NewServiceWithClients(mockRepo, logger.NewNopLogger(), nil, nil, nil)

		transport, server, err := svc.GetTransportType(context.Background(), "server-123")

		require.NoError(t, err)
		assert.Equal(t, domain.TransportHTTP, transport)
		assert.NotNil(t, server)
	})
}

func TestService_InjectAuth(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("no auth config", func(t *testing.T) {
		svc := NewServiceWithClients(nil, log, nil, nil, nil)
		req := httptest.NewRequest("GET", "/test", nil)
		server := &domain.MCPServer{
			ID:         "server-123",
			AuthConfig: nil,
		}

		svc.injectAuth(req, server)

		assert.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("empty auth config", func(t *testing.T) {
		svc := NewServiceWithClients(nil, log, nil, nil, nil)
		req := httptest.NewRequest("GET", "/test", nil)
		server := &domain.MCPServer{
			ID:         "server-123",
			AuthConfig: json.RawMessage(`{}`),
			AuthType:   domain.ServerAuthNone,
		}

		svc.injectAuth(req, server)

		assert.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("bearer token auth", func(t *testing.T) {
		svc := NewServiceWithClients(nil, log, nil, nil, nil)
		req := httptest.NewRequest("GET", "/test", nil)
		server := &domain.MCPServer{
			ID:         "server-123",
			AuthType:   domain.ServerAuthBearer,
			AuthConfig: json.RawMessage(`{"token":"my-secret-token"}`),
		}

		svc.injectAuth(req, server)

		assert.Equal(t, "Bearer my-secret-token", req.Header.Get("Authorization"))
	})

	t.Run("basic auth", func(t *testing.T) {
		svc := NewServiceWithClients(nil, log, nil, nil, nil)
		req := httptest.NewRequest("GET", "/test", nil)
		server := &domain.MCPServer{
			ID:         "server-123",
			AuthType:   domain.ServerAuthBasic,
			AuthConfig: json.RawMessage(`{"username":"user","password":"pass"}`),
		}

		svc.injectAuth(req, server)

		user, pass, ok := req.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", user)
		assert.Equal(t, "pass", pass)
	})

	t.Run("invalid auth config JSON", func(t *testing.T) {
		svc := NewServiceWithClients(nil, log, nil, nil, nil)
		req := httptest.NewRequest("GET", "/test", nil)
		server := &domain.MCPServer{
			ID:         "server-123",
			AuthType:   domain.ServerAuthBearer,
			AuthConfig: json.RawMessage(`{invalid`),
		}

		svc.injectAuth(req, server)

		assert.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("unknown auth type", func(t *testing.T) {
		svc := NewServiceWithClients(nil, log, nil, nil, nil)
		req := httptest.NewRequest("GET", "/test", nil)
		server := &domain.MCPServer{
			ID:         "server-123",
			AuthType:   "unknown-type",
			AuthConfig: json.RawMessage(`{"key":"value"}`),
		}

		svc.injectAuth(req, server)

		assert.Empty(t, req.Header.Get("Authorization"))
	})
}
