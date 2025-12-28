package admin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/waffles/waffles/internal/service/role"
	"github.com/waffles/waffles/pkg/logger"
)

// RolesHandler handles admin role management endpoints
type RolesHandler struct {
	service *role.Service
	logger  logger.Logger
}

// NewRolesHandler creates a new admin roles handler
func NewRolesHandler(service *role.Service, log logger.Logger) *RolesHandler {
	return &RolesHandler{
		service: service,
		logger:  log.With().Str("handler", "admin-roles").Logger(),
	}
}

// ListRoles returns all roles with user counts
// GET /api/v1/admin/roles
func (h *RolesHandler) ListRoles(c *gin.Context) {
	roles, err := h.service.List(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list roles")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list roles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// GetRole returns a single role by ID with permissions
// GET /api/v1/admin/roles/:id
func (h *RolesHandler) GetRole(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role ID is required"})
		return
	}

	roleWithPerms, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		h.logger.Error().Err(err).Str("role_id", id).Msg("Failed to get role")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role"})
		return
	}

	c.JSON(http.StatusOK, roleWithPerms)
}

// CreateRole creates a new custom role
// POST /api/v1/admin/roles
func (h *RolesHandler) CreateRole(c *gin.Context) {
	var req role.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	roleWithPerms, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, role.ErrRoleNameExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "Role with this name already exists"})
			return
		}
		h.logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create role")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}

	c.JSON(http.StatusCreated, roleWithPerms)
}

// UpdateRole updates a role's permissions
// PUT /api/v1/admin/roles/:id
func (h *RolesHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role ID is required"})
		return
	}

	var req role.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	roleWithPerms, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		h.logger.Error().Err(err).Str("role_id", id).Msg("Failed to update role")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	c.JSON(http.StatusOK, roleWithPerms)
}

// DeleteRole deletes a custom role (built-in roles cannot be deleted)
// DELETE /api/v1/admin/roles/:id
func (h *RolesHandler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role ID is required"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		if errors.Is(err, role.ErrBuiltInRole) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete built-in roles"})
			return
		}
		h.logger.Error().Err(err).Str("role_id", id).Msg("Failed to delete role")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}

// ListPermissions returns all available permissions
// GET /api/v1/admin/permissions
func (h *RolesHandler) ListPermissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list permissions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}
