package mcp

import (
	"errors"
	"fmt"
)

// Common MCP errors
var (
	ErrConnectionFailed   = errors.New("failed to connect to MCP server")
	ErrInitializeFailed   = errors.New("failed to initialize MCP connection")
	ErrToolNotFound       = errors.New("tool not found")
	ErrToolCallFailed     = errors.New("tool call failed")
	ErrResourceNotFound   = errors.New("resource not found")
	ErrResourceReadFailed = errors.New("failed to read resource")
	ErrPromptNotFound     = errors.New("prompt not found")
	ErrPromptGetFailed    = errors.New("failed to get prompt")
	ErrInvalidResponse    = errors.New("invalid response from MCP server")
	ErrUnsupportedVersion = errors.New("unsupported protocol version")
	ErrMissingCapability  = errors.New("server does not support required capability")
)

// MCPError represents an error from an MCP server
type MCPError struct {
	Code    string
	Message string
	Data    map[string]interface{}
}

func (e *MCPError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("MCP error [%s]: %s", e.Code, e.Message)
	}
	return e.Message
}

// NewMCPError creates a new MCP error
func NewMCPError(code, message string, data map[string]interface{}) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// FromErrorResponse converts an ErrorResponse to an MCPError
func FromErrorResponse(errResp *ErrorResponse) *MCPError {
	return &MCPError{
		Code:    errResp.Error.Code,
		Message: errResp.Error.Message,
		Data:    errResp.Error.Data,
	}
}
