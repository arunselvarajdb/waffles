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
	SessionSecret  string        `mapstructure:"session_secret"`
	SessionMaxAge  time.Duration `mapstructure:"session_max_age"`  // Default: 24h
	CookieSecure   bool          `mapstructure:"cookie_secure"`    // Set false for local dev
	CookieSameSite string        `mapstructure:"cookie_same_site"` // strict, lax, none
	CookieDomain   string        `mapstructure:"cookie_domain"`    // Optional: for cross-subdomain

	// Casbin authorization
	CasbinModelPath  string `mapstructure:"casbin_model_path"`
	CasbinPolicyPath string `mapstructure:"casbin_policy_path"`

	// Resource RBAC - controls which MCP servers users can see/execute based on role
	// When enabled, users only see servers in namespaces their role has access to
	// When disabled, all authenticated users see all servers (existing behavior)
	ResourceRBACEnabled bool `mapstructure:"resource_rbac_enabled"`

	// Legacy alias for backwards compatibility - deprecated, use ResourceRBACEnabled
	ServerGroupRBACEnabled bool `mapstructure:"server_group_rbac_enabled"`

	// MCP Client Authentication - controls which auth methods are accepted for MCP clients
	MCPAuth MCPAuthConfig `mapstructure:"mcp_auth"`

	// OAuth/SSO configuration (Keycloak or other OIDC provider)
	OAuth OAuthConfig `mapstructure:"oauth"`

	// Legacy JWT config (kept for API key validation, can be removed if not needed)
	JWTSecret             string        `mapstructure:"jwt_secret"`
	JWTAccessTokenExpiry  time.Duration `mapstructure:"jwt_access_token_expiry"`
	JWTRefreshTokenExpiry time.Duration `mapstructure:"jwt_refresh_token_expiry"`
}

// MCPAuthConfig controls which authentication methods are accepted for MCP clients
// This allows fine-grained control over how MCP clients (Claude Code, etc.) authenticate
type MCPAuthConfig struct {
	// API Key authentication - tokens prefixed with mcpgw_
	// Default: true
	APIKeyEnabled bool `mapstructure:"api_key"`

	// Session authentication - use existing browser session
	// Default: true
	SessionEnabled bool `mapstructure:"session"`

	// OAuth authentication for MCP clients (DCR flow)
	// When false, /.well-known/oauth-protected-resource returns 404
	// This allows enabling UI SSO while requiring MCP clients to use API keys
	// Default: true (if auth.oauth.enabled is true)
	OAuthEnabled bool `mapstructure:"oauth"`
}

// OAuthConfig holds SSO configuration for any OIDC-compliant provider
// Recommended: Keycloak (supports DCR for MCP clients like Claude Code)
// Also supports: Zitadel, Okta, Auth0, Azure AD, Google, and others
type OAuthConfig struct {
	// Enable SSO login (can be enabled alongside local password auth)
	Enabled bool `mapstructure:"enabled"`

	// Base URL for OAuth callbacks (e.g., "https://gateway.example.com")
	// Used to construct redirect URL: {BaseURL}/api/v1/auth/sso/callback
	BaseURL string `mapstructure:"base_url"`

	// Default role assigned to new SSO users (e.g., "viewer")
	DefaultRole string `mapstructure:"default_role"`

	// Auto-create users on first SSO login
	AutoCreateUsers bool `mapstructure:"auto_create_users"`

	// OIDC Provider configuration
	// The issuer URL is used to discover authorization, token, and userinfo endpoints
	Issuer       string   `mapstructure:"issuer"`        // OIDC issuer URL (e.g., "https://auth.example.com/realms/myrealm")
	ClientID     string   `mapstructure:"client_id"`     // OAuth client ID
	ClientSecret string   `mapstructure:"client_secret"` // OAuth client secret
	Scopes       []string `mapstructure:"scopes"`        // OAuth scopes (default: openid, email, profile)

	// Optional: restrict login to specific email domains
	// e.g., ["example.com", "company.org"]
	AllowedDomains []string `mapstructure:"allowed_domains"`
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
