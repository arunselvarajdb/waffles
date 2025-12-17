package mcp

import "encoding/json"

// ProtocolVersion represents the MCP protocol version
const ProtocolVersion = "1.0.0"

// InitializeRequest represents the MCP initialize request
type InitializeRequest struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ClientInfo      ClientInfo   `json:"clientInfo"`
}

// InitializeResponse represents the MCP initialize response
type InitializeResponse struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// Capabilities represents MCP capabilities
type Capabilities struct {
	Tools     bool `json:"tools,omitempty"`
	Resources bool `json:"resources,omitempty"`
	Prompts   bool `json:"prompts,omitempty"`
}

// ClientInfo represents information about the MCP client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo represents information about the MCP server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsListRequest represents a request to list tools
type ToolsListRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ToolsListResponse represents the response from listing tools
type ToolsListResponse struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolCallRequest represents a request to call a tool
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResponse represents the response from calling a tool
type ToolCallResponse struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content in MCP responses
type Content struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	Data     string                 `json:"data,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	URI      string                 `json:"uri,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

// ResourcesListRequest represents a request to list resources
type ResourcesListRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ResourcesListResponse represents the response from listing resources
type ResourcesListResponse struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceReadRequest represents a request to read a resource
type ResourceReadRequest struct {
	URI string `json:"uri"`
}

// ResourceReadResponse represents the response from reading a resource
type ResourceReadResponse struct {
	Contents []Content `json:"contents"`
}

// PromptsListRequest represents a request to list prompts
type PromptsListRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// PromptsListResponse represents the response from listing prompts
type PromptsListResponse struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents an argument for a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptsGetRequest represents a request to get a prompt
type PromptsGetRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// PromptsGetResponse represents the response from getting a prompt
type PromptsGetResponse struct {
	Description string    `json:"description,omitempty"`
	Messages    []Message `json:"messages"`
}

// Message represents a message in a prompt
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// ErrorResponse represents an MCP error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents the error details
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
