package server

import (
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/waffles/mcp-gateway/internal/handler"
	"github.com/waffles/mcp-gateway/internal/handler/middleware"
	"github.com/waffles/mcp-gateway/internal/repository"
	"github.com/waffles/mcp-gateway/internal/service/audit"
	"github.com/waffles/mcp-gateway/internal/service/authz"
	"github.com/waffles/mcp-gateway/internal/service/gateway"
	"github.com/waffles/mcp-gateway/internal/service/registry"
)

// SetupRoutes configures all routes for the server
func (s *Server) SetupRoutes() {
	// Apply global middleware
	s.router.Use(middleware.Recovery(s.logger))
	// Metrics middleware - positioned after recovery, before logging for accurate timing
	if s.metrics != nil {
		s.router.Use(middleware.Metrics(s.metrics))
	}
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.Logger(s.logger))
	s.router.Use(s.corsWithCredentials()) // Updated CORS for cookie auth
	s.router.Use(middleware.Timeout(30 * time.Second))

	// Setup session store
	sessionStore := s.setupSessionStore()
	s.router.Use(sessions.Sessions("mcp_session", sessionStore))

	// Create health handler
	healthHandler := handler.NewHealthHandler(s.db, s.logger)

	// Health check endpoints (public)
	s.router.GET("/health", healthHandler.Health)
	s.router.GET("/ready", healthHandler.Ready)

	// Initialize repositories
	serverRepo := repository.NewServerRepository(s.db.Pool, s.logger)
	auditRepo := repository.NewAuditRepository(s.db.Pool)
	userRepo := repository.NewUserRepository(s.db.Pool, s.logger)
	apiKeyRepo := repository.NewAPIKeyRepository(s.db.Pool, s.logger)

	// Initialize services
	registryService := registry.NewService(serverRepo, s.logger)
	gatewayService := gateway.NewService(serverRepo, s.logger, s.metrics)
	auditService := audit.NewService(auditRepo, s.logger)

	// Initialize Casbin for authorization
	casbinService, err := authz.NewCasbinServiceWithDefaults(s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to initialize Casbin, using permissive mode")
	}

	// Initialize handlers
	registryHandler := handler.NewRegistryHandler(registryService, s.logger)
	gatewayHandler := handler.NewGatewayHandler(gatewayService, s.logger)
	authHandler := handler.NewAuthHandler(userRepo, s.logger)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyRepo, s.logger)

	// Auth middleware config
	authConfig := &middleware.AuthConfig{
		Logger:      s.logger,
		UserRepo:    userRepo,
		APIKeyRepo:  apiKeyRepo,
		SessionName: "mcp_session",
	}

	// Authz middleware config
	var authzConfig *middleware.AuthzConfig
	if casbinService != nil {
		authzConfig = &middleware.AuthzConfig{
			Logger:   s.logger,
			Enforcer: casbinService.GetEnforcer(),
		}
	}

	// Check if authentication is enabled
	authEnabled := s.config.Auth.Enabled

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Public auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		// Status endpoint (public) - includes auth config for frontend
		v1.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"database": gin.H{
					"connected": s.db != nil,
				},
				"auth": gin.H{
					"enabled": authEnabled,
				},
			})
		})

		// Protected routes - require authentication only if enabled
		protected := v1.Group("")
		if authEnabled {
			protected.Use(middleware.CombinedAuth(authConfig))
		}
		{
			// Current user info
			protected.GET("/me", authHandler.GetCurrentUser)
			protected.PUT("/auth/password", authHandler.ChangePassword)

			// API key management (users can manage their own keys)
			apiKeys := protected.Group("/api-keys")
			{
				apiKeys.GET("", apiKeyHandler.ListAPIKeys)
				apiKeys.POST("", apiKeyHandler.CreateAPIKey)
				apiKeys.GET("/:id", apiKeyHandler.GetAPIKey)
				apiKeys.DELETE("/:id", apiKeyHandler.DeleteAPIKey)
			}

			// MCP Server Registry routes
			servers := protected.Group("/servers")
			if authEnabled && authzConfig != nil {
				servers.Use(middleware.Authz(authzConfig))
			}
			{
				servers.GET("", registryHandler.ListServers)
				servers.POST("", registryHandler.CreateServer)
				servers.POST("/test-connection", registryHandler.TestConnection) // Test connection without saving
				servers.POST("/call-tool", registryHandler.CallTool)           // Call tool for inspection
				servers.GET("/:id", registryHandler.GetServer)
				servers.PUT("/:id", registryHandler.UpdateServer)
				servers.DELETE("/:id", registryHandler.DeleteServer)
				servers.PATCH("/:id/toggle", registryHandler.ToggleServer)
				servers.GET("/:id/health", registryHandler.GetHealthStatus)
				servers.POST("/:id/health", registryHandler.CheckHealth)
			}

			// MCP Gateway Proxy routes (with audit middleware)
			gatewayGroup := protected.Group("/gateway")
			gatewayGroup.Use(middleware.AuditMiddleware(auditService))
			if authEnabled && authzConfig != nil {
				gatewayGroup.Use(middleware.Authz(authzConfig))
			}
			{
				// Native MCP proxy endpoint - allows MCP clients (Claude Code, etc.) to connect directly
				// This proxies MCP JSON-RPC requests to the backend server
				gatewayGroup.Any("/:server_id", gatewayHandler.MCPProxy)

				// REST-style endpoints for programmatic access
				gatewayGroup.POST("/:server_id/initialize", gatewayHandler.Initialize)
				gatewayGroup.POST("/:server_id/tools/list", gatewayHandler.ListTools)
				gatewayGroup.POST("/:server_id/tools/call", gatewayHandler.CallTool)
				gatewayGroup.GET("/:server_id/resources/list", gatewayHandler.ListResources)
				gatewayGroup.GET("/:server_id/resources/read", gatewayHandler.ReadResource)
				gatewayGroup.POST("/:server_id/prompts/list", gatewayHandler.ListPrompts)
				gatewayGroup.POST("/:server_id/prompts/get", gatewayHandler.GetPrompt)
			}
		}
	}

	// Log auth status
	if authEnabled {
		s.logger.Info().Msg("Authentication is ENABLED - login required")
	} else {
		s.logger.Warn().Msg("Authentication is DISABLED - all routes are public (development mode)")
	}

	// Serve Vue.js application (SPA)
	s.setupStaticFileServing()

	s.logger.Info().Bool("auth_enabled", authEnabled).Msg("Routes configured successfully")
}

// setupSessionStore creates and configures the session store
func (s *Server) setupSessionStore() sessions.Store {
	// Get session secret from config or use a default for development
	secret := s.config.Auth.SessionSecret
	if secret == "" {
		secret = "dev-session-secret-change-in-production-32b"
		s.logger.Warn().Msg("Using default session secret - please set auth.session_secret in production")
	}

	// Create cookie-based session store
	store := cookie.NewStore([]byte(secret))

	// Configure session options
	maxAge := int(s.config.Auth.SessionMaxAge.Seconds())
	if maxAge == 0 {
		maxAge = 86400 // Default: 24 hours
	}

	// Determine secure flag based on environment
	secure := s.config.Auth.CookieSecure
	if s.config.Server.Environment == "development" {
		secure = false // Allow non-HTTPS in development
	}

	// Determine SameSite mode
	sameSite := http.SameSiteStrictMode
	switch strings.ToLower(s.config.Auth.CookieSameSite) {
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}

	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,  // Prevent JavaScript access (XSS protection)
		Secure:   secure, // HTTPS only in production
		SameSite: sameSite,
		Domain:   s.config.Auth.CookieDomain,
	})

	s.logger.Info().
		Int("max_age", maxAge).
		Bool("secure", secure).
		Str("same_site", s.config.Auth.CookieSameSite).
		Msg("Session store configured")

	return store
}

// corsWithCredentials returns CORS middleware configured for cookie-based auth
func (s *Server) corsWithCredentials() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// In development, allow the frontend dev server
		allowedOrigins := []string{
			"http://localhost:5173",  // Vite dev server
			"http://localhost:3000",  // Alternative dev port
			"http://127.0.0.1:5173",
			"http://127.0.0.1:3000",
		}

		// In production, you'd set this from config
		if s.config.Server.Environment == "production" {
			// Add production frontend URL
			// allowedOrigins = append(allowedOrigins, "https://your-frontend.com")
		}

		// Check if origin is allowed
		allowed := false
		for _, o := range allowedOrigins {
			if o == origin {
				allowed = true
				break
			}
		}

		// If same-origin request or allowed origin
		if origin == "" || allowed {
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID, X-API-Key")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// setupStaticFileServing configures serving of Vue.js static files
func (s *Server) setupStaticFileServing() {
	// Extract dist subfolder from embedded FS
	distFS, err := fs.Sub(s.webAppFS, "dist")
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load web-app dist folder, static files will not be served")
		return
	}

	// Serve static assets (JS, CSS, images, etc.)
	s.router.StaticFS("/assets", http.FS(distFS))

	// SPA fallback - serve index.html for all non-API routes
	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Don't handle API routes
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "API endpoint not found",
				"path":  path,
			})
			return
		}

		// Don't handle health check routes
		if path == "/health" || path == "/ready" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Endpoint not found",
				"path":  path,
			})
			return
		}

		// Check if requesting a static file
		if filepath.Ext(path) != "" && !strings.HasSuffix(path, ".html") {
			// Try to serve the file from dist
			file, err := distFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				defer file.Close()
				stat, _ := file.Stat()
				http.ServeContent(c.Writer, c.Request, path, stat.ModTime(), file.(io.ReadSeeker))
				return
			}
		}

		// Serve index.html for all other routes (SPA routing)
		indexFile, err := distFS.Open("index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to load application",
			})
			return
		}
		defer indexFile.Close()

		stat, _ := indexFile.Stat()
		c.Header("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(c.Writer, c.Request, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
	})

	s.logger.Info().Msg("Vue.js application configured to be served from embedded files")
}
