package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/service/audit"
)

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// AuditMiddleware creates a middleware for audit logging
func AuditMiddleware(auditService *audit.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID if not present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header("X-Request-ID", requestID)
		}

		// Store request ID in context for later use
		c.Set("request_id", requestID)

		// Capture request body
		var requestBody json.RawMessage
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil && len(bodyBytes) > 0 {
				// Restore body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Only store body if it's JSON
				if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
					requestBody = bodyBytes
				}
			}
		}

		// Capture query params as JSON
		var queryParams json.RawMessage
		if len(c.Request.URL.RawQuery) > 0 {
			params := make(map[string][]string)
			for key, values := range c.Request.URL.Query() {
				params[key] = values
			}
			queryParams, _ = json.Marshal(params)
		}

		// Wrap response writer to capture response body
		blw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := int(time.Since(start).Milliseconds())

		// Extract server_id from path if this is a gateway request
		var serverID *string
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/gateway/") {
			if sid := c.Param("server_id"); sid != "" {
				serverID = &sid
			}
		}

		// Capture response body (only if JSON and not too large)
		var responseBody json.RawMessage
		if blw.body.Len() > 0 && blw.body.Len() < 10000 { // Max 10KB
			if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
				responseBody = blw.body.Bytes()
			}
		}

		// Capture error message if any
		var errorMessage *string
		if len(c.Errors) > 0 {
			errMsg := c.Errors.String()
			errorMessage = &errMsg
		}

		// Create audit log entry
		status := c.Writer.Status()
		auditLog := &domain.AuditLog{
			RequestID:      requestID,
			ServerID:       serverID,
			Method:         c.Request.Method,
			Path:           c.Request.URL.Path,
			QueryParams:    queryParams,
			RequestBody:    requestBody,
			ResponseStatus: &status,
			ResponseBody:   responseBody,
			LatencyMS:      &latency,
			IPAddress:      c.ClientIP(),
			UserAgent:      c.Request.UserAgent(),
			ErrorMessage:   errorMessage,
		}

		// Log asynchronously to avoid blocking response
		// Use background context since request context will be canceled
		go func() {
			ctx := context.Background()
			_ = auditService.Log(ctx, auditLog) // Error already logged in service
		}()
	}
}
