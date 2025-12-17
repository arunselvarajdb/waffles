package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/waffles/mcp-gateway/internal/database"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db     *database.DB
	logger logger.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.DB, log logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: log,
	}
}

// Health checks if the service is alive
// @Summary Health check
// @Description Check if the service is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	// Basic health check - service is running
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "mcp-gateway",
	})
}

// Ready checks if the service is ready to accept requests
// @Summary Readiness check
// @Description Check if the service is ready (database connected, etc.)
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Success 503 {object} map[string]interface{}
// @Router /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	response := gin.H{
		"status": "ready",
		"checks": gin.H{},
	}

	allHealthy := true

	// Check database health
	if h.db != nil {
		dbHealth := h.db.Health(c.Request.Context())
		response["checks"].(gin.H)["database"] = gin.H{
			"healthy":            dbHealth.Healthy,
			"total_connections":  dbHealth.TotalConnections,
			"idle_connections":   dbHealth.IdleConnections,
			"max_connections":    dbHealth.MaxConnections,
		}

		if !dbHealth.Healthy {
			allHealthy = false
			response["checks"].(gin.H)["database"].(gin.H)["message"] = dbHealth.Message
		}
	} else {
		response["checks"].(gin.H)["database"] = gin.H{
			"healthy": false,
			"message": "database not configured",
		}
		allHealthy = false
	}

	// Set overall status
	if allHealthy {
		response["status"] = "ready"
		c.JSON(http.StatusOK, response)
	} else {
		response["status"] = "not_ready"
		c.JSON(http.StatusServiceUnavailable, response)
	}
}
