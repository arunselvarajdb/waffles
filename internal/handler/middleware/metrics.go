package middleware

import (
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/waffles/mcp-gateway/internal/metrics"
)

// uuidRegex matches UUID v4 format
var uuidRegex = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// numericIDRegex matches numeric IDs
var numericIDRegex = regexp.MustCompile(`/\d+(/|$)`)

// Metrics creates a middleware that tracks HTTP request metrics
func Metrics(metricsRegistry *metrics.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()

		// Increment in-flight requests
		metricsRegistry.HTTPRequestsInFlight.Inc()
		defer metricsRegistry.HTTPRequestsInFlight.Dec()

		// Process request
		c.Next()

		// Record request metrics
		duration := time.Since(start).Seconds()
		path := normalizePath(c.Request.URL.Path)
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())

		// Record metrics
		metricsRegistry.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metricsRegistry.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}

// normalizePath normalizes URL paths to prevent cardinality explosion
// It replaces UUIDs and numeric IDs with placeholders
//
// Examples:
//   - /api/v1/servers/550e8400-e29b-41d4-a716-446655440000 -> /api/v1/servers/:id
//   - /api/v1/gateway/550e8400-e29b-41d4-a716-446655440000/tools/call -> /api/v1/gateway/:id/tools/call
//   - /api/v1/servers/123 -> /api/v1/servers/:id
func normalizePath(path string) string {
	// Replace UUIDs with :id
	normalized := uuidRegex.ReplaceAllString(path, ":id")

	// Replace numeric IDs with :id
	normalized = numericIDRegex.ReplaceAllString(normalized, "/:id$1")

	return normalized
}
