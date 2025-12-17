package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for request ID
	RequestIDKey = "request_id"
)

// RequestID returns a middleware that adds a unique request ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			// Generate new UUID if not provided
			requestID = uuid.New().String()
		}

		// Set request ID in context
		c.Set(RequestIDKey, requestID)

		// Set request ID in response header
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}
