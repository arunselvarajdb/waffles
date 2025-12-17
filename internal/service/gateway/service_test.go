package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRewriteProxyPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		serverID   string
		wantPath   string
	}{
		{
			name:      "tools/list endpoint",
			path:      "/api/v1/gateway/server-123/tools/list",
			serverID:  "server-123",
			wantPath:  "/tools/list",
		},
		{
			name:      "tools/call endpoint",
			path:      "/api/v1/gateway/abc-def/tools/call",
			serverID:  "abc-def",
			wantPath:  "/tools/call",
		},
		{
			name:      "initialize endpoint",
			path:      "/api/v1/gateway/test-id/initialize",
			serverID:  "test-id",
			wantPath:  "/initialize",
		},
		{
			name:      "resources/list endpoint",
			path:      "/api/v1/gateway/my-server/resources/list",
			serverID:  "my-server",
			wantPath:  "/resources/list",
		},
		{
			name:      "path with query params",
			path:      "/api/v1/gateway/srv1/tools/list?filter=calc",
			serverID:  "srv1",
			wantPath:  "/tools/list?filter=calc",
		},
		{
			name:      "prompts/list endpoint",
			path:      "/api/v1/gateway/server-xyz/prompts/list",
			serverID:  "server-xyz",
			wantPath:  "/prompts/list",
		},
		{
			name:      "resources/read endpoint",
			path:      "/api/v1/gateway/srv-abc/resources/read",
			serverID:  "srv-abc",
			wantPath:  "/resources/read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath := rewriteProxyPath(tt.path, tt.serverID)
			assert.Equal(t, tt.wantPath, gotPath, "path should be correctly rewritten")
		})
	}
}

func TestRewriteProxyPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		serverID string
		wantPath string
	}{
		{
			name:     "server ID with dashes",
			path:     "/api/v1/gateway/my-test-server-123/tools/call",
			serverID: "my-test-server-123",
			wantPath: "/tools/call",
		},
		{
			name:     "server ID with underscores",
			path:     "/api/v1/gateway/test_server_1/initialize",
			serverID: "test_server_1",
			wantPath: "/initialize",
		},
		{
			name:     "nested path",
			path:     "/api/v1/gateway/server-123/some/nested/path",
			serverID: "server-123",
			wantPath: "/some/nested/path",
		},
		{
			name:     "path with multiple query params",
			path:     "/api/v1/gateway/srv1/tools/list?filter=calc&limit=10&offset=0",
			serverID: "srv1",
			wantPath: "/tools/list?filter=calc&limit=10&offset=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath := rewriteProxyPath(tt.path, tt.serverID)
			assert.Equal(t, tt.wantPath, gotPath, "path should handle edge cases")
		})
	}
}

// Note: Integration tests for ProxyToServer, injectAuth, etc. should be in
// integration_test.go with a real database using testcontainers
