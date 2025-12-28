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
	StaticDir       string        `mapstructure:"static_dir"`  // Path to frontend static files (empty = no UI)
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

	// LDAP/AD configuration (can be enabled alongside OAuth)
	LDAP LDAPConfig `mapstructure:"ldap"`

	// Local database authentication (can be enabled alongside OAuth/LDAP)
	Local LocalAuthConfig `mapstructure:"local"`

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

// LDAPConfig holds LDAP/Active Directory authentication configuration
// Can be enabled alongside OAuth for organizations using AD
type LDAPConfig struct {
	// Enable LDAP authentication
	Enabled bool `mapstructure:"enabled"`

	// LDAP server URL (e.g., "ldaps://ad.example.com:636" or "ldap://ad.example.com:389")
	URL string `mapstructure:"url"`

	// Base DN for user searches (e.g., "dc=example,dc=com")
	BaseDN string `mapstructure:"base_dn"`

	// Bind DN for connecting to LDAP (service account)
	// e.g., "cn=svc-mcp,ou=services,dc=example,dc=com"
	BindDN string `mapstructure:"bind_dn"`

	// Bind password (use environment variable: ${LDAP_BIND_PASSWORD})
	BindPassword string `mapstructure:"bind_password"`

	// User search filter - {username} will be replaced with the login username
	// For AD: "(sAMAccountName={username})"
	// For OpenLDAP: "(uid={username})"
	UserFilter string `mapstructure:"user_filter"`

	// Group search filter - {dn} will be replaced with the user's DN
	// e.g., "(member={dn})"
	GroupFilter string `mapstructure:"group_filter"`

	// User attributes to fetch
	UserAttributes LDAPUserAttributes `mapstructure:"user_attributes"`

	// Group to role mappings
	// Maps LDAP group DNs to internal roles (admin, operator, viewer)
	GroupMappings map[string]string `mapstructure:"group_mappings"`

	// Default role for users without matching group mappings
	DefaultRole string `mapstructure:"default_role"`

	// TLS configuration
	TLS LDAPTLSConfig `mapstructure:"tls"`

	// Connection timeout
	Timeout time.Duration `mapstructure:"timeout"`
}

// LDAPUserAttributes specifies which LDAP attributes to use for user info
type LDAPUserAttributes struct {
	// Username attribute (default: "sAMAccountName" for AD, "uid" for OpenLDAP)
	Username string `mapstructure:"username"`
	// Email attribute (default: "mail")
	Email string `mapstructure:"email"`
	// Display name attribute (default: "displayName")
	DisplayName string `mapstructure:"display_name"`
	// Group membership attribute (default: "memberOf")
	MemberOf string `mapstructure:"member_of"`
}

// LDAPTLSConfig holds TLS settings for LDAP connections
type LDAPTLSConfig struct {
	// Skip certificate verification (NOT recommended for production)
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
	// Path to CA certificate file for verifying LDAP server
	CACertFile string `mapstructure:"ca_cert_file"`
	// Minimum TLS version (default: "1.2")
	MinVersion string `mapstructure:"min_version"`
}

// LocalAuthConfig holds configuration for local database authentication
// Provides password-based auth with built-in security features
type LocalAuthConfig struct {
	// Enable local database authentication (default: true for development)
	Enabled bool `mapstructure:"enabled"`

	// Account lockout settings (handled by application when using local auth)
	Lockout LockoutConfig `mapstructure:"lockout"`

	// Password policy settings
	PasswordPolicy PasswordPolicyConfig `mapstructure:"password_policy"`
}

// LockoutConfig holds account lockout settings for local authentication
type LockoutConfig struct {
	// Maximum failed login attempts before lockout (default: 5)
	MaxAttempts int `mapstructure:"max_attempts"`
	// Duration of account lockout (default: 15m)
	Duration time.Duration `mapstructure:"duration"`
	// Reset failed attempts counter after this duration of successful logins
	ResetAfter time.Duration `mapstructure:"reset_after"`
}

// PasswordPolicyConfig holds password requirements for local authentication
type PasswordPolicyConfig struct {
	// Minimum password length (default: 12)
	MinLength int `mapstructure:"min_length"`
	// Require at least one uppercase letter
	RequireUppercase bool `mapstructure:"require_uppercase"`
	// Require at least one lowercase letter
	RequireLowercase bool `mapstructure:"require_lowercase"`
	// Require at least one number
	RequireNumber bool `mapstructure:"require_number"`
	// Require at least one special character
	RequireSpecial bool `mapstructure:"require_special"`
	// Maximum password age before requiring change (0 = never expires)
	MaxAge time.Duration `mapstructure:"max_age"`
	// Number of previous passwords to prevent reuse (0 = no history)
	HistoryCount int `mapstructure:"history_count"`
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
