package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/waffles/mcp-gateway/internal/config"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// ProviderOIDC is the provider name stored for OIDC-authenticated users
const ProviderOIDC = "oidc"

// UserInfo represents the user information obtained from the OIDC provider
type UserInfo struct {
	ID       string // Provider's unique user ID (sub claim)
	Email    string
	Name     string
	Provider string
}

// OIDCDiscovery represents the OIDC discovery document
// See: https://openid.net/specs/openid-connect-discovery-1_0.html
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

// Service handles OAuth/OIDC authentication
// Works with any OIDC-compliant provider (Authentik, Keycloak, Okta, Auth0, Azure AD, etc.)
type Service struct {
	config       config.OAuthConfig
	oauth2Config *oauth2.Config
	discovery    *OIDCDiscovery
	logger       logger.Logger
}

// NewService creates a new OAuth service for any OIDC provider
func NewService(cfg config.OAuthConfig, log logger.Logger) *Service {
	s := &Service{
		config: cfg,
		logger: log,
	}

	if !cfg.Enabled {
		log.Info().Msg("SSO/OIDC is disabled")
		return s
	}

	if cfg.Issuer == "" {
		log.Warn().Msg("OIDC issuer not configured, SSO will not work")
		return s
	}

	// Fetch OIDC discovery document
	discovery, err := s.fetchOIDCDiscovery(cfg.Issuer)
	if err != nil {
		log.Error().Err(err).Str("issuer", cfg.Issuer).Msg("Failed to fetch OIDC discovery document")
		return s
	}
	s.discovery = discovery

	// Default scopes if not configured
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	// Create OAuth2 config
	s.oauth2Config = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  discovery.AuthorizationEndpoint,
			TokenURL: discovery.TokenEndpoint,
		},
		RedirectURL: fmt.Sprintf("%s/api/v1/auth/sso/callback", cfg.BaseURL),
	}

	log.Info().
		Str("issuer", cfg.Issuer).
		Str("redirect_url", s.oauth2Config.RedirectURL).
		Msg("OIDC SSO configured")

	return s
}

// fetchOIDCDiscovery fetches the OIDC discovery document from the issuer
func (s *Service) fetchOIDCDiscovery(issuer string) (*OIDCDiscovery, error) {
	// OIDC discovery endpoint is always at /.well-known/openid-configuration
	discoveryURL := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"

	resp, err := http.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read discovery document: %w", err)
	}

	var discovery OIDCDiscovery
	if err := json.Unmarshal(body, &discovery); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	return &discovery, nil
}

// IsEnabled returns whether OAuth is enabled and configured
func (s *Service) IsEnabled() bool {
	return s.config.Enabled && s.oauth2Config != nil
}

// GenerateState generates a random state string for CSRF protection
func (s *Service) GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthURL returns the OAuth authorization URL
func (s *Service) GetAuthURL(state string) (string, error) {
	if s.oauth2Config == nil {
		return "", fmt.Errorf("OAuth is not configured")
	}
	return s.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

// ExchangeCode exchanges the authorization code for a token and fetches user info
func (s *Service) ExchangeCode(ctx context.Context, code string) (*UserInfo, error) {
	if s.oauth2Config == nil {
		return nil, fmt.Errorf("OAuth is not configured")
	}

	// Exchange code for token
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Fetch user info from userinfo endpoint
	userInfo, err := s.fetchUserInfo(ctx, token)
	if err != nil {
		return nil, err
	}

	// Validate email domain if configured
	if err := s.validateEmailDomain(userInfo.Email); err != nil {
		return nil, err
	}

	return userInfo, nil
}

// fetchUserInfo fetches user information from the OIDC userinfo endpoint
func (s *Service) fetchUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	if s.discovery == nil || s.discovery.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("userinfo endpoint not available")
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get(s.discovery.UserinfoEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	var info struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	// Use name or fall back to preferred_username
	name := info.Name
	if name == "" {
		name = info.PreferredUsername
	}

	return &UserInfo{
		ID:       info.Sub,
		Email:    info.Email,
		Name:     name,
		Provider: ProviderOIDC,
	}, nil
}

// validateEmailDomain validates that the email domain is allowed
func (s *Service) validateEmailDomain(email string) error {
	allowedDomains := s.config.AllowedDomains

	// If no domains specified, allow all
	if len(allowedDomains) == 0 {
		return nil
	}

	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format")
	}
	domain := strings.ToLower(parts[1])

	// Check if domain is in allowed list
	for _, allowed := range allowedDomains {
		if strings.ToLower(allowed) == domain {
			return nil
		}
	}

	s.logger.Warn().
		Str("email", email).
		Str("allowed_domains", strings.Join(allowedDomains, ",")).
		Msg("Email domain not allowed")

	return fmt.Errorf("email domain %s is not allowed", domain)
}

// GetDefaultRole returns the default role for new OAuth users
func (s *Service) GetDefaultRole() string {
	if s.config.DefaultRole == "" {
		return "viewer"
	}
	return s.config.DefaultRole
}

// AutoCreateUsers returns whether to auto-create users on first OAuth login
func (s *Service) AutoCreateUsers() bool {
	return s.config.AutoCreateUsers
}

// ValidateBearerToken validates an OAuth bearer token by calling the userinfo endpoint
// Returns user info if the token is valid, error otherwise
// This allows MCP clients to authenticate with OAuth access tokens
func (s *Service) ValidateBearerToken(ctx context.Context, bearerToken string) (*UserInfo, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("OAuth is not enabled")
	}

	if s.discovery == nil || s.discovery.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("userinfo endpoint not available")
	}

	// Create a token source with the bearer token
	token := &oauth2.Token{
		AccessToken: bearerToken,
		TokenType:   "Bearer",
	}

	// Use the token to fetch user info
	userInfo, err := s.fetchUserInfo(ctx, token)
	if err != nil {
		s.logger.Debug().Err(err).Msg("Bearer token validation failed")
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Validate email domain if configured
	if err := s.validateEmailDomain(userInfo.Email); err != nil {
		return nil, err
	}

	s.logger.Debug().
		Str("email", userInfo.Email).
		Str("name", userInfo.Name).
		Msg("Bearer token validated successfully")

	return userInfo, nil
}

// GetIssuer returns the OIDC issuer URL
func (s *Service) GetIssuer() string {
	return s.config.Issuer
}

// GetBaseURL returns the OAuth base URL
func (s *Service) GetBaseURL() string {
	return s.config.BaseURL
}
