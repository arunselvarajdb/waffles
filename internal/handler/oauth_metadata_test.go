package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/config"
	"github.com/waffles/waffles/pkg/logger"
)

func TestNewOAuthMetadataHandler(t *testing.T) {
	log := logger.NewNopLogger()
	oauthCfg := config.OAuthConfig{
		Enabled: true,
		Issuer:  "https://auth.example.com",
		BaseURL: "https://gateway.example.com",
	}
	mcpAuthCfg := config.MCPAuthConfig{
		OAuthEnabled: true,
	}

	handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

	require.NotNil(t, handler)
	assert.Equal(t, oauthCfg.Enabled, handler.oauthConfig.Enabled)
	assert.Equal(t, oauthCfg.Issuer, handler.oauthConfig.Issuer)
	assert.Equal(t, mcpAuthCfg.OAuthEnabled, handler.mcpAuthConfig.OAuthEnabled)
}

func TestOAuthMetadataHandler_GetProtectedResourceMetadata(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns 404 when MCP OAuth is disabled", func(t *testing.T) {
		oauthCfg := config.OAuthConfig{
			Enabled: true,
			Issuer:  "https://auth.example.com",
		}
		mcpAuthCfg := config.MCPAuthConfig{
			OAuthEnabled: false,
		}
		handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)

		handler.GetProtectedResourceMetadata(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "OAuth is not enabled")
	})

	t.Run("returns 404 when OAuth is not configured", func(t *testing.T) {
		oauthCfg := config.OAuthConfig{
			Enabled: false,
			Issuer:  "",
		}
		mcpAuthCfg := config.MCPAuthConfig{
			OAuthEnabled: true,
		}
		handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)

		handler.GetProtectedResourceMetadata(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "OAuth is not configured")
	})

	t.Run("returns 404 when OAuth enabled but issuer is empty", func(t *testing.T) {
		oauthCfg := config.OAuthConfig{
			Enabled: true,
			Issuer:  "",
		}
		mcpAuthCfg := config.MCPAuthConfig{
			OAuthEnabled: true,
		}
		handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)

		handler.GetProtectedResourceMetadata(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns metadata with configured base URL", func(t *testing.T) {
		oauthCfg := config.OAuthConfig{
			Enabled: true,
			Issuer:  "https://auth.example.com",
			BaseURL: "https://gateway.example.com",
		}
		mcpAuthCfg := config.MCPAuthConfig{
			OAuthEnabled: true,
		}
		handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)

		handler.GetProtectedResourceMetadata(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response ProtectedResourceMetadata
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "https://gateway.example.com", response.Resource)
		assert.Contains(t, response.AuthorizationServers, "https://auth.example.com")
		assert.Contains(t, response.ScopesSupported, "openid")
		assert.Contains(t, response.BearerMethodsSupported, "header")
	})

	t.Run("uses request host when base URL is empty", func(t *testing.T) {
		oauthCfg := config.OAuthConfig{
			Enabled: true,
			Issuer:  "https://auth.example.com",
			BaseURL: "",
		}
		mcpAuthCfg := config.MCPAuthConfig{
			OAuthEnabled: true,
		}
		handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)
		c.Request.Host = "localhost:8080"

		handler.GetProtectedResourceMetadata(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response ProtectedResourceMetadata
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080", response.Resource)
	})
}

func TestOAuthMetadataHandler_GetAuthorizationServerMetadata(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns 404 when MCP OAuth is disabled", func(t *testing.T) {
		oauthCfg := config.OAuthConfig{
			Enabled: true,
			Issuer:  "https://auth.example.com",
		}
		mcpAuthCfg := config.MCPAuthConfig{
			OAuthEnabled: false,
		}
		handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)

		handler.GetAuthorizationServerMetadata(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 404 when OAuth is not configured", func(t *testing.T) {
		oauthCfg := config.OAuthConfig{
			Enabled: false,
			Issuer:  "",
		}
		mcpAuthCfg := config.MCPAuthConfig{
			OAuthEnabled: true,
		}
		handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)

		handler.GetAuthorizationServerMetadata(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 502 when OIDC discovery fails", func(t *testing.T) {
		oauthCfg := config.OAuthConfig{
			Enabled: true,
			Issuer:  "http://invalid.nonexistent.domain.test",
		}
		mcpAuthCfg := config.MCPAuthConfig{
			OAuthEnabled: true,
		}
		handler := NewOAuthMetadataHandler(oauthCfg, mcpAuthCfg, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)

		handler.GetAuthorizationServerMetadata(c)

		assert.Equal(t, http.StatusBadGateway, w.Code)
	})
}

func TestProtectedResourceMetadata_JSON(t *testing.T) {
	metadata := ProtectedResourceMetadata{
		Resource:               "https://gateway.example.com",
		AuthorizationServers:   []string{"https://auth.example.com"},
		ScopesSupported:        []string{"openid", "email", "profile"},
		BearerMethodsSupported: []string{"header"},
		ResourceDocumentation:  "https://docs.example.com",
	}

	data, err := json.Marshal(metadata)
	require.NoError(t, err)

	var parsed ProtectedResourceMetadata
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, metadata.Resource, parsed.Resource)
	assert.Equal(t, metadata.AuthorizationServers, parsed.AuthorizationServers)
	assert.Equal(t, metadata.ScopesSupported, parsed.ScopesSupported)
	assert.Equal(t, metadata.BearerMethodsSupported, parsed.BearerMethodsSupported)
	assert.Equal(t, metadata.ResourceDocumentation, parsed.ResourceDocumentation)
}
