package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/internal/metrics"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// proxyStartTimeKey is the context key for tracking proxy start time
const proxyStartTimeKey contextKey = "proxy_start_time"

// ServerRepository defines the interface for server data access.
type ServerRepository interface {
	Get(ctx context.Context, id string) (*domain.MCPServer, error)
}

// SSEClientInterface defines the interface for SSE client operations.
type SSEClientInterface interface {
	Call(ctx context.Context, server *domain.MCPServer, method string, params interface{}) (json.RawMessage, error)
}

// StreamableHTTPClientInterface defines the interface for Streamable HTTP client operations.
type StreamableHTTPClientInterface interface {
	Call(ctx context.Context, server *domain.MCPServer, method string, params interface{}) (json.RawMessage, error)
	Initialize(ctx context.Context, server *domain.MCPServer) (*MCPSession, error)
	TerminateSession(ctx context.Context, server *domain.MCPServer) error
}

// Service handles MCP gateway operations using ReverseProxy
type Service struct {
	repo                 ServerRepository
	logger               logger.Logger
	metrics              *metrics.Registry
	sseClient            SSEClientInterface            // Legacy SSE client (deprecated)
	streamableHTTPClient StreamableHTTPClientInterface // Streamable HTTP client (MCP 2025-11-25)
}

// NewService creates a new gateway service
func NewService(repo ServerRepository, log logger.Logger, metricsReg *metrics.Registry) *Service {
	return &Service{
		repo:                 repo,
		logger:               log,
		metrics:              metricsReg,
		sseClient:            NewSSEClient(log, 30*time.Second),
		streamableHTTPClient: NewStreamableHTTPClient(log, 30*time.Second),
	}
}

// NewServiceWithClients creates a new gateway service with custom clients (useful for testing).
func NewServiceWithClients(repo ServerRepository, log logger.Logger, metricsReg *metrics.Registry, sseClient SSEClientInterface, streamableHTTPClient StreamableHTTPClientInterface) *Service {
	return &Service{
		repo:                 repo,
		logger:               log,
		metrics:              metricsReg,
		sseClient:            sseClient,
		streamableHTTPClient: streamableHTTPClient,
	}
}

// ProxyToServer creates a reverse proxy for a registered MCP server
func (s *Service) ProxyToServer(
	ctx context.Context,
	serverID string,
) (*httputil.ReverseProxy, *domain.MCPServer, error) {
	// Look up server from registry
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return nil, nil, fmt.Errorf("server not found: %w", err)
	}

	// Check if server is active
	if !server.IsActive {
		return nil, nil, fmt.Errorf("server %s is inactive", serverID)
	}

	// Parse server URL
	target, err := url.Parse(server.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid server URL %s: %w", server.URL, err)
	}

	// Create reverse proxy with custom Director
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Save original path for logging
			originalPath := req.URL.Path

			// Track start time for latency measurement
			startTime := time.Now()
			req = req.WithContext(context.WithValue(req.Context(), proxyStartTimeKey, startTime))

			// Increment in-flight gauge
			if s.metrics != nil {
				s.metrics.GatewayRequestsInFlight.WithLabelValues(serverID, server.Name).Inc()
			}

			// Rewrite the path: strip /api/v1/gateway/:server_id prefix
			// Original: /api/v1/gateway/SERVER_ID/tools/list
			// Target:   /tools/list (or server's base path like /mcp)
			rewrittenPath := rewriteProxyPath(req.URL.Path, serverID)

			// Combine target path with rewritten path
			// e.g., target.Path="/mcp", rewrittenPath="" -> "/mcp"
			// e.g., target.Path="/mcp", rewrittenPath="/sse" -> "/mcp/sse"
			finalPath := target.Path
			if rewrittenPath != "" && rewrittenPath != "/" {
				finalPath = strings.TrimSuffix(target.Path, "/") + rewrittenPath
			}
			if finalPath == "" {
				finalPath = "/"
			}

			// Set the target URL
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = finalPath
			req.Host = target.Host

			// Add MCP-specific auth if configured
			s.injectAuth(req, server)

			// Log the proxied request
			s.logger.Info().
				Str("server_id", serverID).
				Str("server_name", server.Name).
				Str("method", req.Method).
				Str("original_path", originalPath).
				Str("final_path", finalPath).
				Str("target_url", target.String()).
				Msg("Proxying request to MCP server")
		},
		Transport: &http.Transport{
			MaxIdleConns:        server.MaxConnections,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     time.Duration(server.TimeoutSeconds) * time.Second,
			DisableKeepAlives:   false,
		},
	}

	// Hook ModifyResponse for logging responses and metrics
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Decrement in-flight gauge
		if s.metrics != nil {
			s.metrics.GatewayRequestsInFlight.WithLabelValues(serverID, server.Name).Dec()

			// Record request duration
			if startTime, ok := resp.Request.Context().Value(proxyStartTimeKey).(time.Time); ok {
				duration := time.Since(startTime).Seconds()
				s.metrics.GatewayRequestDuration.WithLabelValues(serverID, server.Name).Observe(duration)
			}

			// Increment total counter
			status := fmt.Sprintf("%d", resp.StatusCode)
			s.metrics.GatewayRequestsTotal.WithLabelValues(serverID, server.Name, status).Inc()
		}

		s.logger.Info().
			Str("server_id", serverID).
			Int("status", resp.StatusCode).
			Str("content_type", resp.Header.Get("Content-Type")).
			Msg("MCP server responded")
		return nil
	}

	// Handle proxy errors
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		// Decrement in-flight gauge and record error metrics
		if s.metrics != nil {
			s.metrics.GatewayRequestsInFlight.WithLabelValues(serverID, server.Name).Dec()

			// Record request duration even on error
			if startTime, ok := r.Context().Value(proxyStartTimeKey).(time.Time); ok {
				duration := time.Since(startTime).Seconds()
				s.metrics.GatewayRequestDuration.WithLabelValues(serverID, server.Name).Observe(duration)
			}

			// Increment error counter (502 Bad Gateway)
			s.metrics.GatewayRequestsTotal.WithLabelValues(serverID, server.Name, "502").Inc()
		}

		s.logger.Error().
			Err(err).
			Str("server_id", serverID).
			Str("server_name", server.Name).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("Proxy error")

		w.WriteHeader(http.StatusBadGateway)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "failed to proxy request to MCP server: %s"}`, err.Error())
	}

	return proxy, server, nil
}

// injectAuth adds authentication to requests based on server config
func (s *Service) injectAuth(req *http.Request, server *domain.MCPServer) {
	// AuthConfig is json.RawMessage ([]byte), needs to be unmarshaled
	if len(server.AuthConfig) == 0 {
		s.logger.Debug().
			Str("server_id", server.ID).
			Msg("No authentication configuration found")
		return
	}

	var authConfig map[string]interface{}
	if err := json.Unmarshal(server.AuthConfig, &authConfig); err != nil {
		s.logger.Error().
			Err(err).
			Str("server_id", server.ID).
			Msg("Failed to parse authentication config")
		return
	}

	switch server.AuthType {
	case domain.ServerAuthBearer:
		// Extract token from config (if encrypted, decrypt here)
		if token, ok := authConfig["token"].(string); ok && token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			s.logger.Debug().
				Str("server_id", server.ID).
				Msg("Added Bearer token authentication")
		}

	case domain.ServerAuthBasic:
		// Extract username and password from config
		username, _ := authConfig["username"].(string)
		password, _ := authConfig["password"].(string)
		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
			s.logger.Debug().
				Str("server_id", server.ID).
				Str("username", username).
				Msg("Added Basic authentication")
		}

	case domain.ServerAuthNone:
		// No authentication needed
		s.logger.Debug().
			Str("server_id", server.ID).
			Msg("No authentication configured")

	default:
		s.logger.Warn().
			Str("server_id", server.ID).
			Str("auth_type", string(server.AuthType)).
			Msg("Unknown authentication type")
	}
}

// Initialize sends an initialize request to an MCP server (direct call, not proxied)
func (s *Service) Initialize(ctx context.Context, serverID string) (*domain.MCPServer, error) {
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Check if server is active
	if !server.IsActive {
		return nil, fmt.Errorf("server %s is inactive", serverID)
	}

	s.logger.Info().
		Str("server_id", serverID).
		Str("server_name", server.Name).
		Str("url", server.URL).
		Msg("Initialize request for MCP server")

	// Return server info (actual initialize handshake would happen via proxy)
	return server, nil
}

// GetServerInfo retrieves server information
func (s *Service) GetServerInfo(ctx context.Context, serverID string) (*domain.MCPServer, error) {
	return s.repo.Get(ctx, serverID)
}

// CallSSE sends a JSON-RPC request to an SSE-based MCP server
func (s *Service) CallSSE(ctx context.Context, serverID string, method string, params interface{}) (json.RawMessage, error) {
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}

	if !server.IsActive {
		return nil, fmt.Errorf("server %s is inactive", serverID)
	}

	s.logger.Info().
		Str("server_id", serverID).
		Str("server_name", server.Name).
		Str("method", method).
		Msg("Calling SSE-based MCP server")

	return s.sseClient.Call(ctx, server, method, params)
}

// IsSSEServer checks if a server uses SSE transport
func (s *Service) IsSSEServer(ctx context.Context, serverID string) (bool, *domain.MCPServer, error) {
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return false, nil, err
	}
	return IsSSEServer(server), server, nil
}

// IsStreamableHTTPServer checks if a server uses Streamable HTTP transport (MCP 2025-11-25)
func (s *Service) IsStreamableHTTPServer(ctx context.Context, serverID string) (bool, *domain.MCPServer, error) {
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return false, nil, err
	}
	return IsStreamableHTTPServer(server), server, nil
}

// CallStreamableHTTP sends a JSON-RPC request to a Streamable HTTP MCP server
func (s *Service) CallStreamableHTTP(ctx context.Context, serverID string, method string, params interface{}) (json.RawMessage, error) {
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}

	if !server.IsActive {
		return nil, fmt.Errorf("server %s is inactive", serverID)
	}

	s.logger.Info().
		Str("server_id", serverID).
		Str("server_name", server.Name).
		Str("method", method).
		Msg("Calling Streamable HTTP MCP server")

	return s.streamableHTTPClient.Call(ctx, server, method, params)
}

// InitializeStreamableHTTP initializes an MCP session with a Streamable HTTP server
func (s *Service) InitializeStreamableHTTP(ctx context.Context, serverID string) (*MCPSession, error) {
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}

	if !server.IsActive {
		return nil, fmt.Errorf("server %s is inactive", serverID)
	}

	s.logger.Info().
		Str("server_id", serverID).
		Str("server_name", server.Name).
		Str("url", server.URL).
		Msg("Initializing Streamable HTTP MCP session")

	return s.streamableHTTPClient.Initialize(ctx, server)
}

// TerminateStreamableHTTP terminates an MCP session with a Streamable HTTP server
func (s *Service) TerminateStreamableHTTP(ctx context.Context, serverID string) error {
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return err
	}

	return s.streamableHTTPClient.TerminateSession(ctx, server)
}

// GetTransportType determines the transport type for a server
func (s *Service) GetTransportType(ctx context.Context, serverID string) (domain.TransportType, *domain.MCPServer, error) {
	server, err := s.repo.Get(ctx, serverID)
	if err != nil {
		return "", nil, err
	}

	// Check explicit transport setting first
	if server.Transport != "" {
		return server.Transport, server, nil
	}

	// Auto-detect based on URL patterns
	if IsStreamableHTTPServer(server) {
		return domain.TransportStreamableHTTP, server, nil
	}
	if IsSSEServer(server) {
		return domain.TransportSSE, server, nil
	}

	// Default to HTTP
	return domain.TransportHTTP, server, nil
}

// rewriteProxyPath strips the gateway prefix from the path
// Example: /api/v1/gateway/SERVER_ID/tools/list -> /tools/list
func rewriteProxyPath(originalPath, serverID string) string {
	// Remove /api/v1/gateway/:server_id prefix
	prefix := fmt.Sprintf("/api/v1/gateway/%s", serverID)
	if strings.HasPrefix(originalPath, prefix) {
		return strings.TrimPrefix(originalPath, prefix)
	}
	return originalPath
}
