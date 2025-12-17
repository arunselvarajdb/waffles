package domain

import (
	"time"
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"` // Never expose key hash
	Prefix     string     `json:"prefix"` // First 8 chars for identification
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// APIKeyCreate represents the data required to create a new API key
type APIKeyCreate struct {
	Name      string     `json:"name" validate:"required,min=3,max=255"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKeyResponse includes the plain-text key (only shown once)
type APIKeyResponse struct {
	APIKey
	Key string `json:"key"` // Plain-text key, only returned on creation
}
