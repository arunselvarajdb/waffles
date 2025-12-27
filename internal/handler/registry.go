package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/handler/middleware"
	"github.com/waffles/waffles/internal/service/registry"
	"github.com/waffles/waffles/internal/service/serveraccess"
	"github.com/waffles/waffles/pkg/logger"
)

// Note: registry package imported for TestConnectionRequest type

// RegistryHandler handles HTTP requests for MCP server registry
type RegistryHandler struct {
	service       RegistryServiceInterface
	accessService ServerAccessServiceInterface
	logger        logger.Logger
}

// NewRegistryHandler creates a new registry handler
func NewRegistryHandler(service *registry.Service, accessService *serveraccess.Service, log logger.Logger) *RegistryHandler {
	var svc RegistryServiceInterface
	var accessSvc ServerAccessServiceInterface

	if service != nil {
		svc = service
	}
	if accessService != nil {
		accessSvc = accessService
	}

	return &RegistryHandler{
		service:       svc,
		accessService: accessSvc,
		logger:        log,
	}
}

// NewRegistryHandlerWithInterfaces creates a new registry handler with interface dependencies (for testing).
func NewRegistryHandlerWithInterfaces(service RegistryServiceInterface, accessService ServerAccessServiceInterface, log logger.Logger) *RegistryHandler {
	return &RegistryHandler{
		service:       service,
		accessService: accessService,
		logger:        log,
	}
}

// ListServers handles GET /api/v1/servers
func (h *RegistryHandler) ListServers(c *gin.Context) {
	// Parse query parameters
	filter := &domain.ServerFilter{
		Name: c.Query("name"),
	}

	// Parse is_active filter
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid is_active parameter",
			})
			return
		}
		filter.IsActive = &isActive
	}

	// Parse tags filter
	if tags := c.QueryArray("tags"); len(tags) > 0 {
		filter.Tags = tags
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid limit parameter (must be 1-100)",
			})
			return
		}
		filter.Limit = limit
	} else {
		filter.Limit = 20 // Default limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid offset parameter",
			})
			return
		}
		filter.Offset = offset
	}

	// Get user's roles for access filtering
	roles := middleware.GetUserRoles(c)

	// Get accessible server IDs (nil = admin, all servers; empty = no access; list = filtered)
	var accessibleServerIDs []string
	var err error
	if h.accessService != nil {
		accessibleServerIDs, err = h.accessService.GetAccessibleServerIDs(c.Request.Context(), roles, domain.AccessLevelView)
		if err != nil {
			h.logger.Error().Err(err).Any("roles", roles).Msg("Failed to get accessible servers")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check server access",
			})
			return
		}
	}

	// Call service with access filter
	servers, err := h.service.ListServersForUser(c.Request.Context(), filter, accessibleServerIDs)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list servers")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list servers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"servers": servers,
		"count":   len(servers),
	})
}

// CreateServer handles POST /api/v1/servers
func (h *RegistryHandler) CreateServer(c *gin.Context) {
	var req domain.ServerCreate

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Create server
	server, err := h.service.CreateServer(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create server")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create server",
		})
		return
	}

	c.JSON(http.StatusCreated, server)
}

// GetServer handles GET /api/v1/servers/:id
func (h *RegistryHandler) GetServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server ID is required",
		})
		return
	}

	// Check access if access service is configured
	if h.accessService != nil {
		roles := middleware.GetUserRoles(c)
		canAccess, err := h.accessService.CanAccessServer(c.Request.Context(), roles, id, domain.AccessLevelView)
		if err != nil {
			h.logger.Error().Err(err).Str("server_id", id).Any("roles", roles).Msg("Failed to check server access")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check server access",
			})
			return
		}
		if !canAccess {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied to this server",
			})
			return
		}
	}

	server, err := h.service.GetServer(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Server not found",
			})
			return
		}

		h.logger.Error().Err(err).Str("server_id", id).Msg("Failed to get server")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get server",
		})
		return
	}

	c.JSON(http.StatusOK, server)
}

// UpdateServer handles PUT /api/v1/servers/:id
func (h *RegistryHandler) UpdateServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server ID is required",
		})
		return
	}

	var req domain.ServerUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	server, err := h.service.UpdateServer(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Server not found",
			})
			return
		}

		h.logger.Error().Err(err).Str("server_id", id).Msg("Failed to update server")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update server",
		})
		return
	}

	c.JSON(http.StatusOK, server)
}

// DeleteServer handles DELETE /api/v1/servers/:id
func (h *RegistryHandler) DeleteServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server ID is required",
		})
		return
	}

	err := h.service.DeleteServer(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Server not found",
			})
			return
		}

		h.logger.Error().Err(err).Str("server_id", id).Msg("Failed to delete server")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete server",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Server deleted successfully",
	})
}

// ToggleServer handles PATCH /api/v1/servers/:id/toggle
func (h *RegistryHandler) ToggleServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server ID is required",
		})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	server, err := h.service.ToggleServer(c.Request.Context(), id, req.IsActive)
	if err != nil {
		if errors.Is(err, domain.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Server not found",
			})
			return
		}

		h.logger.Error().Err(err).Str("server_id", id).Msg("Failed to toggle server")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to toggle server",
		})
		return
	}

	c.JSON(http.StatusOK, server)
}

// GetHealthStatus handles GET /api/v1/servers/:id/health
func (h *RegistryHandler) GetHealthStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server ID is required",
		})
		return
	}

	health, err := h.service.GetHealthStatus(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("server_id", id).Msg("Failed to get health status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get health status",
		})
		return
	}

	c.JSON(http.StatusOK, health)
}

// CheckHealth handles POST /api/v1/servers/:id/health
func (h *RegistryHandler) CheckHealth(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server ID is required",
		})
		return
	}

	// Trigger health check
	err := h.service.CheckHealth(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Server not found",
			})
			return
		}

		h.logger.Error().Err(err).Str("server_id", id).Msg("Failed to check health")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check health",
		})
		return
	}

	// Return latest health status
	health, err := h.service.GetHealthStatus(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("server_id", id).Msg("Failed to get health status after check")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Health check completed but failed to retrieve status",
		})
		return
	}

	c.JSON(http.StatusOK, health)
}

// TestConnection handles POST /api/v1/servers/test-connection
// Tests connectivity to an MCP server without saving it
func (h *RegistryHandler) TestConnection(c *gin.Context) {
	var req registry.TestConnectionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	result, err := h.service.TestConnection(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Str("url", req.URL).Msg("Connection test failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Connection test failed",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CallTool handles POST /api/v1/servers/call-tool
// Calls a tool on an MCP server (for inspection/testing)
func (h *RegistryHandler) CallTool(c *gin.Context) {
	var req registry.CallToolRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	if req.ToolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tool name is required",
		})
		return
	}

	result, err := h.service.CallTool(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Str("url", req.URL).Str("tool", req.ToolName).Msg("Tool call failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Tool call failed",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
