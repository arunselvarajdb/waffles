package server

import (
	"net/http"
	"os"
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
	"github.com/waffles/mcp-gateway/internal/service/oauth"
	"github.com/waffles/mcp-gateway/internal/service/registry"
	"github.com/waffles/mcp-gateway/internal/service/serveraccess"
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
	namespaceRepo := repository.NewNamespaceRepository(s.db.Pool, s.logger)

	// Initialize services
	registryService := registry.NewService(serverRepo, s.logger)
	gatewayService := gateway.NewService(serverRepo, s.logger, s.metrics)
	auditService := audit.NewService(auditRepo, s.logger)

	// Initialize server access service only if RBAC is enabled
	// Support both new resource_rbac_enabled and legacy server_group_rbac_enabled
	var accessService *serveraccess.Service
	resourceRBACEnabled := s.config.Auth.ResourceRBACEnabled || s.config.Auth.ServerGroupRBACEnabled
	if resourceRBACEnabled {
		accessService = serveraccess.NewService(namespaceRepo, s.logger)
		s.logger.Info().Msg("Resource RBAC is ENABLED - users will only see servers they have access to")
	} else {
		s.logger.Info().Msg("Resource RBAC is DISABLED - all authenticated users see all servers")
	}

	// Initialize Casbin for authorization
	casbinService, err := authz.NewCasbinServiceWithDefaults(s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to initialize Casbin, using permissive mode")
	}

	// Initialize OAuth service (if enabled)
	oauthService := oauth.NewService(s.config.Auth.OAuth, s.logger)

	// Determine frontend URL for OAuth redirects
	frontendURL := s.config.Auth.OAuth.BaseURL
	if frontendURL == "" {
		frontendURL = "http://localhost:5173" // Default for development
	}

	// Initialize handlers
	registryHandler := handler.NewRegistryHandler(registryService, accessService, s.logger)
	gatewayHandler := handler.NewGatewayHandler(gatewayService, accessService, s.logger)
	authHandler := handler.NewAuthHandler(userRepo, s.logger)
	oauthHandler := handler.NewOAuthHandler(oauthService, userRepo, s.logger, frontendURL)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyRepo, s.logger)
	namespaceHandler := handler.NewNamespaceHandler(namespaceRepo, s.logger)
	oauthMetadataHandler := handler.NewOAuthMetadataHandler(s.config.Auth.OAuth, s.config.Auth.MCPAuth, s.logger)

	// Create OAuth service adapter for bearer token validation
	var oauthValidator middleware.OAuthValidator
	if oauthService.IsEnabled() {
		oauthValidator = middleware.NewOAuthServiceAdapter(oauthService)
	}

	// Auth middleware config
	authConfig := &middleware.AuthConfig{
		Logger:         s.logger,
		UserRepo:       userRepo,
		APIKeyRepo:     apiKeyRepo,
		OAuthValidator: oauthValidator,
		SessionName:    "mcp_session",
		MCPAuth: middleware.MCPAuthConfig{
			APIKeyEnabled:  s.config.Auth.MCPAuth.APIKeyEnabled,
			SessionEnabled: s.config.Auth.MCPAuth.SessionEnabled,
		},
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

	// OAuth Protected Resource Metadata (RFC 9728) - for MCP OAuth authorization
	// This must be a public endpoint for OAuth discovery
	// Support both exact path and path-based discovery (RFC 9115 style)
	s.router.GET("/.well-known/oauth-protected-resource", oauthMetadataHandler.GetProtectedResourceMetadata)
	s.router.GET("/.well-known/oauth-protected-resource/*path", oauthMetadataHandler.GetProtectedResourceMetadata)

	// OAuth Authorization Server Metadata - redirect to the actual auth server
	// MCP clients may also try path-based discovery for the authorization server
	s.router.GET("/.well-known/oauth-authorization-server", oauthMetadataHandler.GetAuthorizationServerMetadata)
	s.router.GET("/.well-known/oauth-authorization-server/*path", oauthMetadataHandler.GetAuthorizationServerMetadata)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Public auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)

			// SSO routes (public - they handle their own authentication)
			auth.GET("/sso/status", oauthHandler.GetSSOStatus)
			auth.GET("/sso", oauthHandler.Authorize)
			auth.GET("/sso/callback", oauthHandler.Callback)
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
					"sso": gin.H{
						"enabled": oauthService.IsEnabled(),
					},
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
				servers.POST("/call-tool", registryHandler.CallTool)             // Call tool for inspection
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

			// Namespaces routes (admin and operator can view, admin only can modify)
			namespaces := protected.Group("/namespaces")
			if authEnabled && authzConfig != nil {
				namespaces.Use(middleware.Authz(authzConfig))
			}
			{
				namespaces.GET("", namespaceHandler.ListNamespaces)
				namespaces.POST("", namespaceHandler.CreateNamespace)
				namespaces.GET("/:id", namespaceHandler.GetNamespace)
				namespaces.PUT("/:id", namespaceHandler.UpdateNamespace)
				namespaces.DELETE("/:id", namespaceHandler.DeleteNamespace)

				// Server membership management
				namespaces.GET("/:id/servers", namespaceHandler.ListServers)
				namespaces.POST("/:id/servers", namespaceHandler.AddServer)
				namespaces.DELETE("/:id/servers/:server_id", namespaceHandler.RemoveServer)

				// Role access management
				namespaces.GET("/:id/access", namespaceHandler.ListRoleAccess)
				namespaces.POST("/:id/access", namespaceHandler.SetRoleAccess)
				namespaces.DELETE("/:id/access/:role_id", namespaceHandler.RemoveRoleAccess)
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

	s.logger.Info().Bool("auth_enabled", authEnabled).Bool("resource_rbac_enabled", resourceRBACEnabled).Msg("Routes configured successfully")
}

// setupSessionStore creates and configures the session store
func (s *Server) setupSessionStore() sessions.Store {
	// Get session secret from config or use a default for development
	secret := s.config.Auth.SessionSecret
	if secret == "" {
		// This is a development-only fallback with a clear warning
		secret = "dev-session-secret-change-in-production-32b" // #nosec G101 -- intentional dev default
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
		HttpOnly: true,   // Prevent JavaScript access (XSS protection)
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
			"http://localhost:5173", // Vite dev server
			"http://localhost:3000", // Alternative dev port
			"http://127.0.0.1:5173",
			"http://127.0.0.1:3000",
		}

		// In production, add your frontend URL to allowedOrigins:
		// allowedOrigins = append(allowedOrigins, "https://your-frontend.com")

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

// setupStaticFileServing configures serving of Vue.js static files from filesystem.
func (s *Server) setupStaticFileServing() {
	staticDir := s.config.Server.StaticDir
	if staticDir == "" {
		s.logger.Info().Msg("No static_dir configured, UI will not be served")
		s.setupNoRouteHandler("")

		return
	}

	// Check if the directory exists
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		s.logger.Warn().Str("path", staticDir).Msg("Static files directory not found, UI will not be served")
		s.setupNoRouteHandler("")
		return
	}

	// Serve static assets (JS, CSS, images, etc.) from assets subdirectory
	assetsDir := filepath.Join(staticDir, "assets")
	if _, err := os.Stat(assetsDir); err == nil {
		s.router.Static("/assets", assetsDir)
	}

	// Setup SPA fallback with static directory
	s.setupNoRouteHandler(staticDir)

	s.logger.Info().Str("path", staticDir).Msg("Vue.js application configured to be served from filesystem")
}

// setupNoRouteHandler configures the NoRoute handler for SPA routing.
func (s *Server) setupNoRouteHandler(staticDir string) {
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

		// If no static directory, return 404
		if staticDir == "" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Not found",
				"path":  path,
			})

			return
		}

		// Check if requesting a static file
		if filepath.Ext(path) != "" && !strings.HasSuffix(path, ".html") {
			// Try to serve the file from static directory
			filePath := filepath.Join(staticDir, strings.TrimPrefix(path, "/"))
			if _, err := os.Stat(filePath); err == nil {
				c.File(filePath)
				return
			}
		}

		// Serve index.html for all other routes (SPA routing)
		indexPath := filepath.Join(staticDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to load application",
			})
			return
		}
		c.File(indexPath)
	})
}
