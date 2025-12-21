package handler

import (
	"errors"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/internal/handler/middleware"
	"github.com/waffles/mcp-gateway/internal/repository"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	userRepo UserRepositoryInterface
	logger   logger.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userRepo *repository.UserRepository, log logger.Logger) *AuthHandler {
	var repo UserRepositoryInterface
	if userRepo != nil {
		repo = userRepo
	}

	return &AuthHandler{
		userRepo: repo,
		logger:   log,
	}
}

// NewAuthHandlerWithInterface creates a new auth handler with interface (for testing).
func NewAuthHandlerWithInterface(userRepo UserRepositoryInterface, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		logger:   log,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	User UserInfo `json:"user"`
}

// UserInfo represents user information returned to the client
type UserInfo struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	IsActive bool     `json:"is_active"`
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "Invalid request body",
		})
		return
	}

	// Find user by email
	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			h.logger.Warn().Str("email", req.Email).Msg("Login attempt for non-existent user")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_credentials",
				"message": "Invalid email or password",
			})
			return
		}
		h.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "An error occurred during login",
		})
		return
	}

	// Check if user is active
	if !user.IsActive {
		h.logger.Warn().Str("email", req.Email).Msg("Login attempt for inactive user")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "user_inactive",
			"message": "Your account is inactive. Please contact an administrator.",
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.logger.Warn().Str("email", req.Email).Msg("Login attempt with incorrect password")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_credentials",
			"message": "Invalid email or password",
		})
		return
	}

	// Get user roles
	roles, err := h.userRepo.GetUserRoles(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to get user roles")
		roles = []string{} // Default to no roles
	}

	// Create session
	session := sessions.Default(c)
	session.Set(middleware.ContextKeyUserID, user.ID)
	session.Set(middleware.ContextKeyUserEmail, user.Email)
	session.Set(middleware.ContextKeyUserRoles, roles)

	if err := session.Save(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to save session")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "session_error",
			"message": "Failed to create session",
		})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID).
		Str("email", user.Email).
		Msg("User logged in successfully")

	c.JSON(http.StatusOK, LoginResponse{
		User: UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Name:     user.Name,
			Roles:    roles,
			IsActive: user.IsActive,
		},
	})
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)

	// Log the logout
	if userID := session.Get(middleware.ContextKeyUserID); userID != nil {
		h.logger.Info().
			Str("user_id", userID.(string)).
			Msg("User logged out")
	}

	// Clear session
	session.Clear()
	// Must include Path to match the original cookie settings, otherwise cookie won't be deleted
	session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	if err := session.Save(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to clear session")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// GetCurrentUser handles GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Not authenticated",
		})
		return
	}

	// Get user from database to ensure fresh data
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "user_not_found",
				"message": "User not found",
			})
			return
		}
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get user information",
		})
		return
	}

	// Get roles from context (already fetched during auth)
	roles := middleware.GetUserRoles(c)

	c.JSON(http.StatusOK, UserInfo{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Roles:    roles,
		IsActive: user.IsActive,
	})
}

// ChangePasswordRequest represents the change password request body
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword handles PUT /api/v1/auth/password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Not authenticated",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "Invalid request body. Password must be at least 8 characters.",
		})
		return
	}

	// Get current user
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to process request",
		})
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_password",
			"message": "Current password is incorrect",
		})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update password",
		})
		return
	}

	// Update password
	if err := h.userRepo.UpdatePassword(c.Request.Context(), userID, string(hashedPassword)); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to update password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update password",
		})
		return
	}

	h.logger.Info().Str("user_id", userID).Msg("Password changed successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}
