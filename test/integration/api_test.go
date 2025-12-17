// Package integration provides integration tests for the MCP Gateway API
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// APITestSuite is the main test suite for API integration tests
type APITestSuite struct {
	suite.Suite
	baseURL    string
	client     *http.Client
	serverIDs  map[string]string // Track created server IDs for cleanup
}

// SetupSuite runs once before all tests
func (s *APITestSuite) SetupSuite() {
	s.baseURL = os.Getenv("API_BASE_URL")
	if s.baseURL == "" {
		s.baseURL = "http://localhost:8080"
	}

	// Create HTTP client with cookie jar for session management
	jar, err := cookiejar.New(nil)
	require.NoError(s.T(), err)

	s.client = &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	s.serverIDs = make(map[string]string)

	// Wait for gateway to be ready
	s.waitForGateway()

	// Authenticate
	s.authenticate()
}

// TearDownSuite runs once after all tests
func (s *APITestSuite) TearDownSuite() {
	// Cleanup created servers
	for name, id := range s.serverIDs {
		s.T().Logf("Cleaning up server: %s (%s)", name, id)
		s.deleteServer(id)
	}
}

// waitForGateway waits for the gateway to be ready
func (s *APITestSuite) waitForGateway() {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := s.client.Get(s.baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	s.T().Fatal("Gateway failed to become ready")
}

// authenticate logs in as admin user
func (s *APITestSuite) authenticate() {
	loginReq := map[string]string{
		"email":    "admin@example.com",
		"password": "admin123",
	}
	resp := s.doRequest("POST", "/api/v1/auth/login", loginReq)
	defer resp.Body.Close()
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

// doRequest performs an HTTP request and returns the response
func (s *APITestSuite) doRequest(method, path string, body interface{}) *http.Response {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(s.T(), err)
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, s.baseURL+path, bodyReader)
	require.NoError(s.T(), err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	require.NoError(s.T(), err)
	return resp
}

// parseResponse parses JSON response body into target
func (s *APITestSuite) parseResponse(resp *http.Response, target interface{}) {
	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	err = json.Unmarshal(body, target)
	require.NoError(s.T(), err, "Failed to parse response: %s", string(body))
}

// deleteServer deletes a server by ID
func (s *APITestSuite) deleteServer(id string) {
	resp := s.doRequest("DELETE", "/api/v1/servers/"+id, nil)
	resp.Body.Close()
}

// ============================================================================
// Authentication Tests
// ============================================================================

func (s *APITestSuite) TestAuth_Login() {
	// Create new client without cookies to test fresh login
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}

	loginReq := map[string]string{
		"email":    "admin@example.com",
		"password": "admin123",
	}
	body, _ := json.Marshal(loginReq)
	resp, err := client.Post(s.baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)
	assert.NotNil(s.T(), result["user"])
}

func (s *APITestSuite) TestAuth_LoginInvalidCredentials() {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}

	loginReq := map[string]string{
		"email":    "admin@example.com",
		"password": "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)
	resp, err := client.Post(s.baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (s *APITestSuite) TestAuth_Me() {
	// The /me endpoint is under protected routes, not auth routes
	resp := s.doRequest("GET", "/api/v1/me", nil)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)
	// Check for email in user object or directly
	if user, ok := result["user"].(map[string]interface{}); ok {
		assert.Equal(s.T(), "admin@example.com", user["email"])
	} else {
		assert.Equal(s.T(), "admin@example.com", result["email"])
	}
}

// ============================================================================
// Server Management Tests
// ============================================================================

func (s *APITestSuite) TestServer_Create() {
	serverReq := map[string]interface{}{
		"name":        "test-server-create",
		"description": "Test server for creation test",
		"url":         "http://mock-mcp:9001",
		"transport":   "http",
		"tags":        []string{"test"},
	}

	resp := s.doRequest("POST", "/api/v1/servers", serverReq)
	defer resp.Body.Close()

	// Accept both 200 OK and 201 Created
	assert.True(s.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
		"Expected 200 or 201, got %d", resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	id := result["id"].(string)
	s.serverIDs["test-server-create"] = id

	assert.NotEmpty(s.T(), id)
	assert.Equal(s.T(), "test-server-create", result["name"])
	assert.Equal(s.T(), "http", result["transport"])
}

func (s *APITestSuite) TestServer_List() {
	resp := s.doRequest("GET", "/api/v1/servers", nil)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	servers := result["servers"].([]interface{})
	assert.GreaterOrEqual(s.T(), len(servers), 0)
}

func (s *APITestSuite) TestServer_Get() {
	// First create a server
	serverReq := map[string]interface{}{
		"name":        "test-server-get",
		"description": "Test server for get test",
		"url":         "http://mock-mcp:9001",
		"transport":   "http",
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)
	s.serverIDs["test-server-get"] = id

	// Now get the server
	resp := s.doRequest("GET", "/api/v1/servers/"+id, nil)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)
	assert.Equal(s.T(), "test-server-get", result["name"])
}

func (s *APITestSuite) TestServer_Delete() {
	// Create a server to delete
	serverReq := map[string]interface{}{
		"name":        "test-server-delete",
		"description": "Test server for delete test",
		"url":         "http://mock-mcp:9001",
		"transport":   "http",
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)

	// Delete the server
	resp := s.doRequest("DELETE", "/api/v1/servers/"+id, nil)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	// Verify it's deleted
	getResp := s.doRequest("GET", "/api/v1/servers/"+id, nil)
	defer getResp.Body.Close()
	assert.Equal(s.T(), http.StatusNotFound, getResp.StatusCode)
}

// ============================================================================
// Transport Tests
// ============================================================================

func (s *APITestSuite) TestTransport_HTTP() {
	// Register HTTP server
	serverReq := map[string]interface{}{
		"name":        "test-http-transport",
		"description": "HTTP transport test",
		"url":         "http://mock-mcp:9001",
		"transport":   "http",
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)
	s.serverIDs["test-http-transport"] = id

	// Initialize
	initResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/initialize", id), nil)
	defer initResp.Body.Close()
	assert.Equal(s.T(), http.StatusOK, initResp.StatusCode)

	// Call tool
	toolReq := map[string]interface{}{
		"name": "calculator",
		"arguments": map[string]interface{}{
			"operation": "add",
			"a":         5,
			"b":         3,
		},
	}
	callResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer callResp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, callResp.StatusCode)

	var callResult map[string]interface{}
	s.parseResponse(callResp, &callResult)
	assert.NotNil(s.T(), callResult["content"])
}

func (s *APITestSuite) TestTransport_StreamableHTTP() {
	// Register Streamable HTTP server
	serverReq := map[string]interface{}{
		"name":             "test-streamable-http-transport",
		"description":      "Streamable HTTP transport test",
		"url":              "http://mock-mcp:9001/mcp",
		"protocol_version": "2025-11-25",
		"transport":        "streamable_http",
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)
	s.serverIDs["test-streamable-http-transport"] = id

	// Verify transport persisted
	getResp := s.doRequest("GET", "/api/v1/servers/"+id, nil)
	var getResult map[string]interface{}
	s.parseResponse(getResp, &getResult)
	getResp.Body.Close()
	assert.Equal(s.T(), "streamable_http", getResult["transport"])

	// Initialize
	initResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/initialize", id), nil)
	defer initResp.Body.Close()
	assert.Equal(s.T(), http.StatusOK, initResp.StatusCode)

	// Call tool
	toolReq := map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": "Hello Streamable HTTP",
		},
	}
	callResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer callResp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, callResp.StatusCode)

	var callResult map[string]interface{}
	s.parseResponse(callResp, &callResult)
	assert.NotNil(s.T(), callResult["content"])
}

func (s *APITestSuite) TestTransport_SSE() {
	// Register SSE server
	serverReq := map[string]interface{}{
		"name":        "test-sse-transport",
		"description": "SSE transport test",
		"url":         "http://mock-mcp:9001/sse",
		"transport":   "sse",
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)
	s.serverIDs["test-sse-transport"] = id

	// Initialize
	initResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/initialize", id), nil)
	defer initResp.Body.Close()
	assert.Equal(s.T(), http.StatusOK, initResp.StatusCode)

	// Call tool
	toolReq := map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": "Hello SSE",
		},
	}
	callResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer callResp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, callResp.StatusCode)

	var callResult map[string]interface{}
	s.parseResponse(callResp, &callResult)
	assert.NotNil(s.T(), callResult["content"])
}

// ============================================================================
// MCP Tools Verification Tests
// ============================================================================

// createTestServer creates a server and returns its ID
func (s *APITestSuite) createTestServer(name, url, transport string) string {
	serverReq := map[string]interface{}{
		"name":        name,
		"description": "Test server for MCP tools verification",
		"url":         url,
		"transport":   transport,
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)
	s.serverIDs[name] = id

	// Initialize the server
	initResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/initialize", id), nil)
	initResp.Body.Close()

	return id
}

func (s *APITestSuite) TestMCPTools_ListTools() {
	// Create and initialize server
	id := s.createTestServer("test-tools-list", "http://mock-mcp:9001", "http")

	// List tools - use POST to /tools/list as per router
	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/list", id), nil)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	tools := result["tools"].([]interface{})
	assert.GreaterOrEqual(s.T(), len(tools), 2, "Expected at least 2 tools (calculator, echo)")

	// Verify tool names exist
	toolNames := make(map[string]bool)
	for _, t := range tools {
		tool := t.(map[string]interface{})
		toolNames[tool["name"].(string)] = true
	}

	assert.True(s.T(), toolNames["calculator"], "Expected 'calculator' tool")
	assert.True(s.T(), toolNames["echo"], "Expected 'echo' tool")
}

func (s *APITestSuite) TestMCPTools_CalculatorAdd() {
	id := s.createTestServer("test-calc-add", "http://mock-mcp:9001", "http")

	toolReq := map[string]interface{}{
		"name": "calculator",
		"arguments": map[string]interface{}{
			"operation": "add",
			"a":         10,
			"b":         5,
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	// Verify response structure
	assert.NotNil(s.T(), result["content"], "Expected 'content' in response")
	assert.False(s.T(), result["isError"].(bool), "Expected isError to be false")

	// Verify content contains expected data
	content := result["content"].([]interface{})
	assert.Greater(s.T(), len(content), 0, "Expected at least one content item")

	firstContent := content[0].(map[string]interface{})
	assert.Equal(s.T(), "text", firstContent["type"])
	assert.Contains(s.T(), firstContent["text"].(string), "calculator")
}

func (s *APITestSuite) TestMCPTools_CalculatorSubtract() {
	id := s.createTestServer("test-calc-sub", "http://mock-mcp:9001", "http")

	toolReq := map[string]interface{}{
		"name": "calculator",
		"arguments": map[string]interface{}{
			"operation": "subtract",
			"a":         20,
			"b":         8,
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	assert.NotNil(s.T(), result["content"])
	assert.False(s.T(), result["isError"].(bool))
}

func (s *APITestSuite) TestMCPTools_CalculatorMultiply() {
	id := s.createTestServer("test-calc-mult", "http://mock-mcp:9001", "http")

	toolReq := map[string]interface{}{
		"name": "calculator",
		"arguments": map[string]interface{}{
			"operation": "multiply",
			"a":         7,
			"b":         6,
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	assert.NotNil(s.T(), result["content"])
	assert.False(s.T(), result["isError"].(bool))
}

func (s *APITestSuite) TestMCPTools_CalculatorDivide() {
	id := s.createTestServer("test-calc-div", "http://mock-mcp:9001", "http")

	toolReq := map[string]interface{}{
		"name": "calculator",
		"arguments": map[string]interface{}{
			"operation": "divide",
			"a":         100,
			"b":         4,
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	assert.NotNil(s.T(), result["content"])
	assert.False(s.T(), result["isError"].(bool))
}

func (s *APITestSuite) TestMCPTools_Echo() {
	id := s.createTestServer("test-echo", "http://mock-mcp:9001", "http")

	testMessage := "Hello, MCP Gateway!"
	toolReq := map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": testMessage,
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	assert.NotNil(s.T(), result["content"])
	assert.False(s.T(), result["isError"].(bool))

	// Verify echo response contains our message
	content := result["content"].([]interface{})
	firstContent := content[0].(map[string]interface{})
	assert.Contains(s.T(), firstContent["text"].(string), testMessage)
}

func (s *APITestSuite) TestMCPTools_EchoEmptyMessage() {
	id := s.createTestServer("test-echo-empty", "http://mock-mcp:9001", "http")

	toolReq := map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": "",
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	assert.NotNil(s.T(), result["content"])
}

func (s *APITestSuite) TestMCPTools_EchoSpecialCharacters() {
	id := s.createTestServer("test-echo-special", "http://mock-mcp:9001", "http")

	specialMessage := "Hello! @#$%^&*() 日本語 emoji: 🚀"
	toolReq := map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": specialMessage,
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	assert.NotNil(s.T(), result["content"])
	content := result["content"].([]interface{})
	firstContent := content[0].(map[string]interface{})
	assert.Contains(s.T(), firstContent["text"].(string), specialMessage)
}

func (s *APITestSuite) TestMCPTools_ListResources() {
	id := s.createTestServer("test-resources-list", "http://mock-mcp:9001", "http")

	// Use GET to /resources/list as per router
	resp := s.doRequest("GET", fmt.Sprintf("/api/v1/gateway/%s/resources/list", id), nil)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	resources := result["resources"].([]interface{})
	assert.GreaterOrEqual(s.T(), len(resources), 1, "Expected at least 1 resource")

	// Verify resource structure
	firstResource := resources[0].(map[string]interface{})
	assert.NotEmpty(s.T(), firstResource["uri"])
	assert.NotEmpty(s.T(), firstResource["name"])
}

func (s *APITestSuite) TestMCPTools_StreamableHTTPToolsList() {
	// Test tools list via Streamable HTTP transport
	serverReq := map[string]interface{}{
		"name":             "test-streamable-tools-list",
		"description":      "Streamable HTTP tools list test",
		"url":              "http://mock-mcp:9001/mcp",
		"protocol_version": "2025-11-25",
		"transport":        "streamable_http",
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)
	s.serverIDs["test-streamable-tools-list"] = id

	// Initialize
	initResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/initialize", id), nil)
	initResp.Body.Close()

	// List tools - use POST to /tools/list as per router
	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/list", id), nil)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	tools := result["tools"].([]interface{})
	assert.GreaterOrEqual(s.T(), len(tools), 2)
}

func (s *APITestSuite) TestMCPTools_StreamableHTTPCalculator() {
	// Test calculator via Streamable HTTP transport
	serverReq := map[string]interface{}{
		"name":             "test-streamable-calc",
		"description":      "Streamable HTTP calculator test",
		"url":              "http://mock-mcp:9001/mcp",
		"protocol_version": "2025-11-25",
		"transport":        "streamable_http",
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)
	s.serverIDs["test-streamable-calc"] = id

	// Initialize
	initResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/initialize", id), nil)
	initResp.Body.Close()

	// Call calculator
	toolReq := map[string]interface{}{
		"name": "calculator",
		"arguments": map[string]interface{}{
			"operation": "add",
			"a":         100,
			"b":         200,
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	assert.NotNil(s.T(), result["content"])
	assert.False(s.T(), result["isError"].(bool))
}

func (s *APITestSuite) TestMCPTools_SSEEcho() {
	// Test echo via SSE transport
	serverReq := map[string]interface{}{
		"name":        "test-sse-echo",
		"description": "SSE echo test",
		"url":         "http://mock-mcp:9001/sse",
		"transport":   "sse",
	}

	createResp := s.doRequest("POST", "/api/v1/servers", serverReq)
	var createResult map[string]interface{}
	s.parseResponse(createResp, &createResult)
	createResp.Body.Close()

	id := createResult["id"].(string)
	s.serverIDs["test-sse-echo"] = id

	// Initialize
	initResp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/initialize", id), nil)
	initResp.Body.Close()

	// Call echo
	toolReq := map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": "SSE Transport Test",
		},
	}

	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	assert.NotNil(s.T(), result["content"])
}

func (s *APITestSuite) TestMCPTools_MultipleCallsSequential() {
	id := s.createTestServer("test-multi-calls", "http://mock-mcp:9001", "http")

	// Make multiple sequential calls
	operations := []struct {
		op string
		a  int
		b  int
	}{
		{"add", 1, 2},
		{"subtract", 10, 5},
		{"multiply", 3, 4},
		{"divide", 20, 4},
	}

	for _, op := range operations {
		toolReq := map[string]interface{}{
			"name": "calculator",
			"arguments": map[string]interface{}{
				"operation": op.op,
				"a":         op.a,
				"b":         op.b,
			},
		}

		resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/tools/call", id), toolReq)
		assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "Failed on operation: %s", op.op)

		var result map[string]interface{}
		s.parseResponse(resp, &result)
		assert.NotNil(s.T(), result["content"])
		resp.Body.Close()
	}
}

func (s *APITestSuite) TestMCPTools_ServerCapabilities() {
	id := s.createTestServer("test-capabilities", "http://mock-mcp:9001", "http")

	// The gateway initialize endpoint returns server registration info, not raw MCP capabilities
	// This is the gateway's response format for initialization confirmation
	resp := s.doRequest("POST", fmt.Sprintf("/api/v1/gateway/%s/initialize", id), nil)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	s.parseResponse(resp, &result)

	// Gateway Initialize returns: server_id, server_name, url, status
	assert.NotEmpty(s.T(), result["server_id"], "Expected 'server_id' in response")
	assert.NotEmpty(s.T(), result["server_name"], "Expected 'server_name' in response")
	assert.NotEmpty(s.T(), result["url"], "Expected 'url' in response")
	assert.Equal(s.T(), "initialized", result["status"], "Expected status to be 'initialized'")
}

// ============================================================================
// Test Runner
// ============================================================================

func TestAPIIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TEST=1 to run.")
	}
	suite.Run(t, new(APITestSuite))
}
