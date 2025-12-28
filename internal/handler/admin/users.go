package admin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/service/user"
	"github.com/waffles/waffles/pkg/logger"
)

// UsersHandler handles admin user management endpoints
type UsersHandler struct {
	service *user.Service
	logger  logger.Logger
}

// NewUsersHandler creates a new admin users handler
func NewUsersHandler(service *user.Service, log logger.Logger) *UsersHandler {
	return &UsersHandler{
		service: service,
		logger:  log.With().Str("handler", "admin-users").Logger(),
	}
}

// ListUsers returns a paginated list of users
// GET /api/v1/admin/users
func (h *UsersHandler) ListUsers(c *gin.Context) {
	var req user.ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	resp, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetUser returns a single user by ID
// GET /api/v1/admin/users/:id
func (h *UsersHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	userWithRoles, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		h.logger.Error().Err(err).Str("user_id", id).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	c.JSON(http.StatusOK, userWithRoles)
}

// CreateUser creates a new user
// POST /api/v1/admin/users
func (h *UsersHandler) CreateUser(c *gin.Context) {
	var req user.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	resp, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
			return
		}
		h.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// UpdateUser updates a user's information
// PUT /api/v1/admin/users/:id
func (h *UsersHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	var req user.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	userWithRoles, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		h.logger.Error().Err(err).Str("user_id", id).Msg("Failed to update user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, userWithRoles)
}

// DeleteUser deactivates a user (soft delete)
// DELETE /api/v1/admin/users/:id
func (h *UsersHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	if err := h.service.Deactivate(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		h.logger.Error().Err(err).Str("user_id", id).Msg("Failed to deactivate user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deactivated successfully"})
}

// UpdateUserRoles updates a user's roles
// PUT /api/v1/admin/users/:id/roles
func (h *UsersHandler) UpdateUserRoles(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	var req user.RoleAssignment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	userWithRoles, err := h.service.UpdateRoles(c.Request.Context(), id, req.Roles)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		h.logger.Error().Err(err).Str("user_id", id).Msg("Failed to update user roles")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user roles"})
		return
	}

	c.JSON(http.StatusOK, userWithRoles)
}

// ResetPassword generates a new temp password for a user
// POST /api/v1/admin/users/:id/reset-password
func (h *UsersHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	tempPassword, err := h.service.ResetPassword(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		h.logger.Error().Err(err).Str("user_id", id).Msg("Failed to reset password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Password reset successfully",
		"temp_password": tempPassword,
	})
}
