package domain

import (
	"errors"
	"fmt"
)

// Common domain errors
var (
	// Generic errors
	ErrNotFound = errors.New("not found")

	// User errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")

	// Server errors
	ErrServerNotFound      = errors.New("server not found")
	ErrServerAlreadyExists = errors.New("server with this name already exists")
	ErrServerUnhealthy     = errors.New("server is unhealthy")

	// API Key errors
	ErrAPIKeyNotFound = errors.New("API key not found")
	ErrAPIKeyExpired  = errors.New("API key has expired")
	ErrAPIKeyInvalid  = errors.New("invalid API key")

	// Token errors
	ErrTokenInvalid   = errors.New("invalid token")
	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenMalformed = errors.New("malformed token")

	// Permission errors
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden: insufficient permissions")

	// Validation errors
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidInput     = errors.New("invalid input")
)

// ValidationError represents a validation error with field-specific details
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
