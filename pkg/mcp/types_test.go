package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolVersion(t *testing.T) {
	assert.Equal(t, "1.0.0", ProtocolVersion)
}

func TestInitializeRequest_JSONMarshaling(t *testing.T) {
	req := InitializeRequest{
		ProtocolVersion: "1.0.0",
		Capabilities: Capabilities{
			Tools:     true,
			Resources: true,
			Prompts:   false,
		},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed InitializeRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, req.ProtocolVersion, parsed.ProtocolVersion)
	assert.Equal(t, req.Capabilities.Tools, parsed.Capabilities.Tools)
	assert.Equal(t, req.ClientInfo.Name, parsed.ClientInfo.Name)
}

func TestInitializeResponse_JSONMarshaling(t *testing.T) {
	resp := InitializeResponse{
		ProtocolVersion: "1.0.0",
		Capabilities: Capabilities{
			Tools:     true,
			Resources: false,
			Prompts:   true,
		},
		ServerInfo: ServerInfo{
			Name:    "test-server",
			Version: "2.0.0",
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed InitializeResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, resp.ProtocolVersion, parsed.ProtocolVersion)
	assert.Equal(t, resp.ServerInfo.Name, parsed.ServerInfo.Name)
	assert.Equal(t, resp.ServerInfo.Version, parsed.ServerInfo.Version)
}

func TestCapabilities_OmitEmpty(t *testing.T) {
	cap := Capabilities{
		Tools:     true,
		Resources: false,
		Prompts:   false,
	}

	data, err := json.Marshal(cap)
	require.NoError(t, err)

	// When omitempty is used, false values should be omitted
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.True(t, parsed["tools"].(bool))
}

func TestTool_JSONMarshaling(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {"arg1": {"type": "string"}}}`),
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var parsed Tool
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, tool.Name, parsed.Name)
	assert.Equal(t, tool.Description, parsed.Description)
	assert.NotEmpty(t, parsed.InputSchema)
}

func TestToolsListRequest_WithCursor(t *testing.T) {
	req := ToolsListRequest{
		Cursor: "next-page-token",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed ToolsListRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "next-page-token", parsed.Cursor)
}

func TestToolsListResponse_WithTools(t *testing.T) {
	resp := ToolsListResponse{
		Tools: []Tool{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
		},
		NextCursor: "cursor-123",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed ToolsListResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Len(t, parsed.Tools, 2)
	assert.Equal(t, "tool1", parsed.Tools[0].Name)
	assert.Equal(t, "cursor-123", parsed.NextCursor)
}

func TestToolCallRequest_JSONMarshaling(t *testing.T) {
	req := ToolCallRequest{
		Name: "test_tool",
		Arguments: map[string]interface{}{
			"arg1": "value1",
			"arg2": 42,
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed ToolCallRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test_tool", parsed.Name)
	assert.Equal(t, "value1", parsed.Arguments["arg1"])
	assert.Equal(t, float64(42), parsed.Arguments["arg2"])
}

func TestToolCallResponse_JSONMarshaling(t *testing.T) {
	resp := ToolCallResponse{
		Content: []Content{
			{Type: "text", Text: "Result text"},
		},
		IsError: false,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed ToolCallResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Len(t, parsed.Content, 1)
	assert.Equal(t, "text", parsed.Content[0].Type)
	assert.Equal(t, "Result text", parsed.Content[0].Text)
	assert.False(t, parsed.IsError)
}

func TestToolCallResponse_IsError(t *testing.T) {
	resp := ToolCallResponse{
		Content: []Content{
			{Type: "text", Text: "Error occurred"},
		},
		IsError: true,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed ToolCallResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.True(t, parsed.IsError)
}

func TestContent_AllTypes(t *testing.T) {
	tests := []struct {
		content Content
		name    string
	}{
		{
			name: "text content",
			content: Content{
				Type: "text",
				Text: "Some text",
			},
		},
		{
			name: "image content",
			content: Content{
				Type:     "image",
				Data:     "base64-encoded-data",
				MimeType: "image/png",
			},
		},
		{
			name: "resource content",
			content: Content{
				Type:     "resource",
				URI:      "file:///path/to/resource",
				MimeType: "text/plain",
			},
		},
		{
			name: "content with meta",
			content: Content{
				Type: "text",
				Text: "With metadata",
				Meta: map[string]interface{}{"key": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.content)
			require.NoError(t, err)

			var parsed Content
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)

			assert.Equal(t, tt.content.Type, parsed.Type)
		})
	}
}

func TestResource_JSONMarshaling(t *testing.T) {
	resource := Resource{
		URI:         "file:///path/to/file.txt",
		Name:        "file.txt",
		Description: "A text file",
		MimeType:    "text/plain",
	}

	data, err := json.Marshal(resource)
	require.NoError(t, err)

	var parsed Resource
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, resource.URI, parsed.URI)
	assert.Equal(t, resource.Name, parsed.Name)
	assert.Equal(t, resource.Description, parsed.Description)
	assert.Equal(t, resource.MimeType, parsed.MimeType)
}

func TestResourcesListResponse_JSONMarshaling(t *testing.T) {
	resp := ResourcesListResponse{
		Resources: []Resource{
			{URI: "file:///a.txt", Name: "a.txt"},
			{URI: "file:///b.txt", Name: "b.txt"},
		},
		NextCursor: "next-cursor",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed ResourcesListResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Len(t, parsed.Resources, 2)
	assert.Equal(t, "next-cursor", parsed.NextCursor)
}

func TestResourceReadRequest_JSONMarshaling(t *testing.T) {
	req := ResourceReadRequest{
		URI: "file:///path/to/file.txt",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed ResourceReadRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, req.URI, parsed.URI)
}

func TestResourceReadResponse_JSONMarshaling(t *testing.T) {
	resp := ResourceReadResponse{
		Contents: []Content{
			{Type: "text", Text: "File contents"},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed ResourceReadResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Len(t, parsed.Contents, 1)
	assert.Equal(t, "File contents", parsed.Contents[0].Text)
}

func TestPrompt_JSONMarshaling(t *testing.T) {
	prompt := Prompt{
		Name:        "test_prompt",
		Description: "A test prompt",
		Arguments: []PromptArgument{
			{Name: "arg1", Description: "First argument", Required: true},
			{Name: "arg2", Description: "Second argument", Required: false},
		},
	}

	data, err := json.Marshal(prompt)
	require.NoError(t, err)

	var parsed Prompt
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, prompt.Name, parsed.Name)
	assert.Len(t, parsed.Arguments, 2)
	assert.True(t, parsed.Arguments[0].Required)
	assert.False(t, parsed.Arguments[1].Required)
}

func TestPromptsListResponse_JSONMarshaling(t *testing.T) {
	resp := PromptsListResponse{
		Prompts: []Prompt{
			{Name: "prompt1", Description: "First prompt"},
			{Name: "prompt2", Description: "Second prompt"},
		},
		NextCursor: "cursor-456",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed PromptsListResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Len(t, parsed.Prompts, 2)
	assert.Equal(t, "cursor-456", parsed.NextCursor)
}

func TestPromptsGetRequest_JSONMarshaling(t *testing.T) {
	req := PromptsGetRequest{
		Name: "test_prompt",
		Arguments: map[string]interface{}{
			"key": "value",
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed PromptsGetRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test_prompt", parsed.Name)
	assert.Equal(t, "value", parsed.Arguments["key"])
}

func TestPromptsGetResponse_JSONMarshaling(t *testing.T) {
	resp := PromptsGetResponse{
		Description: "Prompt description",
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{Type: "text", Text: "User message"},
				},
			},
			{
				Role: "assistant",
				Content: []Content{
					{Type: "text", Text: "Assistant response"},
				},
			},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed PromptsGetResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Prompt description", parsed.Description)
	assert.Len(t, parsed.Messages, 2)
	assert.Equal(t, "user", parsed.Messages[0].Role)
	assert.Equal(t, "assistant", parsed.Messages[1].Role)
}

func TestMessage_JSONMarshaling(t *testing.T) {
	msg := Message{
		Role: "user",
		Content: []Content{
			{Type: "text", Text: "Hello"},
			{Type: "image", Data: "base64data", MimeType: "image/png"},
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var parsed Message
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "user", parsed.Role)
	assert.Len(t, parsed.Content, 2)
}

func TestErrorResponse_JSONMarshaling(t *testing.T) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    "INVALID_REQUEST",
			Message: "The request was invalid",
			Data:    map[string]interface{}{"field": "name"},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed ErrorResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "INVALID_REQUEST", parsed.Error.Code)
	assert.Equal(t, "The request was invalid", parsed.Error.Message)
	assert.Equal(t, "name", parsed.Error.Data["field"])
}

func TestErrorDetail_OmitEmptyData(t *testing.T) {
	detail := ErrorDetail{
		Code:    "ERROR",
		Message: "An error occurred",
		Data:    nil,
	}

	data, err := json.Marshal(detail)
	require.NoError(t, err)

	// Check that nil data is omitted
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	_, hasData := parsed["data"]
	assert.False(t, hasData, "data field should be omitted when nil")
}
