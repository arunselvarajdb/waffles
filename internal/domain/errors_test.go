package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		message  string
		expected string
	}{
		{
			name:     "standard validation error",
			field:    "email",
			message:  "must be a valid email address",
			expected: "validation error on field 'email': must be a valid email address",
		},
		{
			name:     "empty field name",
			field:    "",
			message:  "is required",
			expected: "validation error on field '': is required",
		},
		{
			name:     "empty message",
			field:    "name",
			message:  "",
			expected: "validation error on field 'name': ",
		},
		{
			name:     "field with special characters",
			field:    "user.address.city",
			message:  "cannot be empty",
			expected: "validation error on field 'user.address.city': cannot be empty",
		},
		{
			name:     "message with quotes",
			field:    "status",
			message:  "must be one of 'active', 'inactive'",
			expected: "validation error on field 'status': must be one of 'active', 'inactive'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ValidationError{
				Field:   tt.field,
				Message: tt.message,
			}
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		message string
	}{
		{
			name:    "creates error with field and message",
			field:   "password",
			message: "must be at least 8 characters",
		},
		{
			name:    "creates error with empty values",
			field:   "",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.message)

			require.NotNil(t, err)
			assert.Equal(t, tt.field, err.Field)
			assert.Equal(t, tt.message, err.Message)
		})
	}
}

func TestValidationError_ImplementsError(t *testing.T) {
	var err error = NewValidationError("test", "test message")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test")
}

func TestDomainErrors_AreDistinct(t *testing.T) {
	// Ensure all domain errors are distinct
	errors := []error{
		ErrNotFound,
		ErrUserNotFound,
		ErrUserAlreadyExists,
		ErrInvalidCredentials,
		ErrUserInactive,
		ErrServerNotFound,
		ErrServerAlreadyExists,
		ErrServerUnhealthy,
		ErrAPIKeyNotFound,
		ErrAPIKeyExpired,
		ErrAPIKeyInvalid,
		ErrTokenInvalid,
		ErrTokenExpired,
		ErrTokenMalformed,
		ErrUnauthorized,
		ErrForbidden,
		ErrValidationFailed,
		ErrInvalidInput,
	}

	// Check that each error has a unique message
	seen := make(map[string]bool)
	for _, err := range errors {
		msg := err.Error()
		assert.False(t, seen[msg], "duplicate error message: %s", msg)
		seen[msg] = true
	}
}

func TestDomainErrors_Messages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{"ErrNotFound", ErrNotFound, "not found"},
		{"ErrUserNotFound", ErrUserNotFound, "user not found"},
		{"ErrServerNotFound", ErrServerNotFound, "server not found"},
		{"ErrAPIKeyNotFound", ErrAPIKeyNotFound, "API key not found"},
		{"ErrAPIKeyExpired", ErrAPIKeyExpired, "expired"},
		{"ErrUnauthorized", ErrUnauthorized, "unauthorized"},
		{"ErrForbidden", ErrForbidden, "forbidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.err.Error(), tt.contains)
		})
	}
}
