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
	"sync"
	"sync/atomic"
	"time"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

const (
	// MCPProtocolVersion is the MCP protocol version supported
	MCPProtocolVersion = "2025-11-25"

	// Header names
	HeaderMCPProtocolVersion = "MCP-Protocol-Version"
	HeaderMCPSessionID       = "MCP-Session-Id"
	HeaderAccept             = "Accept"
	HeaderContentType        = "Content-Type"

	// Content types
	ContentTypeJSON        = "application/json"
	ContentTypeEventStream = "text/event-stream"
)

// StreamableHTTPClient handles communication with MCP servers using the Streamable HTTP transport
// Per MCP spec 2025-11-25: https://modelcontextprotocol.io/specification/2025-11-25/basic/transports
type StreamableHTTPClient struct {
	httpClient *http.Client
	logger     logger.Logger
	requestID  atomic.Int64

	// Session management per server
	sessions   map[string]*MCPSession
	sessionsMu sync.RWMutex
}

// MCPSession represents an MCP session with a server
type MCPSession struct {
	SessionID       string
	ServerID        string
	ServerURL       string
	Initialized     bool
	ProtocolVersion string
	LastEventID     string
	CreatedAt       time.Time
	mu              sync.RWMutex
}

// NewStreamableHTTPClient creates a new Streamable HTTP MCP client
func NewStreamableHTTPClient(log logger.Logger, timeout time.Duration) *StreamableHTTPClient {
	return &StreamableHTTPClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger:   log,
		sessions: make(map[string]*MCPSession),
	}
}

// Initialize sends an initialize request to establish an MCP session
func (c *StreamableHTTPClient) Initialize(ctx context.Context, server *domain.MCPServer) (*MCPSession, error) {
	c.logger.Info().
		Str("server_id", server.ID).
		Str("url", server.URL).
		Msg("Initializing MCP session with Streamable HTTP transport")

	// Build initialize request
	params := InitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		ClientInfo: ClientInfo{
			Name:    "mcp-gateway",
			Version: "1.0.0",
		},
	}

	result, sessionID, err := c.callWithSessionHandling(ctx, server, "", "initialize", params)
	if err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	// Create session
	session := &MCPSession{
		SessionID:       sessionID,
		ServerID:        server.ID,
		ServerURL:       server.URL,
		Initialized:     true,
		ProtocolVersion: MCPProtocolVersion,
		CreatedAt:       time.Now(),
	}

	// Store session
	c.sessionsMu.Lock()
	c.sessions[server.ID] = session
	c.sessionsMu.Unlock()

	c.logger.Info().
		Str("server_id", server.ID).
		Str("session_id", sessionID).
		Str("result", string(result)).
		Msg("MCP session initialized")

	// Send initialized notification
	_, _, err = c.callWithSessionHandling(ctx, server, sessionID, "notifications/initialized", nil)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Failed to send initialized notification")
		// Don't fail - some servers may not require this
	}

	return session, nil
}

// Call sends a JSON-RPC request to an MCP server and returns the response
func (c *StreamableHTTPClient) Call(ctx context.Context, server *domain.MCPServer, method string, params interface{}) (json.RawMessage, error) {
	// Get or create session
	session := c.getSession(server.ID)
	sessionID := ""
	if session != nil {
		sessionID = session.SessionID
	}

	result, newSessionID, err := c.callWithSessionHandling(ctx, server, sessionID, method, params)
	if err != nil {
		// Check if session expired (404)
		if strings.Contains(err.Error(), "404") {
			c.logger.Info().Str("server_id", server.ID).Msg("Session expired, reinitializing")
			c.clearSession(server.ID)

			// Re-initialize and retry
			_, err = c.Initialize(ctx, server)
			if err != nil {
				return nil, fmt.Errorf("failed to reinitialize session: %w", err)
			}
			return c.Call(ctx, server, method, params)
		}
		return nil, err
	}

	// Update session ID if changed
	if newSessionID != "" && session != nil && newSessionID != session.SessionID {
		session.mu.Lock()
		session.SessionID = newSessionID
		session.mu.Unlock()
	}

	return result, nil
}

// callWithSessionHandling performs the actual HTTP request with session management
func (c *StreamableHTTPClient) callWithSessionHandling(
	ctx context.Context,
	server *domain.MCPServer,
	sessionID string,
	method string,
	params interface{},
) (json.RawMessage, string, error) {
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
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.Debug().
		Str("server_id", server.ID).
		Str("method", method).
		Int("request_id", int(reqID)).
		Str("session_id", sessionID).
		Msg("Sending Streamable HTTP MCP request")

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers per MCP spec 2025-11-25
	req.Header.Set(HeaderContentType, ContentTypeJSON)
	req.Header.Set(HeaderAccept, ContentTypeJSON+", "+ContentTypeEventStream)
	req.Header.Set(HeaderMCPProtocolVersion, MCPProtocolVersion)

	// Add session ID if we have one
	if sessionID != "" {
		req.Header.Set(HeaderMCPSessionID, sessionID)
	}

	// Add authentication if configured
	c.injectAuth(req, server)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Get session ID from response (may be set during initialize)
	respSessionID := resp.Header.Get(HeaderMCPSessionID)

	// Handle response based on status code
	switch resp.StatusCode {
	case http.StatusOK:
		// Success - parse response based on content type
		contentType := resp.Header.Get(HeaderContentType)
		if strings.Contains(contentType, ContentTypeEventStream) {
			return c.parseSSEStream(resp.Body)
		}
		return c.parseJSONResponse(resp.Body)

	case http.StatusAccepted:
		// 202 Accepted - for notifications/responses (no body expected)
		return nil, respSessionID, nil

	case http.StatusBadRequest:
		// Session ID missing or invalid
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("bad request (400): %s", string(body))

	case http.StatusNotFound:
		// Session expired
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("session not found (404): %s", string(body))

	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}
}

// parseJSONResponse parses a single JSON-RPC response
func (c *StreamableHTTPClient) parseJSONResponse(body io.Reader) (json.RawMessage, string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(data, &rpcResp); err != nil {
		return nil, "", fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, "", fmt.Errorf("MCP error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	c.logger.Debug().
		Any("id", rpcResp.ID).
		Msg("Received JSON MCP response")

	return rpcResp.Result, "", nil
}

// parseSSEStream parses an SSE stream and extracts the JSON-RPC response
func (c *StreamableHTTPClient) parseSSEStream(body io.Reader) (json.RawMessage, string, error) {
	scanner := bufio.NewScanner(body)
	var lastData string
	var lastEventID string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse SSE fields
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			if data != "" {
				lastData = data
			}
		} else if strings.HasPrefix(line, "id:") {
			lastEventID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
		// Empty line signals end of event
		// We accumulate data and process the last complete event
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("failed to read SSE stream: %w", err)
	}

	if lastData == "" {
		return nil, lastEventID, fmt.Errorf("no data received in SSE stream")
	}

	// Parse the last data as JSON-RPC response
	var rpcResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lastData), &rpcResp); err != nil {
		return nil, lastEventID, fmt.Errorf("failed to parse JSON-RPC response from SSE: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, lastEventID, fmt.Errorf("MCP error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	c.logger.Debug().
		Any("id", rpcResp.ID).
		Str("last_event_id", lastEventID).
		Msg("Received SSE MCP response")

	return rpcResp.Result, lastEventID, nil
}

// injectAuth adds authentication headers based on server config
func (c *StreamableHTTPClient) injectAuth(req *http.Request, server *domain.MCPServer) {
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

// getSession returns the session for a server if it exists
func (c *StreamableHTTPClient) getSession(serverID string) *MCPSession {
	c.sessionsMu.RLock()
	defer c.sessionsMu.RUnlock()
	return c.sessions[serverID]
}

// clearSession removes a session for a server
func (c *StreamableHTTPClient) clearSession(serverID string) {
	c.sessionsMu.Lock()
	defer c.sessionsMu.Unlock()
	delete(c.sessions, serverID)
}

// IsStreamableHTTPServer determines if a server uses Streamable HTTP transport
// Servers with URLs ending in "/mcp" are assumed to use Streamable HTTP
// This replaces the legacy SSE detection
func IsStreamableHTTPServer(server *domain.MCPServer) bool {
	return strings.HasSuffix(server.URL, "/mcp")
}

// TerminateSession sends a DELETE request to terminate an MCP session
func (c *StreamableHTTPClient) TerminateSession(ctx context.Context, server *domain.MCPServer) error {
	session := c.getSession(server.ID)
	if session == nil || session.SessionID == "" {
		return nil // No session to terminate
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", server.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create terminate request: %w", err)
	}

	req.Header.Set(HeaderMCPSessionID, session.SessionID)
	req.Header.Set(HeaderMCPProtocolVersion, MCPProtocolVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("terminate request failed: %w", err)
	}
	defer resp.Body.Close()

	c.clearSession(server.ID)

	// 405 Method Not Allowed is acceptable - server doesn't support client termination
	if resp.StatusCode == http.StatusMethodNotAllowed {
		c.logger.Debug().Str("server_id", server.ID).Msg("Server doesn't support client-initiated session termination")
		return nil
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("terminate failed with %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info().Str("server_id", server.ID).Msg("MCP session terminated")
	return nil
}
