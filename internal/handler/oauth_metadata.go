package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/waffles/waffles/internal/config"
	"github.com/waffles/waffles/pkg/logger"
)

// OAuthMetadataHandler handles OAuth metadata endpoints for MCP authorization
// Implements RFC 9728 - OAuth Protected Resource Metadata
type OAuthMetadataHandler struct {
	oauthConfig   config.OAuthConfig
	mcpAuthConfig config.MCPAuthConfig
	logger        logger.Logger
}

// NewOAuthMetadataHandler creates a new OAuth metadata handler
func NewOAuthMetadataHandler(oauthCfg config.OAuthConfig, mcpAuthCfg config.MCPAuthConfig, log logger.Logger) *OAuthMetadataHandler {
	return &OAuthMetadataHandler{
		oauthConfig:   oauthCfg,
		mcpAuthConfig: mcpAuthCfg,
		logger:        log,
	}
}

// ProtectedResourceMetadata represents the OAuth Protected Resource Metadata (RFC 9728)
type ProtectedResourceMetadata struct {
	// Resource identifier (the gateway URL)
	Resource string `json:"resource"`

	// Authorization servers that can issue tokens for this resource
	AuthorizationServers []string `json:"authorization_servers"`

	// Supported scopes
	ScopesSupported []string `json:"scopes_supported,omitempty"`

	// Bearer token methods supported
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`

	// Resource documentation URL
	ResourceDocumentation string `json:"resource_documentation,omitempty"`
}

// GetProtectedResourceMetadata handles GET /.well-known/oauth-protected-resource
// This endpoint tells MCP clients where to authenticate
func (h *OAuthMetadataHandler) GetProtectedResourceMetadata(c *gin.Context) {
	// Check if OAuth is enabled for MCP clients
	if !h.mcpAuthConfig.OAuthEnabled {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OAuth is not enabled for MCP clients",
		})
		return
	}

	if !h.oauthConfig.Enabled || h.oauthConfig.Issuer == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OAuth is not configured",
		})
		return
	}

	// Build the resource URL (the gateway itself)
	resource := h.oauthConfig.BaseURL
	if resource == "" {
		// Fall back to request host
		scheme := "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
		resource = scheme + "://" + c.Request.Host
	}

	metadata := ProtectedResourceMetadata{
		Resource:               resource,
		AuthorizationServers:   []string{h.oauthConfig.Issuer},
		ScopesSupported:        []string{"openid", "email", "profile"},
		BearerMethodsSupported: []string{"header"},
	}

	h.logger.Info().
		Str("resource", resource).
		Str("authorization_server", h.oauthConfig.Issuer).
		Msg("Serving OAuth protected resource metadata")

	c.JSON(http.StatusOK, metadata)
}

// GetAuthorizationServerMetadata handles GET /.well-known/oauth-authorization-server
// This proxies the authorization server metadata from the OIDC issuer
// Per MCP spec, clients may request this at the resource server URL
func (h *OAuthMetadataHandler) GetAuthorizationServerMetadata(c *gin.Context) {
	// Check if OAuth is enabled for MCP clients
	if !h.mcpAuthConfig.OAuthEnabled {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OAuth is not enabled for MCP clients",
		})
		return
	}

	if !h.oauthConfig.Enabled || h.oauthConfig.Issuer == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OAuth is not configured",
		})
		return
	}

	// Fetch the OIDC discovery document from the authorization server
	discoveryURL := strings.TrimSuffix(h.oauthConfig.Issuer, "/") + "/.well-known/openid-configuration"

	resp, err := http.Get(discoveryURL) // #nosec G107 -- URL is constructed from admin-configured issuer
	if err != nil {
		h.logger.Error().Err(err).Str("url", discoveryURL).Msg("Failed to fetch OIDC discovery")
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Failed to fetch authorization server metadata",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logger.Error().Int("status", resp.StatusCode).Str("url", discoveryURL).Msg("OIDC discovery returned non-200")
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Authorization server returned error",
		})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to read OIDC discovery response")
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Failed to read authorization server metadata",
		})
		return
	}

	// Validate it's valid JSON
	var metadata map[string]interface{}
	if err := json.Unmarshal(body, &metadata); err != nil {
		h.logger.Error().Err(err).Msg("Invalid JSON from OIDC discovery")
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Invalid authorization server metadata",
		})
		return
	}

	h.logger.Info().
		Str("issuer", h.oauthConfig.Issuer).
		Msg("Serving proxied authorization server metadata")

	c.Header("Content-Type", "application/json")
	_, _ = c.Writer.Write(body) // #nosec G104 -- HTTP response write error is non-critical
}
