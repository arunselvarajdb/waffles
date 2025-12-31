package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// APIKey represents an API key in the database
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	KeyHash     string     `json:"-"`          // Never expose hash
	KeyPrefix   string     `json:"key_prefix"` // First 8 chars for identification
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	// Scope restrictions
	Scopes         []string `json:"scopes"`
	AllowedServers []string `json:"allowed_servers"`
	AllowedTools   []string `json:"allowed_tools"`
	Namespaces     []string `json:"namespaces"`
	IPWhitelist    []string `json:"ip_whitelist"`
	ReadOnly       bool     `json:"read_only"`
}

// APIKeyRepository handles API key data persistence
type APIKeyRepository struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(pool *pgxpool.Pool, log logger.Logger) *APIKeyRepository {
	return &APIKeyRepository{
		pool:   pool,
		logger: log,
	}
}

// GenerateAPIKey generates a new API key with the format mcpgw_<random>
// Returns the plain text key (only shown once) and the key hash for storage
func GenerateAPIKey() (plainKey string, keyHash string, err error) {
	// Generate 32 random bytes
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create the plain text key with prefix
	plainKey = "mcpgw_" + hex.EncodeToString(randomBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(plainKey))
	keyHash = hex.EncodeToString(hash[:])

	return plainKey, keyHash, nil
}

// HashAPIKey hashes an API key for lookup
func HashAPIKey(plainKey string) string {
	hash := sha256.Sum256([]byte(plainKey))
	return hex.EncodeToString(hash[:])
}

// CreateAPIKeyInput contains the parameters for creating a new API key
type CreateAPIKeyInput struct {
	UserID         string
	Name           string
	Description    string
	ExpiresAt      *time.Time
	Scopes         []string
	AllowedServers []string
	AllowedTools   []string
	Namespaces     []string
	IPWhitelist    []string
	ReadOnly       bool
}

// generateKeyPrefix creates an obfuscated key prefix for display
// Format: mcpgw_<first4>****<last4> (e.g., "mcpgw_9e8f****90ef")
func generateKeyPrefix(plainKey string) string {
	// Key format is mcpgw_<64 hex chars>
	// We want to show: mcpgw_<first4>****<last4>
	if len(plainKey) < 16 {
		return plainKey
	}
	// Get first 10 chars (mcpgw_ + first 4 hex) and last 4 chars
	return plainKey[:10] + "****" + plainKey[len(plainKey)-4:]
}

// Create creates a new API key for a user
// Returns the APIKey record and the plain text key (only returned once!)
func (r *APIKeyRepository) Create(ctx context.Context, input *CreateAPIKeyInput) (*APIKey, string, error) {
	plainKey, keyHash, err := GenerateAPIKey()
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to generate API key")
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Generate obfuscated prefix for display (e.g., "mcpgw_abc12345...6789")
	keyPrefix := generateKeyPrefix(plainKey)

	query := `
		INSERT INTO api_keys (
			user_id, name, description, key_hash, key_prefix, expires_at,
			scopes, allowed_servers, allowed_tools, namespaces, ip_whitelist, read_only
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	// Ensure empty slices are not nil (for PostgreSQL array handling)
	scopes := input.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	allowedServers := input.AllowedServers
	if allowedServers == nil {
		allowedServers = []string{}
	}
	allowedTools := input.AllowedTools
	if allowedTools == nil {
		allowedTools = []string{}
	}
	namespaces := input.Namespaces
	if namespaces == nil {
		namespaces = []string{}
	}
	ipWhitelist := input.IPWhitelist
	if ipWhitelist == nil {
		ipWhitelist = []string{}
	}

	apiKey := &APIKey{
		UserID:         input.UserID,
		Name:           input.Name,
		Description:    input.Description,
		KeyHash:        keyHash,
		KeyPrefix:      keyPrefix,
		ExpiresAt:      input.ExpiresAt,
		Scopes:         scopes,
		AllowedServers: allowedServers,
		AllowedTools:   allowedTools,
		Namespaces:     namespaces,
		IPWhitelist:    ipWhitelist,
		ReadOnly:       input.ReadOnly,
	}

	err = r.pool.QueryRow(ctx, query,
		input.UserID, input.Name, input.Description, keyHash, keyPrefix, input.ExpiresAt,
		scopes, allowedServers, allowedTools, namespaces, ipWhitelist, input.ReadOnly,
	).Scan(
		&apiKey.ID,
		&apiKey.CreatedAt,
	)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", input.UserID).Msg("Failed to create API key")
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	r.logger.Info().
		Str("key_id", apiKey.ID).
		Str("user_id", input.UserID).
		Str("name", input.Name).
		Int("scope_count", len(scopes)).
		Msg("API key created")

	return apiKey, plainKey, nil
}

// GetByHash retrieves an API key by its hash
func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `
		SELECT id, user_id, name, COALESCE(description, ''), key_hash, COALESCE(key_prefix, 'mcpgw_****'),
			expires_at, last_used_at, created_at,
			COALESCE(scopes, '{}'), COALESCE(allowed_servers, '{}'), COALESCE(allowed_tools, '{}'),
			COALESCE(namespaces, '{}'), COALESCE(ip_whitelist, '{}'), COALESCE(read_only, false)
		FROM api_keys
		WHERE key_hash = $1
	`

	var apiKey APIKey
	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.Name,
		&apiKey.Description,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.ExpiresAt,
		&apiKey.LastUsedAt,
		&apiKey.CreatedAt,
		&apiKey.Scopes,
		&apiKey.AllowedServers,
		&apiKey.AllowedTools,
		&apiKey.Namespaces,
		&apiKey.IPWhitelist,
		&apiKey.ReadOnly,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAPIKeyNotFound
	}
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get API key by hash")
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Check if expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, domain.ErrAPIKeyExpired
	}

	return &apiKey, nil
}

// ListByUser retrieves all API keys for a user
func (r *APIKeyRepository) ListByUser(ctx context.Context, userID string) ([]*APIKey, error) {
	query := `
		SELECT id, user_id, name, COALESCE(description, ''), key_hash, COALESCE(key_prefix, 'mcpgw_****'),
			expires_at, last_used_at, created_at,
			COALESCE(scopes, '{}'), COALESCE(allowed_servers, '{}'), COALESCE(allowed_tools, '{}'),
			COALESCE(namespaces, '{}'), COALESCE(ip_whitelist, '{}'), COALESCE(read_only, false)
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to list API keys")
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Name,
			&key.Description,
			&key.KeyHash,
			&key.KeyPrefix,
			&key.ExpiresAt,
			&key.LastUsedAt,
			&key.CreatedAt,
			&key.Scopes,
			&key.AllowedServers,
			&key.AllowedTools,
			&key.Namespaces,
			&key.IPWhitelist,
			&key.ReadOnly,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan API key row")
			continue
		}
		keys = append(keys, &key)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating API key rows")
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	r.logger.Debug().Str("user_id", userID).Int("count", len(keys)).Msg("API keys listed")
	return keys, nil
}

// ListAll retrieves all API keys (for admin use)
func (r *APIKeyRepository) ListAll(ctx context.Context) ([]*APIKey, error) {
	query := `
		SELECT ak.id, ak.user_id, ak.name, COALESCE(ak.description, ''), ak.key_hash, COALESCE(ak.key_prefix, 'mcpgw_****'),
			ak.expires_at, ak.last_used_at, ak.created_at,
			COALESCE(ak.scopes, '{}'), COALESCE(ak.allowed_servers, '{}'), COALESCE(ak.allowed_tools, '{}'),
			COALESCE(ak.namespaces, '{}'), COALESCE(ak.ip_whitelist, '{}'), COALESCE(ak.read_only, false),
			COALESCE(u.email, '') as user_email
		FROM api_keys ak
		LEFT JOIN users u ON ak.user_id = u.id::text
		ORDER BY ak.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list all API keys")
		return nil, fmt.Errorf("failed to list all API keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		var userEmail string
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Name,
			&key.Description,
			&key.KeyHash,
			&key.KeyPrefix,
			&key.ExpiresAt,
			&key.LastUsedAt,
			&key.CreatedAt,
			&key.Scopes,
			&key.AllowedServers,
			&key.AllowedTools,
			&key.Namespaces,
			&key.IPWhitelist,
			&key.ReadOnly,
			&userEmail,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan API key row")
			continue
		}
		// Store user email in description field temporarily for admin display
		// This is a workaround since we don't have a separate field
		if key.Description == "" && userEmail != "" {
			key.Description = userEmail
		}
		keys = append(keys, &key)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating API key rows")
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	r.logger.Debug().Int("count", len(keys)).Msg("All API keys listed")
	return keys, nil
}

// AdminDelete deletes any API key by ID (admin only, no ownership check)
func (r *APIKeyRepository) AdminDelete(ctx context.Context, keyID string) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, keyID)
	if err != nil {
		r.logger.Error().Err(err).Str("key_id", keyID).Msg("Failed to delete API key (admin)")
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}

	r.logger.Info().Str("key_id", keyID).Msg("API key deleted by admin")
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp for an API key
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, keyID string) error {
	query := `
		UPDATE api_keys
		SET last_used_at = $1
		WHERE id = $2
	`

	_, err := r.pool.Exec(ctx, query, time.Now(), keyID)
	if err != nil {
		r.logger.Error().Err(err).Str("key_id", keyID).Msg("Failed to update last_used_at")
		return fmt.Errorf("failed to update last_used_at: %w", err)
	}

	return nil
}

// Delete deletes an API key by ID (must belong to user)
func (r *APIKeyRepository) Delete(ctx context.Context, keyID, userID string) error {
	query := `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`

	result, err := r.pool.Exec(ctx, query, keyID, userID)
	if err != nil {
		r.logger.Error().Err(err).Str("key_id", keyID).Msg("Failed to delete API key")
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}

	r.logger.Info().Str("key_id", keyID).Str("user_id", userID).Msg("API key deleted")
	return nil
}

// GetByID retrieves an API key by ID
func (r *APIKeyRepository) GetByID(ctx context.Context, keyID string) (*APIKey, error) {
	query := `
		SELECT id, user_id, name, COALESCE(description, ''), key_hash, COALESCE(key_prefix, 'mcpgw_****'),
			expires_at, last_used_at, created_at,
			COALESCE(scopes, '{}'), COALESCE(allowed_servers, '{}'), COALESCE(allowed_tools, '{}'),
			COALESCE(namespaces, '{}'), COALESCE(ip_whitelist, '{}'), COALESCE(read_only, false)
		FROM api_keys
		WHERE id = $1
	`

	var apiKey APIKey
	err := r.pool.QueryRow(ctx, query, keyID).Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.Name,
		&apiKey.Description,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.ExpiresAt,
		&apiKey.LastUsedAt,
		&apiKey.CreatedAt,
		&apiKey.Scopes,
		&apiKey.AllowedServers,
		&apiKey.AllowedTools,
		&apiKey.Namespaces,
		&apiKey.IPWhitelist,
		&apiKey.ReadOnly,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAPIKeyNotFound
	}
	if err != nil {
		r.logger.Error().Err(err).Str("key_id", keyID).Msg("Failed to get API key")
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// ToDomain converts the repository APIKey to domain.APIKey
func (k *APIKey) ToDomain() *domain.APIKey {
	return &domain.APIKey{
		ID:             k.ID,
		UserID:         k.UserID,
		Name:           k.Name,
		Description:    k.Description,
		KeyHash:        k.KeyHash,
		Prefix:         k.KeyPrefix,
		ExpiresAt:      k.ExpiresAt,
		LastUsedAt:     k.LastUsedAt,
		CreatedAt:      k.CreatedAt,
		Scopes:         k.Scopes,
		AllowedServers: k.AllowedServers,
		AllowedTools:   k.AllowedTools,
		Namespaces:     k.Namespaces,
		IPWhitelist:    k.IPWhitelist,
		ReadOnly:       k.ReadOnly,
	}
}
