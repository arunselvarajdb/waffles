package domain

import (
	"time"
)

// AuthProvider represents the authentication provider type
type AuthProvider string

const (
	AuthProviderLocal AuthProvider = "local"
	AuthProviderOAuth AuthProvider = "oauth"
	AuthProviderLDAP  AuthProvider = "ldap"
)

// User represents a user in the system
type User struct {
	ID           string       `json:"id"`
	Email        string       `json:"email"`
	PasswordHash string       `json:"-"` // Never expose password hash in JSON
	Name         string       `json:"name"`
	AuthProvider AuthProvider `json:"auth_provider"`
	ExternalID   string       `json:"external_id,omitempty"`
	IsActive     bool         `json:"is_active"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// UserCreate represents the data required to create a new user
type UserCreate struct {
	Email        string       `json:"email" validate:"required,email"`
	Password     string       `json:"password" validate:"required,min=8"`
	Name         string       `json:"name" validate:"required"`
	AuthProvider AuthProvider `json:"auth_provider,omitempty"`
}

// UserUpdate represents the data that can be updated for a user
type UserUpdate struct {
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Name     *string `json:"name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// UserLogin represents login credentials
type UserLogin struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// TokenPair represents an access token and refresh token
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"` // "Bearer"
}

// TokenClaims represents the claims in a JWT token
type TokenClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
}
