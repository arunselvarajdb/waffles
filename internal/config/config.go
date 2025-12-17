package config

import (
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Secrets  SecretsConfig  `mapstructure:"secrets"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	Environment     string        `mapstructure:"environment"` // development, staging, production
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	// Enable/disable authentication (set to false for local development)
	Enabled bool `mapstructure:"enabled"`

	// Session-based auth (for browser clients)
	SessionSecret   string        `mapstructure:"session_secret"`
	SessionMaxAge   time.Duration `mapstructure:"session_max_age"`    // Default: 24h
	CookieSecure    bool          `mapstructure:"cookie_secure"`      // Set false for local dev
	CookieSameSite  string        `mapstructure:"cookie_same_site"`   // strict, lax, none
	CookieDomain    string        `mapstructure:"cookie_domain"`      // Optional: for cross-subdomain

	// Casbin authorization
	CasbinModelPath  string `mapstructure:"casbin_model_path"`
	CasbinPolicyPath string `mapstructure:"casbin_policy_path"`

	// Legacy JWT config (kept for API key validation, can be removed if not needed)
	JWTSecret             string        `mapstructure:"jwt_secret"`
	JWTAccessTokenExpiry  time.Duration `mapstructure:"jwt_access_token_expiry"`
	JWTRefreshTokenExpiry time.Duration `mapstructure:"jwt_refresh_token_expiry"`
}

// SecretsConfig holds secrets management configuration
type SecretsConfig struct {
	Provider string           `mapstructure:"provider"` // env or aws
	AWS      AWSSecretsConfig `mapstructure:"aws"`
}

// AWSSecretsConfig holds AWS Secrets Manager configuration
type AWSSecretsConfig struct {
	Region       string `mapstructure:"region"`
	SecretPrefix string `mapstructure:"secret_prefix"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json or console
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	PrometheusPort int  `mapstructure:"prometheus_port"`
}
