package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/waffles/waffles/internal/config"
	"github.com/waffles/waffles/pkg/logger"
)

func createMockOIDCServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// Discovery endpoint
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		// We need to return the server URL, but we don't have it yet
		// So we use a placeholder that will be replaced
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OIDCDiscovery{
			Issuer:                "http://test-issuer",
			AuthorizationEndpoint: "http://test-issuer/auth",
			TokenEndpoint:         "http://test-issuer/token",
			UserinfoEndpoint:      "http://test-issuer/userinfo",
			JwksURI:               "http://test-issuer/jwks",
		})
	})

	// Userinfo endpoint
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":                "user-123",
			"email":              "test@example.com",
			"email_verified":     true,
			"name":               "Test User",
			"preferred_username": "testuser",
		})
	})

	// Token endpoint (for ExchangeCode)
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "test-access-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "test-refresh-token",
		})
	})

	return httptest.NewServer(mux)
}

func TestNewService_Disabled(t *testing.T) {
	log := logger.NewNopLogger()
	cfg := config.OAuthConfig{
		Enabled: false,
	}

	svc := NewService(cfg, log)

	require.NotNil(t, svc)
	assert.False(t, svc.IsEnabled())
}

func TestNewService_MissingIssuer(t *testing.T) {
	log := logger.NewNopLogger()
	cfg := config.OAuthConfig{
		Enabled: true,
		Issuer:  "",
	}

	svc := NewService(cfg, log)

	require.NotNil(t, svc)
	assert.False(t, svc.IsEnabled())
}

func TestNewService_DiscoveryFails(t *testing.T) {
	log := logger.NewNopLogger()
	cfg := config.OAuthConfig{
		Enabled: true,
		Issuer:  "http://localhost:1/nonexistent",
	}

	svc := NewService(cfg, log)

	require.NotNil(t, svc)
	assert.False(t, svc.IsEnabled())
}

func TestNewService_Success(t *testing.T) {
	ts := createMockOIDCServer(t)
	defer ts.Close()

	log := logger.NewNopLogger()
	cfg := config.OAuthConfig{
		Enabled:      true,
		Issuer:       ts.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		BaseURL:      "http://localhost:8080",
	}

	svc := NewService(cfg, log)

	require.NotNil(t, svc)
	assert.True(t, svc.IsEnabled())
}

func TestNewService_DefaultScopes(t *testing.T) {
	ts := createMockOIDCServer(t)
	defer ts.Close()

	log := logger.NewNopLogger()
	cfg := config.OAuthConfig{
		Enabled:      true,
		Issuer:       ts.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		BaseURL:      "http://localhost:8080",
		Scopes:       []string{}, // Empty scopes should default
	}

	svc := NewService(cfg, log)

	require.NotNil(t, svc)
	assert.True(t, svc.IsEnabled())
}

func TestNewService_CustomScopes(t *testing.T) {
	ts := createMockOIDCServer(t)
	defer ts.Close()

	log := logger.NewNopLogger()
	cfg := config.OAuthConfig{
		Enabled:      true,
		Issuer:       ts.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		BaseURL:      "http://localhost:8080",
		Scopes:       []string{"openid", "email"},
	}

	svc := NewService(cfg, log)

	require.NotNil(t, svc)
	assert.True(t, svc.IsEnabled())
}

func TestIsEnabled(t *testing.T) {
	t.Run("disabled config", func(t *testing.T) {
		svc := &Service{
			config: config.OAuthConfig{Enabled: false},
		}
		assert.False(t, svc.IsEnabled())
	})

	t.Run("enabled but no oauth2 config", func(t *testing.T) {
		svc := &Service{
			config: config.OAuthConfig{Enabled: true},
		}
		assert.False(t, svc.IsEnabled())
	})

	t.Run("enabled with oauth2 config", func(t *testing.T) {
		svc := &Service{
			config:       config.OAuthConfig{Enabled: true},
			oauth2Config: &oauth2.Config{},
		}
		assert.True(t, svc.IsEnabled())
	})
}

func TestGenerateState(t *testing.T) {
	svc := &Service{}

	state1, err := svc.GenerateState()
	require.NoError(t, err)
	assert.NotEmpty(t, state1)

	state2, err := svc.GenerateState()
	require.NoError(t, err)
	assert.NotEmpty(t, state2)

	// States should be different
	assert.NotEqual(t, state1, state2)
}

func TestGetAuthURL(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		svc := &Service{}

		url, err := svc.GetAuthURL("test-state")

		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not configured")
	})

	t.Run("configured", func(t *testing.T) {
		svc := &Service{
			oauth2Config: &oauth2.Config{
				ClientID: "test-client",
				Endpoint: oauth2.Endpoint{
					AuthURL: "http://localhost/auth",
				},
			},
		}

		url, err := svc.GetAuthURL("test-state")

		require.NoError(t, err)
		assert.Contains(t, url, "http://localhost/auth")
		assert.Contains(t, url, "test-state")
		assert.Contains(t, url, "test-client")
	})
}

func TestExchangeCode_NotConfigured(t *testing.T) {
	svc := &Service{}

	userInfo, err := svc.ExchangeCode(context.Background(), "test-code")

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "not configured")
}

func TestValidateEmailDomain(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		allowedDomains []string
		expectError    bool
	}{
		{
			name:           "no domains allowed - allow all",
			email:          "test@example.com",
			allowedDomains: nil,
			expectError:    false,
		},
		{
			name:           "empty domains allowed - allow all",
			email:          "test@example.com",
			allowedDomains: []string{},
			expectError:    false,
		},
		{
			name:           "domain in allowed list",
			email:          "test@example.com",
			allowedDomains: []string{"example.com"},
			expectError:    false,
		},
		{
			name:           "domain in allowed list case insensitive",
			email:          "test@EXAMPLE.COM",
			allowedDomains: []string{"example.com"},
			expectError:    false,
		},
		{
			name:           "domain not in allowed list",
			email:          "test@other.com",
			allowedDomains: []string{"example.com"},
			expectError:    true,
		},
		{
			name:           "invalid email format",
			email:          "invalid-email",
			allowedDomains: []string{"example.com"},
			expectError:    true,
		},
		{
			name:           "multiple allowed domains",
			email:          "test@second.com",
			allowedDomains: []string{"first.com", "second.com", "third.com"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{
				config: config.OAuthConfig{
					AllowedDomains: tt.allowedDomains,
				},
				logger: logger.NewNopLogger(),
			}

			err := svc.validateEmailDomain(tt.email)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetDefaultRole(t *testing.T) {
	t.Run("default when not configured", func(t *testing.T) {
		svc := &Service{
			config: config.OAuthConfig{DefaultRole: ""},
		}

		assert.Equal(t, "viewer", svc.GetDefaultRole())
	})

	t.Run("configured role", func(t *testing.T) {
		svc := &Service{
			config: config.OAuthConfig{DefaultRole: "operator"},
		}

		assert.Equal(t, "operator", svc.GetDefaultRole())
	})
}

func TestAutoCreateUsers(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		svc := &Service{
			config: config.OAuthConfig{AutoCreateUsers: false},
		}

		assert.False(t, svc.AutoCreateUsers())
	})

	t.Run("enabled", func(t *testing.T) {
		svc := &Service{
			config: config.OAuthConfig{AutoCreateUsers: true},
		}

		assert.True(t, svc.AutoCreateUsers())
	})
}

func TestValidateBearerToken_NotEnabled(t *testing.T) {
	svc := &Service{
		config: config.OAuthConfig{Enabled: false},
	}

	userInfo, err := svc.ValidateBearerToken(context.Background(), "test-token")

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestValidateBearerToken_NoDiscovery(t *testing.T) {
	svc := &Service{
		config:       config.OAuthConfig{Enabled: true},
		oauth2Config: &oauth2.Config{},
		discovery:    nil,
	}

	userInfo, err := svc.ValidateBearerToken(context.Background(), "test-token")

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "userinfo endpoint not available")
}

func TestValidateBearerToken_EmptyUserinfoEndpoint(t *testing.T) {
	svc := &Service{
		config:       config.OAuthConfig{Enabled: true},
		oauth2Config: &oauth2.Config{},
		discovery:    &OIDCDiscovery{UserinfoEndpoint: ""},
	}

	userInfo, err := svc.ValidateBearerToken(context.Background(), "test-token")

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "userinfo endpoint not available")
}

func TestValidateBearerToken_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token", auth)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":                "user-123",
			"email":              "test@example.com",
			"email_verified":     true,
			"name":               "Test User",
			"preferred_username": "testuser",
		})
	}))
	defer ts.Close()

	svc := &Service{
		config:       config.OAuthConfig{Enabled: true},
		oauth2Config: &oauth2.Config{},
		discovery:    &OIDCDiscovery{UserinfoEndpoint: ts.URL},
		logger:       logger.NewNopLogger(),
	}

	userInfo, err := svc.ValidateBearerToken(context.Background(), "test-token")

	require.NoError(t, err)
	require.NotNil(t, userInfo)
	assert.Equal(t, "user-123", userInfo.ID)
	assert.Equal(t, "test@example.com", userInfo.Email)
	assert.Equal(t, "Test User", userInfo.Name)
	assert.Equal(t, ProviderOIDC, userInfo.Provider)
}

func TestValidateBearerToken_DomainNotAllowed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":   "user-123",
			"email": "test@notallowed.com",
			"name":  "Test User",
		})
	}))
	defer ts.Close()

	svc := &Service{
		config: config.OAuthConfig{
			Enabled:        true,
			AllowedDomains: []string{"example.com"},
		},
		oauth2Config: &oauth2.Config{},
		discovery:    &OIDCDiscovery{UserinfoEndpoint: ts.URL},
		logger:       logger.NewNopLogger(),
	}

	userInfo, err := svc.ValidateBearerToken(context.Background(), "test-token")

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestFetchUserInfo_Fallback(t *testing.T) {
	// Test that preferred_username is used when name is empty
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":                "user-456",
			"email":              "test2@example.com",
			"name":               "",
			"preferred_username": "preferred_user",
		})
	}))
	defer ts.Close()

	svc := &Service{
		config:       config.OAuthConfig{Enabled: true},
		oauth2Config: &oauth2.Config{},
		discovery:    &OIDCDiscovery{UserinfoEndpoint: ts.URL},
		logger:       logger.NewNopLogger(),
	}

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	userInfo, err := svc.fetchUserInfo(context.Background(), token)

	require.NoError(t, err)
	require.NotNil(t, userInfo)
	assert.Equal(t, "preferred_user", userInfo.Name)
}

func TestFetchUserInfo_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	svc := &Service{
		discovery: &OIDCDiscovery{UserinfoEndpoint: ts.URL},
		logger:    logger.NewNopLogger(),
	}

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	userInfo, err := svc.fetchUserInfo(context.Background(), token)

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "401")
}

func TestFetchUserInfo_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	svc := &Service{
		discovery: &OIDCDiscovery{UserinfoEndpoint: ts.URL},
		logger:    logger.NewNopLogger(),
	}

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	userInfo, err := svc.fetchUserInfo(context.Background(), token)

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "parse")
}

func TestFetchUserInfo_NoDiscovery(t *testing.T) {
	svc := &Service{
		discovery: nil,
		logger:    logger.NewNopLogger(),
	}

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	userInfo, err := svc.fetchUserInfo(context.Background(), token)

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "not available")
}

func TestFetchOIDCDiscovery_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	svc := &Service{}

	discovery, err := svc.fetchOIDCDiscovery(ts.URL)

	assert.Error(t, err)
	assert.Nil(t, discovery)
	assert.Contains(t, err.Error(), "404")
}

func TestFetchOIDCDiscovery_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	svc := &Service{}

	discovery, err := svc.fetchOIDCDiscovery(ts.URL)

	assert.Error(t, err)
	assert.Nil(t, discovery)
	assert.Contains(t, err.Error(), "parse")
}

func TestFetchOIDCDiscovery_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/.well-known/openid-configuration", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OIDCDiscovery{
			Issuer:                "http://test-issuer",
			AuthorizationEndpoint: "http://test-issuer/auth",
			TokenEndpoint:         "http://test-issuer/token",
			UserinfoEndpoint:      "http://test-issuer/userinfo",
			JwksURI:               "http://test-issuer/jwks",
		})
	}))
	defer ts.Close()

	svc := &Service{}

	discovery, err := svc.fetchOIDCDiscovery(ts.URL)

	require.NoError(t, err)
	require.NotNil(t, discovery)
	assert.Equal(t, "http://test-issuer", discovery.Issuer)
	assert.Equal(t, "http://test-issuer/auth", discovery.AuthorizationEndpoint)
	assert.Equal(t, "http://test-issuer/token", discovery.TokenEndpoint)
	assert.Equal(t, "http://test-issuer/userinfo", discovery.UserinfoEndpoint)
}

func TestGetIssuer(t *testing.T) {
	svc := &Service{
		config: config.OAuthConfig{Issuer: "https://my-issuer.com"},
	}

	assert.Equal(t, "https://my-issuer.com", svc.GetIssuer())
}

func TestGetBaseURL(t *testing.T) {
	svc := &Service{
		config: config.OAuthConfig{BaseURL: "https://my-app.com"},
	}

	assert.Equal(t, "https://my-app.com", svc.GetBaseURL())
}

func TestProviderOIDCConstant(t *testing.T) {
	assert.Equal(t, "oidc", ProviderOIDC)
}
