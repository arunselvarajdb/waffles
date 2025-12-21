package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/waffles/mcp-gateway/pkg/logger"
)

// Logger returns a middleware that logs HTTP requests
func Logger(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Get request ID if available
		requestID, _ := c.Get(RequestIDKey)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build log event
		event := log.Info()

		// Add request ID if available
		if requestID != nil {
			event = event.Str("request_id", requestID.(string))
		}

		event.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("query", c.Request.URL.RawQuery).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Int("response_size", c.Writer.Size()).
			Msg("HTTP request completed")

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				log.Error().
					Err(err.Err).
					Any("type", err.Type).
					Msg("Request error")
			}
		}
	}
}
