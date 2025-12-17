package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// SSEClient handles communication with SSE-based MCP servers
type SSEClient struct {
	httpClient *http.Client
	logger     logger.Logger
	requestID  atomic.Int64
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int64       `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// NewSSEClient creates a new SSE MCP client
func NewSSEClient(log logger.Logger, timeout time.Duration) *SSEClient {
	return &SSEClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: log,
	}
}

// Call sends a JSON-RPC request to an SSE-based MCP server and returns the response
// For legacy SSE transport, messages are sent to /message endpoint (relative to SSE stream URL)
func (c *SSEClient) Call(ctx context.Context, server *domain.MCPServer, method string, params interface{}) (json.RawMessage, error) {
	reqID := c.requestID.Add(1)

	// Build JSON-RPC request
	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      reqID,
	}

	reqBody, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// For SSE transport, messages go to the /message endpoint
	// If URL is http://server/sse, messages go to http://server/sse/message
	messageURL := strings.TrimSuffix(server.URL, "/") + "/message"

	c.logger.Debug().
		Str("server_id", server.ID).
		Str("method", method).
		Str("message_url", messageURL).
		Int("request_id", int(reqID)).
		Msg("Sending SSE MCP request")

	// Create HTTP request to the message endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", messageURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// SSE-based MCP servers require these headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add authentication if configured
	c.injectAuth(req, server)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response (SSE message endpoint returns JSON, not SSE stream)
	return c.parseJSONResponse(resp.Body)
}

// parseJSONResponse parses a JSON-RPC response from the message endpoint
func (c *SSEClient) parseJSONResponse(body io.Reader) (json.RawMessage, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(data, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	c.logger.Debug().
		Any("id", rpcResp.ID).
		Msg("Received SSE MCP response")

	return rpcResp.Result, nil
}

// parseSSEResponse parses the SSE response format (for streaming responses)
// SSE format: "event: message\ndata: {...json...}\n\n"
func (c *SSEClient) parseSSEResponse(body io.Reader) (json.RawMessage, error) {
	scanner := bufio.NewScanner(body)
	var dataLine string

	for scanner.Scan() {
		line := scanner.Text()

		// Look for "data:" prefix
		if strings.HasPrefix(line, "data:") {
			dataLine = strings.TrimPrefix(line, "data:")
			dataLine = strings.TrimSpace(dataLine)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if dataLine == "" {
		return nil, fmt.Errorf("no data received in SSE response")
	}

	// Parse JSON-RPC response
	var rpcResp JSONRPCResponse
	if err := json.Unmarshal([]byte(dataLine), &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	c.logger.Debug().
		Any("id", rpcResp.ID).
		Msg("Received SSE MCP response")

	return rpcResp.Result, nil
}

// injectAuth adds authentication headers based on server config
func (c *SSEClient) injectAuth(req *http.Request, server *domain.MCPServer) {
	if len(server.AuthConfig) == 0 {
		return
	}

	var authConfig map[string]interface{}
	if err := json.Unmarshal(server.AuthConfig, &authConfig); err != nil {
		c.logger.Error().Err(err).Str("server_id", server.ID).Msg("Failed to parse auth config")
		return
	}

	switch server.AuthType {
	case domain.ServerAuthBearer:
		if token, ok := authConfig["token"].(string); ok && token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
	case domain.ServerAuthBasic:
		username, _ := authConfig["username"].(string)
		password, _ := authConfig["password"].(string)
		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
		}
	}
}

// IsSSEServer determines if a server uses SSE transport
// Servers with URLs ending in "/mcp" are assumed to be SSE-based
func IsSSEServer(server *domain.MCPServer) bool {
	return strings.HasSuffix(server.URL, "/mcp")
}

// ToolsListParams represents parameters for tools/list
type ToolsListParams struct{}

// ToolsCallParams represents parameters for tools/call
type ToolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ResourcesListParams represents parameters for resources/list
type ResourcesListParams struct{}

// ResourcesReadParams represents parameters for resources/read
type ResourcesReadParams struct {
	URI string `json:"uri"`
}

// PromptsListParams represents parameters for prompts/list
type PromptsListParams struct{}

// PromptsGetParams represents parameters for prompts/get
type PromptsGetParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// InitializeParams represents parameters for initialize
type InitializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	ClientInfo      ClientInfo `json:"clientInfo"`
	Capabilities    struct{}   `json:"capabilities,omitempty"`
}

// ClientInfo represents MCP client info
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
