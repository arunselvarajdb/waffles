package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/internal/handler/middleware"
	"github.com/waffles/mcp-gateway/internal/service/gateway"
	"github.com/waffles/mcp-gateway/internal/service/serveraccess"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// MCPRequest represents a JSON-RPC request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolsListResult represents the result of tools/list
type ToolsListResult struct {
	Tools []MCPTool `json:"tools"`
}

// MCPTool represents an MCP tool
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ToolCallParams represents the params for tools/call
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// GatewayHandler handles MCP gateway requests
type GatewayHandler struct {
	service       GatewayServiceInterface
	accessService ServerAccessServiceInterface
	logger        logger.Logger
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(service *gateway.Service, accessService *serveraccess.Service, log logger.Logger) *GatewayHandler {
	var svc GatewayServiceInterface
	var accessSvc ServerAccessServiceInterface

	if service != nil {
		svc = &gatewayServiceAdapter{service: service}
	}
	if accessService != nil {
		accessSvc = accessService
	}

	return &GatewayHandler{
		service:       svc,
		accessService: accessSvc,
		logger:        log,
	}
}

// NewGatewayHandlerWithInterface creates a new gateway handler with interfaces (for testing).
func NewGatewayHandlerWithInterface(service GatewayServiceInterface, accessService ServerAccessServiceInterface, log logger.Logger) *GatewayHandler {
	return &GatewayHandler{
		service:       service,
		accessService: accessService,
		logger:        log,
	}
}

// gatewayServiceAdapter adapts gateway.Service to GatewayServiceInterface.
type gatewayServiceAdapter struct {
	service *gateway.Service
}

func (a *gatewayServiceAdapter) ProxyToServer(ctx context.Context, serverID string) (*httputil.ReverseProxy, *domain.MCPServer, error) {
	return a.service.ProxyToServer(ctx, serverID)
}

func (a *gatewayServiceAdapter) GetServerInfo(ctx context.Context, serverID string) (*domain.MCPServer, error) {
	return a.service.GetServerInfo(ctx, serverID)
}

func (a *gatewayServiceAdapter) Initialize(ctx context.Context, serverID string) (*domain.MCPServer, error) {
	return a.service.Initialize(ctx, serverID)
}

func (a *gatewayServiceAdapter) GetTransportType(ctx context.Context, serverID string) (domain.TransportType, *domain.MCPServer, error) {
	return a.service.GetTransportType(ctx, serverID)
}

func (a *gatewayServiceAdapter) CallSSE(ctx context.Context, serverID string, method string, params interface{}) (json.RawMessage, error) {
	return a.service.CallSSE(ctx, serverID, method, params)
}

func (a *gatewayServiceAdapter) CallStreamableHTTP(ctx context.Context, serverID string, method string, params interface{}) (json.RawMessage, error) {
	return a.service.CallStreamableHTTP(ctx, serverID, method, params)
}

func (a *gatewayServiceAdapter) InitializeStreamableHTTP(ctx context.Context, serverID string) (*MCPSession, error) {
	session, err := a.service.InitializeStreamableHTTP(ctx, serverID)
	if err != nil {
		return nil, err
	}

	return &MCPSession{
		SessionID:       session.SessionID,
		ProtocolVersion: session.ProtocolVersion,
	}, nil
}

func (a *gatewayServiceAdapter) TerminateStreamableHTTP(ctx context.Context, serverID string) error {
	return a.service.TerminateStreamableHTTP(ctx, serverID)
}

// ProxyRequest is a catch-all handler that proxies requests to MCP servers
func (h *GatewayHandler) ProxyRequest(c *gin.Context) {
	serverID := c.Param("server_id")

	// Get reverse proxy for this server
	proxy, server, err := h.service.ProxyToServer(c.Request.Context(), serverID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Str("path", c.Request.URL.Path).
			Msg("Failed to get proxy for server")

		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("server_id", serverID).
		Str("server_name", server.Name).
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Msg("Proxying request to MCP server")

	// Forward the request using the reverse proxy
	proxy.ServeHTTP(c.Writer, c.Request)
}

// MCPProxy handles native MCP protocol requests (Streamable HTTP transport)
// This endpoint allows MCP clients like Claude Code to connect directly via the gateway
// It intercepts requests to enforce tool filtering based on server's allowed_tools setting
func (h *GatewayHandler) MCPProxy(c *gin.Context) {
	serverID := c.Param("server_id")

	h.logger.Info().
		Str("server_id", serverID).
		Str("method", c.Request.Method).
		Str("content_type", c.GetHeader("Content-Type")).
		Str("accept", c.GetHeader("Accept")).
		Msg("MCP Proxy request received")

	// Check execute-level access if access service is configured
	if h.accessService != nil {
		roles := middleware.GetUserRoles(c)
		canExecute, err := h.accessService.CanAccessServer(c.Request.Context(), roles, serverID, domain.AccessLevelExecute)
		if err != nil {
			h.logger.Error().Err(err).Str("server_id", serverID).Any("roles", roles).Msg("Failed to check server execute access")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check server access",
			})
			return
		}
		if !canExecute {
			h.logger.Warn().Str("server_id", serverID).Any("roles", roles).Msg("Execute access denied to server")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You don't have execute permission for this server",
			})
			return
		}
	}

	// Get the server info to check allowed tools
	server, err := h.service.GetServerInfo(c.Request.Context(), serverID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Msg("Failed to get server info")

		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Check if server is active
	if !server.IsActive {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "server is inactive",
		})
		return
	}

	// If no tool filtering, use simple proxy
	if len(server.AllowedTools) == 0 {
		h.proxySimple(c, serverID, server)
		return
	}

	// Tool filtering is enabled - need to intercept and filter
	h.proxyWithToolFiltering(c, serverID, server)
}

// proxySimple forwards requests without any filtering
func (h *GatewayHandler) proxySimple(c *gin.Context, serverID string, server *domain.MCPServer) {
	proxy, _, err := h.service.ProxyToServer(c.Request.Context(), serverID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Msg("Failed to get proxy for server")

		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("server_id", serverID).
		Str("server_name", server.Name).
		Str("target_url", server.URL).
		Msg("Proxying MCP request (no filtering)")

	// Use defer/recover to handle panics from SSE connection closures gracefully
	defer func() {
		if r := recover(); r != nil {
			// SSE connections can cause panics when client disconnects
			// This is expected behavior, just log it at debug level
			h.logger.Debug().
				Str("server_id", serverID).
				Any("panic", r).
				Msg("Proxy connection closed (likely client disconnect)")
		}
	}()

	proxy.ServeHTTP(c.Writer, c.Request)
}

// proxyWithToolFiltering intercepts requests and filters tools based on allowed_tools
func (h *GatewayHandler) proxyWithToolFiltering(c *gin.Context, serverID string, server *domain.MCPServer) {
	// Read the request body to detect the method
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Parse the JSON-RPC request
	var mcpReq MCPRequest
	if err := json.Unmarshal(bodyBytes, &mcpReq); err != nil {
		// Not a valid JSON-RPC request, just proxy it
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		h.proxySimple(c, serverID, server)
		return
	}

	h.logger.Info().
		Str("server_id", serverID).
		Str("mcp_method", mcpReq.Method).
		Int("allowed_tools_count", len(server.AllowedTools)).
		Msg("Processing MCP request with tool filtering")

	// Check if this is a tools/call request - reject if tool not allowed
	if mcpReq.Method == "tools/call" {
		var params ToolCallParams
		if err := json.Unmarshal(mcpReq.Params, &params); err == nil {
			if !h.isToolAllowed(params.Name, server.AllowedTools) {
				h.logger.Warn().
					Str("server_id", serverID).
					Str("tool_name", params.Name).
					Msg("Tool call rejected - not in allowed list")

				// Return JSON-RPC error response
				c.Header("Content-Type", "text/event-stream")
				errorResp := MCPResponse{
					JSONRPC: "2.0",
					ID:      mcpReq.ID,
					Error: &MCPError{
						Code:    -32602,
						Message: fmt.Sprintf("Tool '%s' is not allowed on this server", params.Name),
					},
				}
				respBytes, _ := json.Marshal(errorResp)
				writeSSEEvent(c.Writer, respBytes)
				return
			}
		}
	}

	// Restore the request body for proxying
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// For tools/list, we need to intercept and filter the response
	if mcpReq.Method == "tools/list" {
		h.proxyToolsListWithFiltering(c, serverID, server, mcpReq)
		return
	}

	// For other methods, just proxy
	h.proxySimple(c, serverID, server)
}

// proxyToolsListWithFiltering handles tools/list by making a direct HTTP request,
// parsing the SSE response, filtering tools, and returning the filtered response
func (h *GatewayHandler) proxyToolsListWithFiltering(c *gin.Context, serverID string, server *domain.MCPServer, mcpReq MCPRequest) {
	// Build the tools/list request
	reqBody, _ := json.Marshal(mcpReq)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// Create request to backend server
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", server.URL, bytes.NewReader(reqBody))
	if err != nil {
		h.sendMCPError(c, mcpReq.ID, -32603, fmt.Sprintf("failed to create request: %v", err))
		return
	}

	// Copy relevant headers from original request
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Forward MCP-specific headers (session ID, protocol version)
	if sessionID := c.Request.Header.Get("MCP-Session-Id"); sessionID != "" {
		req.Header.Set("MCP-Session-Id", sessionID)
	}
	if protocolVersion := c.Request.Header.Get("MCP-Protocol-Version"); protocolVersion != "" {
		req.Header.Set("MCP-Protocol-Version", protocolVersion)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error().Err(err).Str("server_id", serverID).Msg("Failed to call tools/list")
		h.sendMCPError(c, mcpReq.ID, -32603, fmt.Sprintf("backend request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.sendMCPError(c, mcpReq.ID, -32603, fmt.Sprintf("backend returned %d: %s", resp.StatusCode, string(body)))
		return
	}

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.sendMCPError(c, mcpReq.ID, -32603, fmt.Sprintf("failed to read response: %v", err))
		return
	}

	// Parse the response - it may be SSE format or JSON
	var mcpResp MCPResponse
	contentType := resp.Header.Get("Content-Type")

	if strings.Contains(contentType, "text/event-stream") {
		// Parse SSE format
		mcpResp, err = h.parseSSEResponse(bodyBytes)
		if err != nil {
			h.logger.Error().Err(err).Str("server_id", serverID).Msg("Failed to parse SSE response")
			h.sendMCPError(c, mcpReq.ID, -32603, fmt.Sprintf("failed to parse SSE response: %v", err))
			return
		}
	} else {
		// Try parsing as JSON
		if err := json.Unmarshal(bodyBytes, &mcpResp); err != nil {
			h.logger.Error().Err(err).Str("server_id", serverID).Str("body", string(bodyBytes)).Msg("Failed to parse JSON response")
			h.sendMCPError(c, mcpReq.ID, -32603, fmt.Sprintf("failed to parse response: %v", err))
			return
		}
	}

	// Check if there was an error from the backend
	if mcpResp.Error != nil {
		h.sendMCPError(c, mcpReq.ID, mcpResp.Error.Code, mcpResp.Error.Message)
		return
	}

	// Parse tools from result
	var toolsResult ToolsListResult
	if err := json.Unmarshal(mcpResp.Result, &toolsResult); err != nil {
		h.logger.Warn().Err(err).Str("server_id", serverID).Msg("Failed to parse tools/list result, returning as-is")
		// Can't parse, return the original response
		c.Header("Content-Type", "text/event-stream")
		writeSSEEvent(c.Writer, bodyBytes)
		return
	}

	// Filter tools
	filteredTools := make([]MCPTool, 0)
	for _, tool := range toolsResult.Tools {
		if h.isToolAllowed(tool.Name, server.AllowedTools) {
			filteredTools = append(filteredTools, tool)
		}
	}

	h.logger.Info().
		Str("server_id", serverID).
		Int("total_tools", len(toolsResult.Tools)).
		Int("filtered_tools", len(filteredTools)).
		Msg("Filtered tools/list response")

	// Build filtered response
	filteredResult := ToolsListResult{Tools: filteredTools}
	filteredResultBytes, _ := json.Marshal(filteredResult)

	finalResp := MCPResponse{
		JSONRPC: "2.0",
		ID:      mcpReq.ID,
		Result:  filteredResultBytes,
	}
	respBytes, _ := json.Marshal(finalResp)

	c.Header("Content-Type", "text/event-stream")
	writeSSEEvent(c.Writer, respBytes)
}

// parseSSEResponse parses an SSE-formatted response to extract the JSON-RPC response
func (h *GatewayHandler) parseSSEResponse(body []byte) (MCPResponse, error) {
	var mcpResp MCPResponse
	scanner := bufio.NewScanner(bytes.NewReader(body))
	var lastData string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			if data != "" {
				lastData = data
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return mcpResp, fmt.Errorf("failed to scan SSE response: %w", err)
	}

	if lastData == "" {
		return mcpResp, fmt.Errorf("no data found in SSE response")
	}

	if err := json.Unmarshal([]byte(lastData), &mcpResp); err != nil {
		return mcpResp, fmt.Errorf("failed to parse JSON from SSE data: %w", err)
	}

	return mcpResp, nil
}

// writeSSEEvent writes an SSE event to the response writer.
// Write errors are intentionally ignored for SSE streams - once a client disconnects,
// writes will fail and there's no recovery action we can take.
func writeSSEEvent(w io.Writer, data []byte) {
	_, _ = w.Write([]byte("event: message\n")) // #nosec G104 -- SSE write errors intentionally ignored
	_, _ = w.Write([]byte("data: "))           // #nosec G104 -- SSE write errors intentionally ignored
	_, _ = w.Write(data)                       // #nosec G104 -- SSE write errors intentionally ignored
	_, _ = w.Write([]byte("\n\n"))             // #nosec G104 -- SSE write errors intentionally ignored
}

// sendMCPError sends a JSON-RPC error response in SSE format
func (h *GatewayHandler) sendMCPError(c *gin.Context, id interface{}, code int, message string) {
	c.Header("Content-Type", "text/event-stream")
	errorResp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
	respBytes, _ := json.Marshal(errorResp)
	writeSSEEvent(c.Writer, respBytes)
}

// isToolAllowed checks if a tool name is in the allowed list
func (h *GatewayHandler) isToolAllowed(toolName string, allowedTools []string) bool {
	for _, allowed := range allowedTools {
		if allowed == toolName {
			return true
		}
	}
	return false
}

// Initialize handles MCP initialize endpoint
func (h *GatewayHandler) Initialize(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := h.service.Initialize(c.Request.Context(), serverID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Msg("Initialization failed")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id":   server.ID,
		"server_name": server.Name,
		"url":         server.URL,
		"status":      "initialized",
	})
}

// ListTools handles tools/list requests (supports HTTP, SSE, and Streamable HTTP servers)
func (h *GatewayHandler) ListTools(c *gin.Context) {
	serverID := c.Param("server_id")

	transport, _, err := h.service.GetTransportType(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	switch transport {
	case domain.TransportStreamableHTTP:
		h.handleStreamableHTTPRequest(c, "tools/list", nil)
	case domain.TransportSSE:
		h.handleSSERequest(c, "tools/list", nil)
	default:
		h.ProxyRequest(c)
	}
}

// CallTool handles tools/call requests (supports HTTP, SSE, and Streamable HTTP servers)
func (h *GatewayHandler) CallTool(c *gin.Context) {
	serverID := c.Param("server_id")

	transport, _, err := h.service.GetTransportType(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// For non-HTTP transports, we need to parse the body
	if transport == domain.TransportStreamableHTTP || transport == domain.TransportSSE {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}

		var params map[string]interface{}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &params); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
				return
			}
		}

		if transport == domain.TransportStreamableHTTP {
			h.handleStreamableHTTPRequest(c, "tools/call", params)
		} else {
			h.handleSSERequest(c, "tools/call", params)
		}
		return
	}
	h.ProxyRequest(c)
}

// ListResources handles resources/list requests
func (h *GatewayHandler) ListResources(c *gin.Context) {
	serverID := c.Param("server_id")

	transport, _, err := h.service.GetTransportType(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	switch transport {
	case domain.TransportStreamableHTTP:
		h.handleStreamableHTTPRequest(c, "resources/list", nil)
	case domain.TransportSSE:
		h.handleSSERequest(c, "resources/list", nil)
	default:
		h.ProxyRequest(c)
	}
}

// ReadResource handles resources/read requests
func (h *GatewayHandler) ReadResource(c *gin.Context) {
	serverID := c.Param("server_id")

	transport, _, err := h.service.GetTransportType(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if transport == domain.TransportStreamableHTTP || transport == domain.TransportSSE {
		body, _ := io.ReadAll(c.Request.Body)
		var params map[string]interface{}
		if len(body) > 0 {
			_ = json.Unmarshal(body, &params) // #nosec G104 -- parse errors handled via empty params
		}
		if transport == domain.TransportStreamableHTTP {
			h.handleStreamableHTTPRequest(c, "resources/read", params)
		} else {
			h.handleSSERequest(c, "resources/read", params)
		}
		return
	}
	h.ProxyRequest(c)
}

// ListPrompts handles prompts/list requests
func (h *GatewayHandler) ListPrompts(c *gin.Context) {
	serverID := c.Param("server_id")

	transport, _, err := h.service.GetTransportType(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	switch transport {
	case domain.TransportStreamableHTTP:
		h.handleStreamableHTTPRequest(c, "prompts/list", nil)
	case domain.TransportSSE:
		h.handleSSERequest(c, "prompts/list", nil)
	default:
		h.ProxyRequest(c)
	}
}

// GetPrompt handles prompts/get requests
func (h *GatewayHandler) GetPrompt(c *gin.Context) {
	serverID := c.Param("server_id")

	transport, _, err := h.service.GetTransportType(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if transport == domain.TransportStreamableHTTP || transport == domain.TransportSSE {
		body, _ := io.ReadAll(c.Request.Body)
		var params map[string]interface{}
		if len(body) > 0 {
			_ = json.Unmarshal(body, &params) // #nosec G104 -- parse errors handled via empty params
		}
		if transport == domain.TransportStreamableHTTP {
			h.handleStreamableHTTPRequest(c, "prompts/get", params)
		} else {
			h.handleSSERequest(c, "prompts/get", params)
		}
		return
	}
	h.ProxyRequest(c)
}

// handleSSERequest handles requests to SSE-based MCP servers (legacy)
func (h *GatewayHandler) handleSSERequest(c *gin.Context, method string, params interface{}) {
	serverID := c.Param("server_id")

	result, err := h.service.CallSSE(c.Request.Context(), serverID, method, params)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Str("method", method).
			Msg("SSE request failed")

		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Return the raw JSON result
	c.Data(http.StatusOK, "application/json", result)
}

// handleStreamableHTTPRequest handles requests to Streamable HTTP MCP servers (MCP 2025-11-25)
func (h *GatewayHandler) handleStreamableHTTPRequest(c *gin.Context, method string, params interface{}) {
	serverID := c.Param("server_id")

	result, err := h.service.CallStreamableHTTP(c.Request.Context(), serverID, method, params)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Str("method", method).
			Msg("Streamable HTTP request failed")

		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Return the raw JSON result
	c.Data(http.StatusOK, "application/json", result)
}

// InitializeStreamableHTTP initializes a Streamable HTTP MCP session
func (h *GatewayHandler) InitializeStreamableHTTP(c *gin.Context) {
	serverID := c.Param("server_id")

	session, err := h.service.InitializeStreamableHTTP(c.Request.Context(), serverID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Msg("Streamable HTTP initialization failed")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id":        serverID,
		"session_id":       session.SessionID,
		"protocol_version": session.ProtocolVersion,
		"status":           "initialized",
	})
}

// TerminateStreamableHTTP terminates a Streamable HTTP MCP session
func (h *GatewayHandler) TerminateStreamableHTTP(c *gin.Context) {
	serverID := c.Param("server_id")

	err := h.service.TerminateStreamableHTTP(c.Request.Context(), serverID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Msg("Streamable HTTP session termination failed")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id": serverID,
		"status":    "terminated",
	})
}
