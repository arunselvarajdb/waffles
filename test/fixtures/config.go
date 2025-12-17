package fixtures

import (
	"time"

	"github.com/waffles/mcp-gateway/internal/config"
)

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            8080,
			Environment:     "development",
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "test_user",
			Password:        "test_password",
			Database:        "test_db",
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		Auth: config.AuthConfig{
			JWTSecret:             "test-secret-key-for-testing-only",
			JWTAccessTokenExpiry:  15 * time.Minute,
			JWTRefreshTokenExpiry: 168 * time.Hour,
		},
		Secrets: config.SecretsConfig{
			Provider: "env",
			AWS: config.AWSSecretsConfig{
				Region:       "us-east-1",
				SecretPrefix: "test/mcp-gateway",
			},
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
		Metrics: config.MetricsConfig{
			Enabled:        true,
			PrometheusPort: 9090,
		},
	}
}

// MinimalTestConfig returns a minimal test configuration
func MinimalTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:        "localhost",
			Port:        8080,
			Environment: "development",
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			Database: "test",
		},
		Auth: config.AuthConfig{
			JWTSecret: "test-secret",
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}
}
