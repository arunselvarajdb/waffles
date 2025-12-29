package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/handler/middleware"
	"github.com/waffles/waffles/internal/repository"
	"github.com/waffles/waffles/pkg/logger"
)

// APIKeyHandler handles API key-related HTTP requests
type APIKeyHandler struct {
	apiKeyRepo APIKeyRepositoryInterface
	logger     logger.Logger
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(apiKeyRepo *repository.APIKeyRepository, log logger.Logger) *APIKeyHandler {
	var repo APIKeyRepositoryInterface
	if apiKeyRepo != nil {
		repo = &apiKeyRepoAdapter{repo: apiKeyRepo}
	}

	return &APIKeyHandler{
		apiKeyRepo: repo,
		logger:     log,
	}
}

// NewAPIKeyHandlerWithInterface creates a new API key handler with interface (for testing).
func NewAPIKeyHandlerWithInterface(apiKeyRepo APIKeyRepositoryInterface, log logger.Logger) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyRepo: apiKeyRepo,
		logger:     log,
	}
}

// apiKeyRepoAdapter adapts the repository.APIKeyRepository to APIKeyRepositoryInterface.
type apiKeyRepoAdapter struct {
	repo *repository.APIKeyRepository
}

// mapRepoKeyToAPIKey converts a repository API key to a handler API key.
// This helper reduces code duplication across adapter methods.
func mapRepoKeyToAPIKey(key *repository.APIKey) *APIKey {
	return &APIKey{
		ID:             key.ID,
		UserID:         key.UserID,
		Name:           key.Name,
		Description:    key.Description,
		KeyPrefix:      key.KeyPrefix,
		ExpiresAt:      key.ExpiresAt,
		LastUsedAt:     key.LastUsedAt,
		CreatedAt:      key.CreatedAt,
		Scopes:         key.Scopes,
		AllowedServers: key.AllowedServers,
		AllowedTools:   key.AllowedTools,
		Namespaces:     key.Namespaces,
		IPWhitelist:    key.IPWhitelist,
		ReadOnly:       key.ReadOnly,
	}
}

func (a *apiKeyRepoAdapter) Create(ctx context.Context, input *CreateAPIKeyInput) (*APIKey, string, error) {
	repoInput := &repository.CreateAPIKeyInput{
		UserID:         input.UserID,
		Name:           input.Name,
		Description:    input.Description,
		ExpiresAt:      input.ExpiresAt,
		Scopes:         input.Scopes,
		AllowedServers: input.AllowedServers,
		AllowedTools:   input.AllowedTools,
		Namespaces:     input.Namespaces,
		IPWhitelist:    input.IPWhitelist,
		ReadOnly:       input.ReadOnly,
	}

	key, plainKey, err := a.repo.Create(ctx, repoInput)
	if err != nil {
		return nil, "", err
	}

	return mapRepoKeyToAPIKey(key), plainKey, nil
}

func (a *apiKeyRepoAdapter) GetByID(ctx context.Context, keyID string) (*APIKey, error) {
	key, err := a.repo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	return mapRepoKeyToAPIKey(key), nil
}

func (a *apiKeyRepoAdapter) GetByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	key, err := a.repo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	return mapRepoKeyToAPIKey(key), nil
}

func (a *apiKeyRepoAdapter) ListByUser(ctx context.Context, userID string) ([]*APIKey, error) {
	keys, err := a.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*APIKey, len(keys))
	for i, key := range keys {
		result[i] = mapRepoKeyToAPIKey(key)
	}

	return result, nil
}

func (a *apiKeyRepoAdapter) Delete(ctx context.Context, keyID, userID string) error {
	return a.repo.Delete(ctx, keyID, userID)
}

func (a *apiKeyRepoAdapter) UpdateLastUsed(ctx context.Context, keyID string) error {
	return a.repo.UpdateLastUsed(ctx, keyID)
}

// CreateAPIKeyRequest represents the create API key request body
type CreateAPIKeyRequest struct {
	Name           string   `json:"name" binding:"required,min=1,max=255"`
	Description    string   `json:"description,omitempty"`
	ExpiresIn      *int     `json:"expires_in_days,omitempty"` // Optional: number of days until expiry
	Scopes         []string `json:"scopes,omitempty"`          // Permission scopes
	AllowedServers []string `json:"allowed_servers,omitempty"` // Server UUIDs (empty = all)
	AllowedTools   []string `json:"allowed_tools,omitempty"`   // Tool names (empty = all)
	Namespaces     []string `json:"namespaces,omitempty"`      // Namespace UUIDs (empty = all)
	IPWhitelist    []string `json:"ip_whitelist,omitempty"`    // CIDR ranges (empty = any)
	ReadOnly       bool     `json:"read_only,omitempty"`       // Only allow read operations
}

// CreateAPIKeyResponse represents the create API key response
// Note: The key is only returned once!
type CreateAPIKeyResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"` // Only returned on creation
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Message   string     `json:"message"`
}

// APIKeyInfo represents API key information (without the actual key)
type APIKeyInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"` // First chars for identification
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateAPIKey handles POST /api/v1/api-keys
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Not authenticated",
		})
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "Invalid request body. Name is required.",
		})
		return
	}

	// Calculate expiry time if provided
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiry := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &expiry
	}

	// Validate scopes if provided
	for _, scope := range req.Scopes {
		if !domain.IsValidScope(scope) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": "Invalid scope: " + scope,
			})
			return
		}
	}

	// Create API key input
	input := &CreateAPIKeyInput{
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		ExpiresAt:      expiresAt,
		Scopes:         req.Scopes,
		AllowedServers: req.AllowedServers,
		AllowedTools:   req.AllowedTools,
		Namespaces:     req.Namespaces,
		IPWhitelist:    req.IPWhitelist,
		ReadOnly:       req.ReadOnly,
	}

	// Create API key
	apiKey, plainKey, err := h.apiKeyRepo.Create(c.Request.Context(), input)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to create API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create API key",
		})
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("key_id", apiKey.ID).
		Str("name", apiKey.Name).
		Msg("API key created")

	c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       plainKey,
		ExpiresAt: apiKey.ExpiresAt,
		CreatedAt: apiKey.CreatedAt,
		Message:   "Save this key securely. It will not be shown again.",
	})
}

// ListAPIKeys handles GET /api/v1/api-keys
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Not authenticated",
		})
		return
	}

	keys, err := h.apiKeyRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to list API keys")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to list API keys",
		})
		return
	}

	// Convert to response format
	result := make([]APIKeyInfo, 0, len(keys))
	for _, key := range keys {
		result = append(result, APIKeyInfo{
			ID:         key.ID,
			Name:       key.Name,
			KeyPrefix:  key.KeyPrefix,
			ExpiresAt:  key.ExpiresAt,
			LastUsedAt: key.LastUsedAt,
			CreatedAt:  key.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys": result,
		"total":    len(result),
	})
}

// DeleteAPIKey handles DELETE /api/v1/api-keys/:id
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Not authenticated",
		})
		return
	}

	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "API key ID is required",
		})
		return
	}

	err := h.apiKeyRepo.Delete(c.Request.Context(), keyID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrAPIKeyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "API key not found",
			})
			return
		}
		h.logger.Error().Err(err).Str("key_id", keyID).Msg("Failed to delete API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete API key",
		})
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("key_id", keyID).
		Msg("API key deleted")

	c.JSON(http.StatusOK, gin.H{
		"message": "API key deleted successfully",
	})
}

// GetAPIKey handles GET /api/v1/api-keys/:id
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Not authenticated",
		})
		return
	}

	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "API key ID is required",
		})
		return
	}

	key, err := h.apiKeyRepo.GetByID(c.Request.Context(), keyID)
	if err != nil {
		if errors.Is(err, domain.ErrAPIKeyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "API key not found",
			})
			return
		}
		h.logger.Error().Err(err).Str("key_id", keyID).Msg("Failed to get API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get API key",
		})
		return
	}

	// Verify ownership
	if key.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "API key not found",
		})
		return
	}

	c.JSON(http.StatusOK, APIKeyInfo{
		ID:         key.ID,
		Name:       key.Name,
		KeyPrefix:  key.KeyPrefix,
		ExpiresAt:  key.ExpiresAt,
		LastUsedAt: key.LastUsedAt,
		CreatedAt:  key.CreatedAt,
	})
}
