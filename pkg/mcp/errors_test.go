package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPError_Error_WithCode(t *testing.T) {
	err := &MCPError{
		Code:    "TOOL_NOT_FOUND",
		Message: "The requested tool was not found",
	}

	result := err.Error()

	assert.Equal(t, "MCP error [TOOL_NOT_FOUND]: The requested tool was not found", result)
}

func TestMCPError_Error_WithoutCode(t *testing.T) {
	err := &MCPError{
		Code:    "",
		Message: "Something went wrong",
	}

	result := err.Error()

	assert.Equal(t, "Something went wrong", result)
}

func TestMCPError_Error_EmptyMessage(t *testing.T) {
	err := &MCPError{
		Code:    "ERROR_CODE",
		Message: "",
	}

	result := err.Error()

	assert.Equal(t, "MCP error [ERROR_CODE]: ", result)
}

func TestNewMCPError(t *testing.T) {
	tests := []struct {
		data    map[string]interface{}
		name    string
		code    string
		message string
	}{
		{
			name:    "with all fields",
			code:    "TEST_ERROR",
			message: "Test error message",
			data:    map[string]interface{}{"key": "value"},
		},
		{
			name:    "with nil data",
			code:    "ANOTHER_ERROR",
			message: "Another message",
			data:    nil,
		},
		{
			name:    "with empty values",
			code:    "",
			message: "",
			data:    map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMCPError(tt.code, tt.message, tt.data)

			require.NotNil(t, err)
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.data, err.Data)
		})
	}
}

func TestFromErrorResponse(t *testing.T) {
	tests := []struct {
		response *ErrorResponse
		name     string
	}{
		{
			name: "converts error response with all fields",
			response: &ErrorResponse{
				Error: ErrorDetail{
					Code:    "INTERNAL_ERROR",
					Message: "Internal server error",
					Data:    map[string]interface{}{"details": "more info"},
				},
			},
		},
		{
			name: "converts error response with minimal fields",
			response: &ErrorResponse{
				Error: ErrorDetail{
					Code:    "ERROR",
					Message: "Error occurred",
				},
			},
		},
		{
			name: "converts error response with empty values",
			response: &ErrorResponse{
				Error: ErrorDetail{
					Code:    "",
					Message: "",
					Data:    nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromErrorResponse(tt.response)

			require.NotNil(t, err)
			assert.Equal(t, tt.response.Error.Code, err.Code)
			assert.Equal(t, tt.response.Error.Message, err.Message)
			assert.Equal(t, tt.response.Error.Data, err.Data)
		})
	}
}

func TestMCPError_ImplementsError(t *testing.T) {
	var err error = NewMCPError("TEST", "test message", nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "TEST")
}

func TestCommonErrors_AreDistinct(t *testing.T) {
	errors := []error{
		ErrConnectionFailed,
		ErrInitializeFailed,
		ErrToolNotFound,
		ErrToolCallFailed,
		ErrResourceNotFound,
		ErrResourceReadFailed,
		ErrPromptNotFound,
		ErrPromptGetFailed,
		ErrInvalidResponse,
		ErrUnsupportedVersion,
		ErrMissingCapability,
	}

	// Check that each error has a unique message
	seen := make(map[string]bool)
	for _, err := range errors {
		msg := err.Error()
		assert.False(t, seen[msg], "duplicate error message: %s", msg)
		seen[msg] = true
	}
}

func TestCommonErrors_Messages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{"ErrConnectionFailed", ErrConnectionFailed, "connect"},
		{"ErrInitializeFailed", ErrInitializeFailed, "initialize"},
		{"ErrToolNotFound", ErrToolNotFound, "tool not found"},
		{"ErrToolCallFailed", ErrToolCallFailed, "tool call failed"},
		{"ErrResourceNotFound", ErrResourceNotFound, "resource not found"},
		{"ErrResourceReadFailed", ErrResourceReadFailed, "read resource"},
		{"ErrPromptNotFound", ErrPromptNotFound, "prompt not found"},
		{"ErrPromptGetFailed", ErrPromptGetFailed, "get prompt"},
		{"ErrInvalidResponse", ErrInvalidResponse, "invalid response"},
		{"ErrUnsupportedVersion", ErrUnsupportedVersion, "version"},
		{"ErrMissingCapability", ErrMissingCapability, "capability"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.err.Error(), tt.contains)
		})
	}
}
