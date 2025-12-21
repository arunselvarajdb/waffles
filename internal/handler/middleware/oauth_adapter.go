package middleware

import (
	"context"

	"github.com/waffles/mcp-gateway/internal/service/oauth"
)

// OAuthServiceAdapter adapts oauth.Service to the OAuthValidator interface
type OAuthServiceAdapter struct {
	service *oauth.Service
}

// NewOAuthServiceAdapter creates a new adapter for oauth.Service
func NewOAuthServiceAdapter(service *oauth.Service) *OAuthServiceAdapter {
	return &OAuthServiceAdapter{service: service}
}

// ValidateBearerToken validates an OAuth bearer token
func (a *OAuthServiceAdapter) ValidateBearerToken(ctx context.Context, token string) (*OAuthUserInfo, error) {
	if a.service == nil {
		return nil, nil
	}
	userInfo, err := a.service.ValidateBearerToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return &OAuthUserInfo{
		ID:       userInfo.ID,
		Email:    userInfo.Email,
		Name:     userInfo.Name,
		Provider: userInfo.Provider,
	}, nil
}

// IsEnabled returns whether OAuth is enabled
func (a *OAuthServiceAdapter) IsEnabled() bool {
	if a.service == nil {
		return false
	}
	return a.service.IsEnabled()
}

// GetIssuer returns the OIDC issuer URL
func (a *OAuthServiceAdapter) GetIssuer() string {
	if a.service == nil {
		return ""
	}
	return a.service.GetIssuer()
}

// GetBaseURL returns the OAuth base URL
func (a *OAuthServiceAdapter) GetBaseURL() string {
	if a.service == nil {
		return ""
	}
	return a.service.GetBaseURL()
}

// GetDefaultRole returns the default role for OAuth users
func (a *OAuthServiceAdapter) GetDefaultRole() string {
	if a.service == nil {
		return ""
	}
	return a.service.GetDefaultRole()
}

// AutoCreateUsers returns whether to auto-create users
func (a *OAuthServiceAdapter) AutoCreateUsers() bool {
	if a.service == nil {
		return false
	}
	return a.service.AutoCreateUsers()
}
