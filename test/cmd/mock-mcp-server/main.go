package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Session management for Streamable HTTP
var (
	sessions   = make(map[string]*Session)
	sessionsMu sync.RWMutex
	sessionID  atomic.Int64
)

type Session struct {
	ID        string
	CreatedAt time.Time
}

// JSON-RPC types
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	// Log all incoming requests for debugging
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[MOCK MCP] Received request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	// === HTTP/REST Transport Endpoints (Legacy) ===
	http.HandleFunc("/initialize", handleHTTPInitialize)
	http.HandleFunc("/tools/list", handleHTTPToolsList)
	http.HandleFunc("/tools/call", handleHTTPToolsCall)
	http.HandleFunc("/resources/list", handleHTTPResourcesList)
	http.HandleFunc("/health", handleHealth)

	// === SSE Transport Endpoints (Legacy, Deprecated) ===
	http.HandleFunc("/sse", handleSSEEndpoint)
	http.HandleFunc("/sse/message", handleSSEMessage)

	// === Streamable HTTP Transport Endpoint (MCP 2025-11-25) ===
	http.HandleFunc("/mcp", handleStreamableHTTP)

	port := ":9001"
	log.Printf("🚀 Mock MCP Server starting on http://localhost%s", port)
	log.Printf("")
	log.Printf("   === HTTP/REST Transport (Legacy) ===")
	log.Printf("   - POST /initialize")
	log.Printf("   - POST /tools/list")
	log.Printf("   - POST /tools/call")
	log.Printf("   - GET  /resources/list")
	log.Printf("   - GET  /health")
	log.Printf("")
	log.Printf("   === SSE Transport (Legacy, Deprecated) ===")
	log.Printf("   - GET  /sse          (SSE event stream)")
	log.Printf("   - POST /sse/message  (Send JSON-RPC request)")
	log.Printf("")
	log.Printf("   === Streamable HTTP Transport (MCP 2025-11-25) ===")
	log.Printf("   - POST /mcp          (JSON-RPC over HTTP)")
	log.Printf("   - GET  /mcp          (Server-initiated messages)")
	log.Printf("   - DELETE /mcp        (Terminate session)")
	log.Printf("")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start mock server: %v", err)
	}
}

// === HTTP/REST Handlers ===

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func handleHTTPInitialize(w http.ResponseWriter, r *http.Request) {
	log.Printf("[MOCK MCP] HTTP Initialize request from %s", r.RemoteAddr)
	response := map[string]interface{}{
		"protocolVersion": "1.0.0",
		"capabilities": map[string]bool{
			"tools":     true,
			"resources": true,
			"prompts":   false,
		},
		"serverInfo": map[string]string{
			"name":    "mock-mcp-server",
			"version": "1.0.0",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHTTPToolsList(w http.ResponseWriter, r *http.Request) {
	log.Printf("[MOCK MCP] HTTP Tools/list request from %s", r.RemoteAddr)
	response := getToolsResponse()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHTTPToolsCall(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	json.NewDecoder(r.Body).Decode(&request)

	toolName := request["name"]
	args := request["arguments"]

	log.Printf("[MOCK MCP] HTTP Tools/call request: tool=%s, args=%v", toolName, args)

	response := getToolCallResponse(toolName, args)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHTTPResourcesList(w http.ResponseWriter, r *http.Request) {
	log.Printf("[MOCK MCP] HTTP Resources/list request from %s", r.RemoteAddr)
	response := getResourcesResponse()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// === SSE Transport Handlers ===

func handleSSEEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Printf("[MOCK MCP] SSE connection from %s", r.RemoteAddr)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send endpoint info
	fmt.Fprintf(w, "event: endpoint\ndata: /sse/message\n\n")
	flusher.Flush()

	// Keep connection open
	<-r.Context().Done()
	log.Printf("[MOCK MCP] SSE connection closed from %s", r.RemoteAddr)
}

func handleSSEMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var rpcReq JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[MOCK MCP] SSE message: method=%s, id=%v", rpcReq.Method, rpcReq.ID)

	response := handleJSONRPCRequest(rpcReq)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// === Streamable HTTP Transport Handler (MCP 2025-11-25) ===

func handleStreamableHTTP(w http.ResponseWriter, r *http.Request) {
	// Check protocol version
	protocolVersion := r.Header.Get("MCP-Protocol-Version")
	log.Printf("[MOCK MCP] Streamable HTTP %s request, protocol=%s", r.Method, protocolVersion)

	switch r.Method {
	case "POST":
		handleStreamableHTTPPost(w, r)
	case "GET":
		handleStreamableHTTPGet(w, r)
	case "DELETE":
		handleStreamableHTTPDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleStreamableHTTPPost(w http.ResponseWriter, r *http.Request) {
	sessionIDHeader := r.Header.Get("MCP-Session-Id")

	var rpcReq JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[MOCK MCP] Streamable HTTP POST: method=%s, session=%s, id=%v",
		rpcReq.Method, sessionIDHeader, rpcReq.ID)

	// Handle initialize specially - create session
	if rpcReq.Method == "initialize" {
		newSessionID := fmt.Sprintf("session-%d", sessionID.Add(1))
		sessionsMu.Lock()
		sessions[newSessionID] = &Session{
			ID:        newSessionID,
			CreatedAt: time.Now(),
		}
		sessionsMu.Unlock()

		log.Printf("[MOCK MCP] Created new session: %s", newSessionID)

		// Set session ID in response header
		w.Header().Set("MCP-Session-Id", newSessionID)
	}

	// Handle notifications (no response expected)
	if rpcReq.Method == "notifications/initialized" {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	response := handleJSONRPCRequest(rpcReq)

	// Determine response format based on Accept header
	accept := r.Header.Get("Accept")
	if containsSSE(accept) && rpcReq.Method != "initialize" {
		// Return as SSE stream
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		data, _ := json.Marshal(response)
		fmt.Fprintf(w, "data: %s\n\n", data)
	} else {
		// Return as JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleStreamableHTTPGet(w http.ResponseWriter, r *http.Request) {
	sessionIDHeader := r.Header.Get("MCP-Session-Id")
	log.Printf("[MOCK MCP] Streamable HTTP GET: session=%s (server-initiated messages)", sessionIDHeader)

	// Verify session exists
	if sessionIDHeader != "" {
		sessionsMu.RLock()
		_, exists := sessions[sessionIDHeader]
		sessionsMu.RUnlock()

		if !exists {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
	}

	// Set SSE headers for server-initiated messages
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send a ping every 30 seconds to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Printf("[MOCK MCP] Streamable HTTP GET connection closed")
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func handleStreamableHTTPDelete(w http.ResponseWriter, r *http.Request) {
	sessionIDHeader := r.Header.Get("MCP-Session-Id")
	log.Printf("[MOCK MCP] Streamable HTTP DELETE: session=%s", sessionIDHeader)

	if sessionIDHeader != "" {
		sessionsMu.Lock()
		delete(sessions, sessionIDHeader)
		sessionsMu.Unlock()
		log.Printf("[MOCK MCP] Session terminated: %s", sessionIDHeader)
	}

	w.WriteHeader(http.StatusOK)
}

// === Shared Response Generators ===

func handleJSONRPCRequest(req JSONRPCRequest) JSONRPCResponse {
	var result interface{}

	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2025-11-25",
			"capabilities": map[string]interface{}{
				"tools":     map[string]bool{"listChanged": false},
				"resources": map[string]bool{"listChanged": false},
			},
			"serverInfo": map[string]string{
				"name":    "mock-mcp-server",
				"version": "1.0.0",
			},
		}

	case "tools/list":
		result = getToolsResponse()

	case "tools/call":
		params, _ := req.Params.(map[string]interface{})
		toolName := params["name"]
		args := params["arguments"]
		result = getToolCallResponse(toolName, args)

	case "resources/list":
		result = getResourcesResponse()

	case "prompts/list":
		result = map[string]interface{}{
			"prompts": []interface{}{},
		}

	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
			ID: req.ID,
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

func getToolsResponse() map[string]interface{} {
	return map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "calculator",
				"description": "Perform basic arithmetic operations",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"operation": map[string]string{"type": "string"},
						"a":         map[string]string{"type": "number"},
						"b":         map[string]string{"type": "number"},
					},
				},
			},
			{
				"name":        "echo",
				"description": "Echo back the input message",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"message": map[string]string{"type": "string"},
					},
				},
			},
		},
	}
}

func getToolCallResponse(toolName, args interface{}) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Mock result for tool '%s' with args: %v", toolName, args),
			},
		},
		"isError": false,
	}
}

func getResourcesResponse() map[string]interface{} {
	return map[string]interface{}{
		"resources": []map[string]interface{}{
			{
				"uri":         "file:///test/sample.txt",
				"name":        "Sample Text File",
				"description": "A sample text file for testing",
				"mimeType":    "text/plain",
			},
		},
	}
}

func containsSSE(accept string) bool {
	return accept != "" && (accept == "text/event-stream" ||
		len(accept) > 17 && accept[:17] == "text/event-stream")
}
