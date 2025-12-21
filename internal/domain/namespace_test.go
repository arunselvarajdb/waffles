package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessLevel_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		level    AccessLevel
		expected bool
	}{
		{
			name:     "view is valid",
			level:    AccessLevelView,
			expected: true,
		},
		{
			name:     "execute is valid",
			level:    AccessLevelExecute,
			expected: true,
		},
		{
			name:     "empty string is invalid",
			level:    AccessLevel(""),
			expected: false,
		},
		{
			name:     "unknown level is invalid",
			level:    AccessLevel("admin"),
			expected: false,
		},
		{
			name:     "uppercase VIEW is invalid",
			level:    AccessLevel("VIEW"),
			expected: false,
		},
		{
			name:     "mixed case Execute is invalid",
			level:    AccessLevel("Execute"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccessLevel_Includes(t *testing.T) {
	tests := []struct {
		name     string
		level    AccessLevel
		other    AccessLevel
		expected bool
	}{
		{
			name:     "execute includes execute",
			level:    AccessLevelExecute,
			other:    AccessLevelExecute,
			expected: true,
		},
		{
			name:     "execute includes view",
			level:    AccessLevelExecute,
			other:    AccessLevelView,
			expected: true,
		},
		{
			name:     "view includes view",
			level:    AccessLevelView,
			other:    AccessLevelView,
			expected: true,
		},
		{
			name:     "view does not include execute",
			level:    AccessLevelView,
			other:    AccessLevelExecute,
			expected: false,
		},
		{
			name:     "view does not include unknown",
			level:    AccessLevelView,
			other:    AccessLevel("unknown"),
			expected: false,
		},
		{
			name:     "execute includes unknown (execute includes all)",
			level:    AccessLevelExecute,
			other:    AccessLevel("unknown"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.Includes(tt.other)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccessLevel_Constants(t *testing.T) {
	assert.Equal(t, AccessLevel("view"), AccessLevelView)
	assert.Equal(t, AccessLevel("execute"), AccessLevelExecute)
}
