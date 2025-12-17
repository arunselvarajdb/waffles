package config

import (
	"fmt"
)

// Validate checks if the configuration is valid
func Validate(cfg *Config) error {
	// Validate server config
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Server.Environment != "development" && cfg.Server.Environment != "staging" && cfg.Server.Environment != "production" {
		return fmt.Errorf("invalid environment: %s (must be development, staging, or production)", cfg.Server.Environment)
	}

	// Validate database config
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", cfg.Database.Port)
	}

	if cfg.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if cfg.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if cfg.Database.MaxOpenConns < 1 {
		return fmt.Errorf("database max_open_conns must be at least 1")
	}

	if cfg.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database max_idle_conns cannot be negative")
	}

	if cfg.Database.MaxIdleConns > cfg.Database.MaxOpenConns {
		return fmt.Errorf("database max_idle_conns (%d) cannot exceed max_open_conns (%d)",
			cfg.Database.MaxIdleConns, cfg.Database.MaxOpenConns)
	}

	// Validate Redis config
	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}

	if cfg.Redis.Port < 1 || cfg.Redis.Port > 65535 {
		return fmt.Errorf("invalid redis port: %d", cfg.Redis.Port)
	}

	// Validate auth config
	if cfg.Auth.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}

	if cfg.Auth.JWTSecret == "change-this-in-production" && cfg.Server.Environment == "production" {
		return fmt.Errorf("jwt_secret must be changed in production")
	}

	if cfg.Auth.JWTAccessTokenExpiry <= 0 {
		return fmt.Errorf("jwt_access_token_expiry must be positive")
	}

	if cfg.Auth.JWTRefreshTokenExpiry <= 0 {
		return fmt.Errorf("jwt_refresh_token_expiry must be positive")
	}

	// Validate secrets config
	if cfg.Secrets.Provider != "env" && cfg.Secrets.Provider != "aws" {
		return fmt.Errorf("invalid secrets provider: %s (must be 'env' or 'aws')", cfg.Secrets.Provider)
	}

	if cfg.Secrets.Provider == "aws" {
		if cfg.Secrets.AWS.Region == "" {
			return fmt.Errorf("aws region is required when using aws secrets provider")
		}
		if cfg.Secrets.AWS.SecretPrefix == "" {
			return fmt.Errorf("aws secret_prefix is required when using aws secrets provider")
		}
	}

	// Validate logging config
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", cfg.Logging.Level)
	}

	validLogFormats := map[string]bool{"json": true, "console": true}
	if !validLogFormats[cfg.Logging.Format] {
		return fmt.Errorf("invalid log format: %s (must be json or console)", cfg.Logging.Format)
	}

	// Validate metrics config
	if cfg.Metrics.Enabled && (cfg.Metrics.PrometheusPort < 1 || cfg.Metrics.PrometheusPort > 65535) {
		return fmt.Errorf("invalid prometheus port: %d", cfg.Metrics.PrometheusPort)
	}

	return nil
}
