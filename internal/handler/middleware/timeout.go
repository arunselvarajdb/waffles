package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout returns a middleware that adds request timeout
// Skips timeout for streaming/SSE endpoints like the gateway proxy
func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip timeout for gateway proxy routes (they use SSE/streaming)
		// These routes handle their own timeouts or are long-lived connections
		if strings.Contains(c.Request.URL.Path, "/gateway/") {
			c.Next()
			return
		}

		// Skip timeout for SSE requests
		if strings.Contains(c.GetHeader("Accept"), "text/event-stream") {
			c.Next()
			return
		}

		// Create timeout context
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Channel to track completion
		done := make(chan struct{})

		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// Request completed successfully
			return
		case <-ctx.Done():
			// Timeout occurred - only abort if headers haven't been written
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
					"error": "Request timeout",
				})
			}
		}
	}
}
