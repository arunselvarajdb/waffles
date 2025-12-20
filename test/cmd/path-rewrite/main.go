package main

import (
	"fmt"
	"strings"
)

func rewriteProxyPath(originalPath, serverID string) string {
	prefix := fmt.Sprintf("/api/v1/gateway/%s", serverID)
	if strings.HasPrefix(originalPath, prefix) {
		return strings.TrimPrefix(originalPath, prefix)
	}
	return originalPath
}

func main() {
	serverID := "test-server-id"
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "/api/v1/gateway/test-server-id/tools/list",
			expected: "/tools/list",
		},
		{
			input:    "/api/v1/gateway/test-server-id/tools/call",
			expected: "/tools/call",
		},
		{
			input:    "/api/v1/gateway/test-server-id/initialize",
			expected: "/initialize",
		},
	}

	for _, tc := range testCases {
		result := rewriteProxyPath(tc.input, serverID)
		status := "✅"
		if result != tc.expected {
			status = "❌"
		}
		fmt.Printf("%s Input: %s → Output: %s (expected: %s)\n", status, tc.input, result, tc.expected)
	}
}
