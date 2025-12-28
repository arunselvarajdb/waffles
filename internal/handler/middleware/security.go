package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders creates a middleware that adds standard HTTP security headers
// to all responses. These headers help protect against common web vulnerabilities.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks by disallowing the page to be framed
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing attacks
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS filter in browsers (legacy but still useful for older browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS connections for 1 year, including subdomains
		// Only set in production to avoid issues with local development
		if c.GetHeader("X-Forwarded-Proto") == "https" || c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Control how referrer information is sent
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Prevent caching of sensitive data
		// Applied to API responses; static assets may override this
		if isAPIRequest(c) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Header("Pragma", "no-cache")
		}

		// Content Security Policy - restrictive by default for API
		// Frontend pages will need to set their own CSP via meta tags or override
		if isAPIRequest(c) {
			c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		}

		// Explicitly disable permissions for sensitive browser features
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// isAPIRequest checks if the request is for an API endpoint
func isAPIRequest(c *gin.Context) bool {
	path := c.Request.URL.Path
	return len(path) >= 4 && path[:4] == "/api"
}

// SecurityHeadersConfig allows customizing security header behavior
type SecurityHeadersConfig struct {
	// EnableHSTS enables Strict-Transport-Security header
	// Set to false for development environments
	EnableHSTS bool

	// HSTSMaxAge is the max-age value for HSTS in seconds (default: 31536000 = 1 year)
	HSTSMaxAge int

	// HSTSIncludeSubDomains includes subdomains in HSTS policy
	HSTSIncludeSubDomains bool

	// HSTSPreload adds preload directive (requires domain to be submitted to preload list)
	HSTSPreload bool

	// ContentSecurityPolicy allows overriding the default CSP
	// Empty string uses the default restrictive policy
	ContentSecurityPolicy string

	// FrameOptions controls X-Frame-Options header
	// Valid values: "DENY", "SAMEORIGIN", or empty to disable
	FrameOptions string
}

// DefaultSecurityHeadersConfig returns the default security headers configuration
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubDomains: true,
		HSTSPreload:           false,
		FrameOptions:          "DENY",
	}
}

// SecurityHeadersWithConfig creates a middleware with custom configuration
func SecurityHeadersWithConfig(cfg SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// X-Frame-Options
		if cfg.FrameOptions != "" {
			c.Header("X-Frame-Options", cfg.FrameOptions)
		}

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS Protection (legacy)
		c.Header("X-XSS-Protection", "1; mode=block")

		// HSTS - only on HTTPS connections
		if cfg.EnableHSTS && (c.GetHeader("X-Forwarded-Proto") == "https" || c.Request.TLS != nil) {
			hsts := "max-age=" + string(rune(cfg.HSTSMaxAge))
			if cfg.HSTSIncludeSubDomains {
				hsts += "; includeSubDomains"
			}
			if cfg.HSTSPreload {
				hsts += "; preload"
			}
			c.Header("Strict-Transport-Security", hsts)
		}

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Cache Control for API requests
		if isAPIRequest(c) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Header("Pragma", "no-cache")
		}

		// Content Security Policy
		if isAPIRequest(c) {
			if cfg.ContentSecurityPolicy != "" {
				c.Header("Content-Security-Policy", cfg.ContentSecurityPolicy)
			} else {
				c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			}
		}

		// Permissions Policy
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
