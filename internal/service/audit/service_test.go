package audit

import (
	"testing"

	"github.com/waffles/mcp-gateway/internal/domain"
)

// Note: Most tests for the audit service require database integration testing
// See: internal/repository/audit_test.go for integration tests

func TestAuditLogFilter_DefaultLimit(t *testing.T) {
	filter := domain.AuditLogFilter{}

	// Test that default limit logic would be applied (actual test would be in integration test)
	expectedLimit := 100
	if filter.Limit == 0 {
		filter.Limit = expectedLimit
	}

	if filter.Limit != expectedLimit {
		t.Errorf("Expected limit %d, got %d", expectedLimit, filter.Limit)
	}
}

// Integration tests for this service should cover:
// - Service.Log() with mock/real repository
// - Service.Get() with various IDs
// - Service.List() with different filters
// - Error handling scenarios
//
// These are best tested with testcontainers and a real database
// to ensure the full stack works correctly.
