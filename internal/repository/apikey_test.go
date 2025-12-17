package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAPIKey(t *testing.T) {
	plainKey, keyHash, err := GenerateAPIKey()

	require.NoError(t, err)
	assert.NotEmpty(t, plainKey)
	assert.NotEmpty(t, keyHash)

	// Check prefix
	assert.True(t, strings.HasPrefix(plainKey, "mcpgw_"), "API key should have mcpgw_ prefix")

	// Check key length: "mcpgw_" (6) + 64 hex chars (32 bytes) = 70
	assert.Len(t, plainKey, 70, "API key should be 70 characters")

	// Check hash length: SHA256 = 64 hex chars
	assert.Len(t, keyHash, 64, "Key hash should be 64 characters")
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	keys := make(map[string]bool)

	// Generate 100 keys and check they're all unique
	for i := 0; i < 100; i++ {
		plainKey, keyHash, err := GenerateAPIKey()
		require.NoError(t, err)

		assert.False(t, keys[plainKey], "Generated duplicate plain key")
		assert.False(t, keys[keyHash], "Generated duplicate key hash")

		keys[plainKey] = true
		keys[keyHash] = true
	}
}

func TestHashAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		plainKey string
	}{
		{
			name:     "standard API key",
			plainKey: "mcpgw_abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "short key",
			plainKey: "mcpgw_short",
		},
		{
			name:     "empty key",
			plainKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashAPIKey(tt.plainKey)

			// SHA256 hash should always be 64 hex characters
			assert.Len(t, hash, 64)

			// Same key should produce same hash
			hash2 := HashAPIKey(tt.plainKey)
			assert.Equal(t, hash, hash2, "Same key should produce same hash")
		})
	}
}

func TestHashAPIKey_Consistency(t *testing.T) {
	plainKey := "mcpgw_test1234567890abcdef"

	hash1 := HashAPIKey(plainKey)
	hash2 := HashAPIKey(plainKey)
	hash3 := HashAPIKey(plainKey)

	assert.Equal(t, hash1, hash2)
	assert.Equal(t, hash2, hash3)
}

func TestHashAPIKey_DifferentKeysProduceDifferentHashes(t *testing.T) {
	key1 := "mcpgw_key1"
	key2 := "mcpgw_key2"

	hash1 := HashAPIKey(key1)
	hash2 := HashAPIKey(key2)

	assert.NotEqual(t, hash1, hash2, "Different keys should produce different hashes")
}

func TestGenerateAPIKey_ConsistentHashing(t *testing.T) {
	// Generate a key and verify its hash matches when rehashed
	plainKey, keyHash, err := GenerateAPIKey()
	require.NoError(t, err)

	// Hash the plain key again
	rehashedKey := HashAPIKey(plainKey)

	assert.Equal(t, keyHash, rehashedKey, "Generated hash should match rehashed key")
}

func TestAPIKey_KeyPrefix(t *testing.T) {
	plainKey, _, err := GenerateAPIKey()
	require.NoError(t, err)

	// Key prefix should be first 14 chars: "mcpgw_" + 8 hex chars
	prefix := plainKey[:14]
	assert.True(t, strings.HasPrefix(prefix, "mcpgw_"))
	assert.Len(t, prefix, 14)
}

func TestAPIKeyStruct(t *testing.T) {
	apiKey := &APIKey{
		ID:        "key-123",
		UserID:    "user-456",
		Name:      "My API Key",
		KeyHash:   "abc123hash",
		KeyPrefix: "mcpgw_abc1",
	}

	assert.Equal(t, "key-123", apiKey.ID)
	assert.Equal(t, "user-456", apiKey.UserID)
	assert.Equal(t, "My API Key", apiKey.Name)
	assert.Equal(t, "abc123hash", apiKey.KeyHash)
	assert.Equal(t, "mcpgw_abc1", apiKey.KeyPrefix)
}

// BenchmarkGenerateAPIKey measures API key generation performance
func BenchmarkGenerateAPIKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := GenerateAPIKey()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHashAPIKey measures API key hashing performance
func BenchmarkHashAPIKey(b *testing.B) {
	plainKey := "mcpgw_abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashAPIKey(plainKey)
	}
}
