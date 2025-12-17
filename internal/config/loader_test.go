package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithValidConfig(t *testing.T) {
	// Test loading the default config.yaml
	cfg, err := Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify defaults are set
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "development", cfg.Server.Environment)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
}

func TestLoad_WithConfigPath(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
server:
  port: 9999
  environment: development
database:
  host: testdb
  port: 5433
  user: testuser
  password: testpass
  database: testdb
auth:
  jwt_secret: test-secret-key
logging:
  level: debug
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config from custom path
	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, 9999, cfg.Server.Port)
	assert.Equal(t, "development", cfg.Server.Environment)
	assert.Equal(t, "testdb", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
}

func TestLoad_WithMissingConfig_ReturnsError(t *testing.T) {
	// Load with non-existent path should return error
	cfg, err := Load("/non/existent/path/config.yaml")
	assert.Error(t, err, "Should return error for missing config file")
	assert.Nil(t, cfg)
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables (use correct format: SECTION_FIELD)
	os.Setenv("SERVER_PORT", "7777")
	os.Setenv("DATABASE_HOST", "envdb")
	os.Setenv("LOGGING_LEVEL", "error")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("LOGGING_LEVEL")
	}()

	cfg, err := Load("")
	require.NoError(t, err)

	// Environment variables should override config file
	assert.Equal(t, 7777, cfg.Server.Port)
	assert.Equal(t, "envdb", cfg.Database.Host)
	assert.Equal(t, "error", cfg.Logging.Level)
}

func TestLoad_Validation(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			envVars: map[string]string{
				"SERVER_PORT":        "8080",
				"SERVER_ENVIRONMENT": "production",
				"DATABASE_HOST":      "localhost",
				"DATABASE_USER":      "user",
				"DATABASE_DATABASE":  "db",
				"AUTH_JWT_SECRET":    "production-secret-key",
				"REDIS_HOST":         "localhost",
			},
			expectError: false,
		},
		{
			name: "invalid port - too low",
			envVars: map[string]string{
				"SERVER_PORT": "0",
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "invalid port - too high",
			envVars: map[string]string{
				"SERVER_PORT": "70000",
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "invalid environment",
			envVars: map[string]string{
				"SERVER_ENVIRONMENT": "invalid",
			},
			expectError: true,
			errorMsg:    "invalid environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			cfg, err := Load("")

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestLoad_WithInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	invalidYAML := `
server:
  port: "not a number"
  invalid yaml here
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err, "Should fail on invalid YAML")
}
