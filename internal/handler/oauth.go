package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/waffles/mcp-gateway/internal/handler/middleware"
	"github.com/waffles/mcp-gateway/internal/repository"
	"github.com/waffles/mcp-gateway/internal/service/oauth"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

const (
	// Session key for OAuth state
	oauthStateKey = "oauth_state"
)

// OAuthHandler handles OAuth/SSO authentication requests
type OAuthHandler struct {
	oauthService OAuthServiceInterface
	userRepo     OAuthUserRepoInterface
	logger       logger.Logger
	frontendURL  string // URL to redirect to after successful login
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(oauthService *oauth.Service, userRepo *repository.UserRepository, log logger.Logger, frontendURL string) *OAuthHandler {
	var svc OAuthServiceInterface
	var repo OAuthUserRepoInterface
	if oauthService != nil {
		svc = oauthService
	}
	if userRepo != nil {
		repo = userRepo
	}

	return &OAuthHandler{
		oauthService: svc,
		userRepo:     repo,
		logger:       log,
		frontendURL:  frontendURL,
	}
}

// NewOAuthHandlerWithInterface creates a new OAuth handler with interfaces for testing.
func NewOAuthHandlerWithInterface(oauthService OAuthServiceInterface, userRepo OAuthUserRepoInterface, log logger.Logger, frontendURL string) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		userRepo:     userRepo,
		logger:       log,
		frontendURL:  frontendURL,
	}
}

// SSOStatusResponse represents the response for SSO status
type SSOStatusResponse struct {
	Enabled bool `json:"enabled"`
}

// GetSSOStatus handles GET /api/v1/auth/sso/status
// Returns whether SSO is enabled
func (h *OAuthHandler) GetSSOStatus(c *gin.Context) {
	c.JSON(http.StatusOK, SSOStatusResponse{
		Enabled: h.oauthService.IsEnabled(),
	})
}

// Authorize handles GET /api/v1/auth/sso
// Redirects the user to the SSO provider's authorization page
func (h *OAuthHandler) Authorize(c *gin.Context) {
	// Check if SSO is enabled
	if !h.oauthService.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "sso_not_enabled",
			"message": "SSO is not enabled or not configured",
		})
		return
	}

	// Generate state for CSRF protection
	state, err := h.oauthService.GenerateState()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate OAuth state")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to initiate SSO flow",
		})
		return
	}

	// Store state in session
	session := sessions.Default(c)
	session.Set(oauthStateKey, state)
	if err := session.Save(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to save OAuth state to session")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "session_error",
			"message": "Failed to initiate SSO flow",
		})
		return
	}

	// Get authorization URL
	authURL, err := h.oauthService.GetAuthURL(state)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get OAuth authorization URL")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to initiate SSO flow",
		})
		return
	}

	h.logger.Info().Msg("Redirecting to SSO provider")

	// Redirect to OAuth provider
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// Callback handles GET /api/v1/auth/sso/callback
// Processes the OAuth callback and creates a session for the user
func (h *OAuthHandler) Callback(c *gin.Context) {
	// Get state and code from query params
	state := c.Query("state")
	code := c.Query("code")
	errorParam := c.Query("error")
	errorDesc := c.Query("error_description")

	// Handle OAuth errors
	if errorParam != "" {
		h.logger.Warn().
			Str("error", errorParam).
			Str("description", errorDesc).
			Msg("SSO provider returned error")
		h.redirectWithError(c, fmt.Sprintf("SSO error: %s", errorDesc))
		return
	}

	// Verify code is present
	if code == "" {
		h.logger.Warn().Msg("OAuth callback missing code")
		h.redirectWithError(c, "Missing authorization code")
		return
	}

	// Verify state matches
	session := sessions.Default(c)
	storedState := session.Get(oauthStateKey)

	if storedState == nil || storedState != state {
		h.logger.Warn().
			Str("expected_state", fmt.Sprintf("%v", storedState)).
			Str("received_state", state).
			Msg("OAuth state mismatch")
		h.redirectWithError(c, "Invalid OAuth state - please try again")
		return
	}

	// Clear OAuth state from session
	session.Delete(oauthStateKey)

	// Exchange code for user info
	userInfo, err := h.oauthService.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to exchange OAuth code")
		h.redirectWithError(c, "Failed to authenticate with SSO provider")
		return
	}

	// Validate email is present
	if userInfo.Email == "" {
		h.logger.Warn().
			Str("external_id", userInfo.ID).
			Msg("SSO user has no email")
		h.redirectWithError(c, "No email associated with your account")
		return
	}

	// Find or create user
	user, isNew, err := h.userRepo.FindOrCreateOAuthUser(
		c.Request.Context(),
		userInfo.Provider,
		userInfo.ID,
		userInfo.Email,
		userInfo.Name,
	)
	if err != nil {
		h.logger.Error().Err(err).
			Str("email", userInfo.Email).
			Msg("Failed to find or create SSO user")
		h.redirectWithError(c, "Failed to create user account")
		return
	}

	// Check if user is active
	if !user.IsActive {
		h.logger.Warn().
			Str("user_id", user.ID).
			Str("email", user.Email).
			Msg("SSO login attempt for inactive user")
		h.redirectWithError(c, "Your account is inactive")
		return
	}

	// If new user, assign default role
	if isNew && h.oauthService.AutoCreateUsers() {
		defaultRole := h.oauthService.GetDefaultRole()
		if err := h.userRepo.AssignRole(c.Request.Context(), user.ID, defaultRole); err != nil {
			h.logger.Error().Err(err).
				Str("user_id", user.ID).
				Str("role", defaultRole).
				Msg("Failed to assign default role to new SSO user")
			// Don't fail the login, just log the error
		} else {
			h.logger.Info().
				Str("user_id", user.ID).
				Str("role", defaultRole).
				Msg("Assigned default role to new SSO user")
		}
	}

	// Get user roles
	roles, err := h.userRepo.GetUserRoles(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to get user roles")
		roles = []string{}
	}

	// Create session
	session.Set(middleware.ContextKeyUserID, user.ID)
	session.Set(middleware.ContextKeyUserEmail, user.Email)
	session.Set(middleware.ContextKeyUserRoles, roles)

	if err := session.Save(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to save session")
		h.redirectWithError(c, "Failed to create session")
		return
	}

	h.logger.Info().
		Str("user_id", user.ID).
		Str("email", user.Email).
		Bool("is_new_user", isNew).
		Msg("SSO login successful")

	// Determine redirect based on role
	hasAdmin := false
	for _, role := range roles {
		if role == "admin" {
			hasAdmin = true
			break
		}
	}

	// Redirect to appropriate frontend page
	var redirectURL string
	if hasAdmin {
		redirectURL = h.frontendURL + "/admin"
	} else {
		redirectURL = h.frontendURL + "/dashboard"
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// redirectWithError redirects to the login page with an error message
func (h *OAuthHandler) redirectWithError(c *gin.Context, message string) {
	redirectURL := h.frontendURL + "/login?error=" + message
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
