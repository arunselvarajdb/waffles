package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/repository"
	"github.com/waffles/waffles/pkg/logger"
)

// Service handles MCP server registry business logic
type Service struct {
	repo   *repository.ServerRepository
	logger logger.Logger
}

// NewService creates a new registry service
func NewService(repo *repository.ServerRepository, log logger.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: log,
	}
}

// CreateServer registers a new MCP server
func (s *Service) CreateServer(ctx context.Context, req *domain.ServerCreate) (*domain.MCPServer, error) {
	// Set defaults if not provided
	if req.ProtocolVersion == "" {
		req.ProtocolVersion = "1.0.0"
	}
	if req.HealthCheckInterval == 0 {
		req.HealthCheckInterval = 60 // Default: 60 seconds
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 30 // Default: 30 seconds
	}
	if req.MaxConnections == 0 {
		req.MaxConnections = 100 // Default: 100 connections
	}

	// Create server in database
	server, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("server_id", server.ID).
		Str("name", server.Name).
		Msg("MCP server registered")

	// Trigger initial health check asynchronously
	go func() {
		healthCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.CheckHealth(healthCtx, server.ID); err != nil {
			s.logger.Warn().Err(err).Str("server_id", server.ID).Msg("Initial health check failed")
		}
	}()

	return server, nil
}

// ListServers retrieves all MCP servers with filtering
func (s *Service) ListServers(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
	servers, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	s.logger.Debug().Int("count", len(servers)).Msg("Servers listed")
	return servers, nil
}

// ListServersForUser retrieves MCP servers filtered by accessible server IDs
// If accessibleServerIDs is nil, returns all servers (admin bypass)
// If accessibleServerIDs is empty slice, returns no servers
// Otherwise, filters servers by the provided IDs
func (s *Service) ListServersForUser(ctx context.Context, filter *domain.ServerFilter, accessibleServerIDs []string) ([]*domain.MCPServer, error) {
	servers, err := s.repo.ListForUser(ctx, filter, accessibleServerIDs)
	if err != nil {
		return nil, err
	}

	s.logger.Debug().
		Int("count", len(servers)).
		Bool("filtered", accessibleServerIDs != nil).
		Msg("Servers listed for user")
	return servers, nil
}

// GetServer retrieves a single MCP server by ID
func (s *Service) GetServer(ctx context.Context, id string) (*domain.MCPServer, error) {
	server, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Fetch current health status
	health, err := s.repo.GetHealthStatus(ctx, id)
	if err != nil {
		s.logger.Warn().Err(err).Str("server_id", id).Msg("Failed to get health status")
	} else {
		server.CurrentStatus = health
	}

	return server, nil
}

// UpdateServer updates an existing MCP server
func (s *Service) UpdateServer(ctx context.Context, id string, req *domain.ServerUpdate) (*domain.MCPServer, error) {
	server, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("server_id", id).
		Str("name", server.Name).
		Msg("MCP server updated")

	return server, nil
}

// DeleteServer deletes an MCP server by ID
func (s *Service) DeleteServer(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	s.logger.Info().Str("server_id", id).Msg("MCP server deleted")
	return nil
}

// ToggleServer enables/disables an MCP server
func (s *Service) ToggleServer(ctx context.Context, id string, enabled bool) (*domain.MCPServer, error) {
	update := &domain.ServerUpdate{
		IsActive: &enabled,
	}

	server, err := s.repo.Update(ctx, id, update)
	if err != nil {
		return nil, err
	}

	action := "enabled"
	if !enabled {
		action = "disabled"
	}

	s.logger.Info().
		Str("server_id", id).
		Str("name", server.Name).
		Str("action", action).
		Msg("MCP server toggled")

	return server, nil
}

// CheckHealth performs a health check on an MCP server
func (s *Service) CheckHealth(ctx context.Context, serverID string) error {
	// Get server details
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return err
	}

	// Determine health check URL
	healthURL := server.HealthCheckURL
	if healthURL == "" {
		// Default to base URL + /health
		healthURL = server.URL + "/health"
	}

	// Perform health check with timeout
	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(server.TimeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	status, responseTimeMs, errorMsg := s.performHealthCheck(checkCtx, healthURL)
	if responseTimeMs == 0 {
		responseTimeMs = int(time.Since(start).Milliseconds())
	}

	// Save health check result
	health := &domain.ServerHealth{
		ServerID:       serverID,
		Status:         status,
		ResponseTimeMs: responseTimeMs,
		ErrorMessage:   errorMsg,
		CheckedAt:      time.Now(),
	}

	if err := s.repo.SaveHealthStatus(ctx, health); err != nil {
		s.logger.Error().Err(err).Str("server_id", serverID).Msg("Failed to save health status")
		return err
	}

	s.logger.Debug().
		Str("server_id", serverID).
		Str("status", string(status)).
		Int("response_time_ms", responseTimeMs).
		Msg("Health check completed")

	return nil
}

// performHealthCheck executes the actual HTTP health check
func (s *Service) performHealthCheck(ctx context.Context, url string) (domain.ServerStatus, int, string) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return domain.ServerStatusUnhealthy, 0, fmt.Sprintf("Failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	responseTimeMs := int(time.Since(start).Milliseconds())

	if err != nil {
		return domain.ServerStatusUnhealthy, responseTimeMs, fmt.Sprintf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Determine status based on HTTP status code
	var status domain.ServerStatus
	var errorMsg string

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		status = domain.ServerStatusHealthy
	case resp.StatusCode >= 500:
		status = domain.ServerStatusUnhealthy
		errorMsg = fmt.Sprintf("Server error: %d", resp.StatusCode)
	default:
		status = domain.ServerStatusDegraded
		errorMsg = fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
	}

	return status, responseTimeMs, errorMsg
}

// GetHealthStatus retrieves the latest health status for a server
func (s *Service) GetHealthStatus(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
	health, err := s.repo.GetHealthStatus(ctx, serverID)
	if err != nil {
		return nil, err
	}

	return health, nil
}

// TestConnectionRequest represents a connection test request
type TestConnectionRequest struct {
	URL             string `json:"url"`
	Transport       string `json:"transport"`
	ProtocolVersion string `json:"protocol_version"`
	TimeoutSeconds  int    `json:"timeout"`
}

// TestConnectionResult represents the result of a connection test
type TestConnectionResult struct {
	Success        bool   `json:"success"`
	ResponseTimeMs int    `json:"response_time_ms"`
	ServerInfo     any    `json:"server_info,omitempty"`
	Tools          []any  `json:"tools,omitempty"`
	ToolCount      int    `json:"tool_count,omitempty"`
	Resources      []any  `json:"resources,omitempty"`
	ResourceCount  int    `json:"resource_count,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// TestConnection tests connectivity to an MCP server without saving it
func (s *Service) TestConnection(ctx context.Context, req *TestConnectionRequest) (*TestConnectionResult, error) {
	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}

	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	start := time.Now()
	result := &TestConnectionResult{}

	// Determine transport type
	transport := req.Transport
	if transport == "" {
		transport = "http"
	}

	// Try to initialize and get tools/resources based on transport
	switch transport {
	case "http":
		result = s.testHTTPTransport(testCtx, req.URL)
	case "streamable_http":
		result = s.testStreamableHTTPTransport(testCtx, req.URL, req.ProtocolVersion)
	case "sse":
		result = s.testSSETransport(testCtx, req.URL)
	default:
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Unsupported transport type: %s", transport)
	}

	result.ResponseTimeMs = int(time.Since(start).Milliseconds())

	s.logger.Info().
		Str("url", req.URL).
		Str("transport", transport).
		Bool("success", result.Success).
		Int("response_time_ms", result.ResponseTimeMs).
		Msg("Connection test completed")

	return result, nil
}

// testHTTPTransport tests HTTP transport connectivity
func (s *Service) testHTTPTransport(ctx context.Context, baseURL string) *TestConnectionResult {
	result := &TestConnectionResult{}
	client := &http.Client{Timeout: 30 * time.Second}

	// Try initialize endpoint
	initURL := baseURL + "/initialize"
	req, err := http.NewRequestWithContext(ctx, "POST", initURL, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		result.ErrorMessage = fmt.Sprintf("Server returned error: %d", resp.StatusCode)
		return result
	}

	result.Success = true
	result.ServerInfo = map[string]string{"status": "connected"}

	// Try to get tools
	toolsURL := baseURL + "/tools/list"
	toolsReq, err := http.NewRequestWithContext(ctx, "POST", toolsURL, nil)
	if err != nil {
		// Skip tools listing if request creation fails
		return result
	}
	toolsReq.Header.Set("Content-Type", "application/json")
	toolsResp, err := client.Do(toolsReq)
	if err == nil && toolsResp.StatusCode < 400 {
		defer toolsResp.Body.Close()
		var toolsResult map[string]interface{}
		if json.NewDecoder(toolsResp.Body).Decode(&toolsResult) == nil {
			if tools, ok := toolsResult["tools"].([]interface{}); ok {
				result.ToolCount = len(tools)
				if len(tools) <= 10 {
					result.Tools = tools
				}
			}
		}
	}

	return result
}

// testStreamableHTTPTransport tests Streamable HTTP transport connectivity
// Note: Streamable HTTP servers may return SSE format responses
func (s *Service) testStreamableHTTPTransport(ctx context.Context, baseURL string, protocolVersion string) *TestConnectionResult {
	result := &TestConnectionResult{}
	client := &http.Client{Timeout: 30 * time.Second}

	if protocolVersion == "" {
		protocolVersion = "2025-11-25"
	}

	// Send initialize request with proper capabilities
	initPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]interface{}{
				"roots": map[string]interface{}{
					"listChanged": true,
				},
			},
			"clientInfo": map[string]string{
				"name":    "waffles",
				"version": "1.0.0",
			},
		},
		"id": 1,
	}

	body, err := json.Marshal(initPayload)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to marshal request: %v", err)
		return result
	}
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", protocolVersion)

	resp, err := client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		result.ErrorMessage = fmt.Sprintf("Server returned error: %d", resp.StatusCode)
		return result
	}

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to read response: %v", err)
		return result
	}

	// Check if response is SSE format or plain JSON
	var initResult map[string]interface{}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") || strings.HasPrefix(string(respBody), "event:") || strings.HasPrefix(string(respBody), "data:") {
		// Parse SSE response
		initResult = s.parseSSEResponse(string(respBody))
	} else {
		// Parse as plain JSON
		if err := json.Unmarshal(respBody, &initResult); err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to parse response: %v", err)
			return result
		}
	}

	result.Success = true
	if rpcResult, ok := initResult["result"].(map[string]interface{}); ok {
		result.ServerInfo = rpcResult["serverInfo"]
	}

	// Get session ID if provided (check both header name variants)
	sessionID := resp.Header.Get("MCP-Session-Id")
	if sessionID == "" {
		sessionID = resp.Header.Get("mcp-session-id")
	}

	// Send initialized notification (required by MCP protocol before making other requests)
	if sessionID != "" {
		initNotification := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "notifications/initialized",
		}
		notifyBody, _ := json.Marshal(initNotification)
		notifyReq, _ := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(notifyBody))
		notifyReq.Header.Set("Content-Type", "application/json")
		notifyReq.Header.Set("Accept", "application/json, text/event-stream")
		notifyReq.Header.Set("mcp-session-id", sessionID)
		notifyResp, err := client.Do(notifyReq)
		if err == nil {
			_ = notifyResp.Body.Close() // #nosec G104 -- best effort close
		}
	}

	// Try to list tools
	toolsPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"id":      2,
	}
	toolsBody, _ := json.Marshal(toolsPayload)
	toolsReq, _ := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(toolsBody))
	toolsReq.Header.Set("Content-Type", "application/json")
	toolsReq.Header.Set("Accept", "application/json, text/event-stream")
	toolsReq.Header.Set("MCP-Protocol-Version", protocolVersion)
	if sessionID != "" {
		toolsReq.Header.Set("mcp-session-id", sessionID)
	}

	toolsResp, err := client.Do(toolsReq)
	if err == nil && toolsResp.StatusCode < 400 {
		defer toolsResp.Body.Close()

		toolsRespBody, err := io.ReadAll(toolsResp.Body)
		if err == nil {
			var toolsResult map[string]interface{}
			toolsContentType := toolsResp.Header.Get("Content-Type")
			if strings.Contains(toolsContentType, "text/event-stream") || strings.HasPrefix(string(toolsRespBody), "event:") || strings.HasPrefix(string(toolsRespBody), "data:") {
				toolsResult = s.parseSSEResponse(string(toolsRespBody))
			} else {
				_ = json.Unmarshal(toolsRespBody, &toolsResult) // #nosec G104 -- parse errors handled via fallback
			}

			if rpcResult, ok := toolsResult["result"].(map[string]interface{}); ok {
				if tools, ok := rpcResult["tools"].([]interface{}); ok {
					result.ToolCount = len(tools)
					result.Tools = tools
				}
			}
		}
	}

	return result
}

// CallToolRequest represents a request to call a tool on an MCP server
type CallToolRequest struct {
	URL             string                 `json:"url"`
	Transport       string                 `json:"transport"`
	ProtocolVersion string                 `json:"protocol_version"`
	ToolName        string                 `json:"tool_name"`
	Arguments       map[string]interface{} `json:"arguments"`
	TimeoutSeconds  int                    `json:"timeout"`
}

// CallToolResult represents the result of calling a tool
type CallToolResult struct {
	Success      bool        `json:"success"`
	Content      interface{} `json:"content,omitempty"`
	IsError      bool        `json:"is_error,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// CallTool executes a tool on an MCP server
func (s *Service) CallTool(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}

	callCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	transport := req.Transport
	if transport == "" {
		transport = "streamable_http"
	}

	switch transport {
	case "streamable_http":
		return s.callToolStreamableHTTP(callCtx, req), nil
	case "http":
		return s.callToolHTTP(callCtx, req), nil
	case "sse":
		return s.callToolSSE(callCtx, req), nil
	default:
		return &CallToolResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Unsupported transport type: %s", transport),
		}, nil
	}
}

// callToolStreamableHTTP calls a tool using Streamable HTTP transport
// Note: Streamable HTTP servers may return SSE format responses
func (s *Service) callToolStreamableHTTP(ctx context.Context, req *CallToolRequest) *CallToolResult {
	result := &CallToolResult{}
	client := &http.Client{Timeout: 30 * time.Second}

	protocolVersion := req.ProtocolVersion
	if protocolVersion == "" {
		protocolVersion = "2025-11-25"
	}

	// First, initialize the connection with proper capabilities
	initPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]interface{}{
				"roots": map[string]interface{}{
					"listChanged": true,
				},
			},
			"clientInfo": map[string]string{
				"name":    "waffles-inspector",
				"version": "1.0.0",
			},
		},
		"id": 1,
	}

	body, _ := json.Marshal(initPayload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.URL, bytes.NewReader(body))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	httpReq.Header.Set("MCP-Protocol-Version", protocolVersion)

	resp, err := client.Do(httpReq)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		return result
	}
	_ = resp.Body.Close() // #nosec G104 -- best effort close

	// Get session ID (check both header name variants)
	sessionID := resp.Header.Get("MCP-Session-Id")
	if sessionID == "" {
		sessionID = resp.Header.Get("mcp-session-id")
	}

	// Send initialized notification (required by MCP protocol)
	if sessionID != "" {
		initNotification := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "notifications/initialized",
		}
		notifyBody, _ := json.Marshal(initNotification)
		notifyReq, _ := http.NewRequestWithContext(ctx, "POST", req.URL, bytes.NewReader(notifyBody))
		notifyReq.Header.Set("Content-Type", "application/json")
		notifyReq.Header.Set("Accept", "application/json, text/event-stream")
		notifyReq.Header.Set("mcp-session-id", sessionID)
		notifyResp, err := client.Do(notifyReq)
		if err == nil {
			_ = notifyResp.Body.Close() // #nosec G104 -- best effort close
		}
	}

	// Now call the tool
	callPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      req.ToolName,
			"arguments": req.Arguments,
		},
		"id": 2,
	}

	callBody, _ := json.Marshal(callPayload)
	callReq, _ := http.NewRequestWithContext(ctx, "POST", req.URL, bytes.NewReader(callBody))
	callReq.Header.Set("Content-Type", "application/json")
	callReq.Header.Set("Accept", "application/json, text/event-stream")
	callReq.Header.Set("MCP-Protocol-Version", protocolVersion)
	if sessionID != "" {
		callReq.Header.Set("mcp-session-id", sessionID)
	}

	callResp, err := client.Do(callReq)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Tool call failed: %v", err)
		return result
	}
	defer callResp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(callResp.Body)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to read response: %v", err)
		return result
	}

	// Check if response is SSE format or plain JSON
	var callResult map[string]interface{}
	contentType := callResp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") || strings.HasPrefix(string(respBody), "event:") || strings.HasPrefix(string(respBody), "data:") {
		// Parse SSE response
		callResult = s.parseSSEResponse(string(respBody))
	} else {
		// Parse as plain JSON
		if err := json.Unmarshal(respBody, &callResult); err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to parse response: %v", err)
			return result
		}
	}

	// Check for RPC error
	if rpcError, ok := callResult["error"].(map[string]interface{}); ok {
		result.IsError = true
		result.ErrorMessage = fmt.Sprintf("%v", rpcError["message"])
		return result
	}

	// Extract the result
	if rpcResult, ok := callResult["result"].(map[string]interface{}); ok {
		result.Success = true
		result.Content = rpcResult["content"]
		if isError, ok := rpcResult["isError"].(bool); ok {
			result.IsError = isError
		}
	} else {
		result.Success = true
		result.Content = callResult
	}

	return result
}

// callToolHTTP calls a tool using HTTP transport
func (s *Service) callToolHTTP(ctx context.Context, req *CallToolRequest) *CallToolResult {
	result := &CallToolResult{}
	client := &http.Client{Timeout: 30 * time.Second}

	callURL := req.URL + "/tools/call"
	payload := map[string]interface{}{
		"name":      req.ToolName,
		"arguments": req.Arguments,
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", callURL, bytes.NewReader(body))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Tool call failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	var callResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&callResult); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to parse response: %v", err)
		return result
	}

	result.Success = true
	result.Content = callResult

	return result
}

// callToolSSE calls a tool using SSE transport
// Note: Many SSE servers actually use Streamable HTTP protocol (POST with SSE response)
func (s *Service) callToolSSE(ctx context.Context, req *CallToolRequest) *CallToolResult {
	result := &CallToolResult{}
	client := &http.Client{Timeout: 30 * time.Second}

	callPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      req.ToolName,
			"arguments": req.Arguments,
		},
		"id": 1,
	}

	body, _ := json.Marshal(callPayload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.URL, bytes.NewReader(body))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := client.Do(httpReq)
	if err != nil {
		// Fallback: Try legacy /message endpoint
		body, _ = json.Marshal(callPayload)
		httpReq, _ = http.NewRequestWithContext(ctx, "POST", req.URL+"/message", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(httpReq)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Tool call failed: %v", err)
			return result
		}
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	var callResult map[string]interface{}
	if strings.HasPrefix(bodyStr, "event:") || strings.HasPrefix(bodyStr, "data:") {
		// SSE format - parse the data line
		callResult = s.parseSSEResponse(bodyStr)
	} else {
		// Plain JSON
		if err := json.Unmarshal(bodyBytes, &callResult); err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to parse response: %v", err)
			return result
		}
	}

	// Check for RPC error
	if rpcError, ok := callResult["error"].(map[string]interface{}); ok {
		result.IsError = true
		result.ErrorMessage = fmt.Sprintf("%v", rpcError["message"])
		return result
	}

	if rpcResult, ok := callResult["result"].(map[string]interface{}); ok {
		result.Success = true
		result.Content = rpcResult["content"]
		if isError, ok := rpcResult["isError"].(bool); ok {
			result.IsError = isError
		}
	} else {
		result.Success = true
		result.Content = callResult
	}

	return result
}

// testSSETransport tests SSE transport connectivity
// Note: Many SSE servers actually use Streamable HTTP protocol (POST with SSE response)
func (s *Service) testSSETransport(ctx context.Context, baseURL string) *TestConnectionResult {
	result := &TestConnectionResult{}
	client := &http.Client{Timeout: 30 * time.Second}

	// SSE servers that support Streamable HTTP require both Accept types
	initPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "waffles",
				"version": "1.0.0",
			},
		},
		"id": 1,
	}
	body, _ := json.Marshal(initPayload)

	// Try POST with both Accept types (Streamable HTTP over SSE)
	msgReq, _ := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	msgReq.Header.Set("Content-Type", "application/json")
	msgReq.Header.Set("Accept", "application/json, text/event-stream")

	msgResp, err := client.Do(msgReq)
	if err != nil {
		// Fallback: Try legacy /message endpoint
		body, _ = json.Marshal(initPayload)
		msgReq, _ = http.NewRequestWithContext(ctx, "POST", baseURL+"/message", bytes.NewReader(body))
		msgReq.Header.Set("Content-Type", "application/json")
		msgResp, err = client.Do(msgReq)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
			return result
		}
	}
	defer msgResp.Body.Close()

	if msgResp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(msgResp.Body)
		result.ErrorMessage = fmt.Sprintf("Server returned error %d: %s", msgResp.StatusCode, string(bodyBytes))
		return result
	}

	// Parse response - could be SSE format or plain JSON
	bodyBytes, _ := io.ReadAll(msgResp.Body)
	bodyStr := string(bodyBytes)

	var initResult map[string]interface{}
	if strings.HasPrefix(bodyStr, "event:") || strings.HasPrefix(bodyStr, "data:") {
		// SSE format - parse the data line
		initResult = s.parseSSEResponse(bodyStr)
	} else {
		// Plain JSON
		_ = json.Unmarshal(bodyBytes, &initResult) // #nosec G104 -- parse errors handled via fallback
	}

	if rpcResult, ok := initResult["result"].(map[string]interface{}); ok {
		result.Success = true
		result.ServerInfo = rpcResult["serverInfo"]
	} else if rpcError, ok := initResult["error"].(map[string]interface{}); ok {
		result.ErrorMessage = fmt.Sprintf("%v", rpcError["message"])
		return result
	} else {
		result.Success = true
		result.ServerInfo = map[string]string{"transport": "sse", "status": "connected"}
	}

	// Now try to list tools
	toolsPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"params":  map[string]interface{}{},
		"id":      2,
	}
	toolsBody, _ := json.Marshal(toolsPayload)
	toolsReq, _ := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(toolsBody))
	toolsReq.Header.Set("Content-Type", "application/json")
	toolsReq.Header.Set("Accept", "application/json, text/event-stream")

	toolsResp, err := client.Do(toolsReq)
	if err != nil {
		// Tools listing failed, but connection succeeded
		return result
	}
	defer toolsResp.Body.Close()

	if toolsResp.StatusCode < 400 {
		toolsBytes, _ := io.ReadAll(toolsResp.Body)
		toolsStr := string(toolsBytes)

		var toolsResult map[string]interface{}
		if strings.HasPrefix(toolsStr, "event:") || strings.HasPrefix(toolsStr, "data:") {
			toolsResult = s.parseSSEResponse(toolsStr)
		} else {
			_ = json.Unmarshal(toolsBytes, &toolsResult) //nolint:errcheck // ignore parse errors, fallback handled
		}

		if rpcResult, ok := toolsResult["result"].(map[string]interface{}); ok {
			if tools, ok := rpcResult["tools"].([]interface{}); ok {
				result.ToolCount = len(tools)
				result.Tools = tools
			}
		}
	}

	return result
}

// parseSSEResponse parses SSE format response and extracts JSON data
func (s *Service) parseSSEResponse(sseData string) map[string]interface{} {
	result := make(map[string]interface{})

	// SSE format is typically:
	// event: message
	// data: {"jsonrpc":"2.0",...}
	lines := strings.Split(sseData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			jsonData := strings.TrimPrefix(line, "data:")
			jsonData = strings.TrimSpace(jsonData)
			if err := json.Unmarshal([]byte(jsonData), &result); err == nil {
				return result
			}
		}
	}

	return result
}
