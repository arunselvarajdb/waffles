package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/waffles/waffles/internal/config"
	"github.com/waffles/waffles/pkg/logger"
)

// LDAPProvider implements Provider interface for LDAP/Active Directory authentication
type LDAPProvider struct {
	config config.LDAPConfig
	logger logger.Logger
}

// NewLDAPProvider creates a new LDAP authentication provider
func NewLDAPProvider(cfg config.LDAPConfig, log logger.Logger) *LDAPProvider {
	// Set default attribute mappings if not specified
	if cfg.UserAttributes.Username == "" {
		cfg.UserAttributes.Username = "sAMAccountName"
	}
	if cfg.UserAttributes.Email == "" {
		cfg.UserAttributes.Email = "mail"
	}
	if cfg.UserAttributes.DisplayName == "" {
		cfg.UserAttributes.DisplayName = "displayName"
	}
	if cfg.UserAttributes.MemberOf == "" {
		cfg.UserAttributes.MemberOf = "memberOf"
	}
	if cfg.DefaultRole == "" {
		cfg.DefaultRole = "viewer"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &LDAPProvider{
		config: cfg,
		logger: log,
	}
}

// Name returns the provider identifier
func (p *LDAPProvider) Name() string {
	return "ldap"
}

// IsEnabled returns whether LDAP authentication is configured and active
func (p *LDAPProvider) IsEnabled() bool {
	return p.config.Enabled && p.config.URL != ""
}

// Authenticate validates credentials against LDAP/AD
func (p *LDAPProvider) Authenticate(ctx context.Context, username, password string) (*UserInfo, error) {
	if !p.IsEnabled() {
		return nil, ErrProviderUnavailable
	}

	// Prevent empty password authentication (LDAP allows this in some configs)
	if password == "" {
		return nil, ErrInvalidCredentials
	}

	// Connect to LDAP server
	conn, err := p.connect()
	if err != nil {
		p.logger.Error().Err(err).Str("url", p.config.URL).Msg("Failed to connect to LDAP server")
		return nil, ErrProviderUnavailable
	}
	defer conn.Close()

	// Bind with service account to search for user
	if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
		p.logger.Error().Err(err).Str("bind_dn", p.config.BindDN).Msg("Failed to bind to LDAP server")
		return nil, ErrProviderUnavailable
	}

	// Search for the user
	userEntry, err := p.searchUser(conn, username)
	if err != nil {
		p.logger.Debug().Err(err).Str("username", username).Msg("User not found in LDAP")
		return nil, ErrInvalidCredentials
	}

	// Authenticate the user by attempting to bind with their credentials
	userDN := userEntry.DN
	if err := conn.Bind(userDN, password); err != nil {
		p.logger.Debug().Str("username", username).Msg("LDAP authentication failed")
		return nil, ErrInvalidCredentials
	}

	// Extract user info from LDAP entry
	userInfo := p.extractUserInfo(userEntry)
	userInfo.Provider = "ldap"

	p.logger.Info().
		Str("username", userInfo.Username).
		Str("email", userInfo.Email).
		Str("roles", strings.Join(userInfo.Roles, ",")).
		Msg("LDAP authentication successful")

	return userInfo, nil
}

// GetUser retrieves user information by DN (external ID)
func (p *LDAPProvider) GetUser(ctx context.Context, externalID string) (*UserInfo, error) {
	if !p.IsEnabled() {
		return nil, ErrProviderUnavailable
	}

	conn, err := p.connect()
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	defer conn.Close()

	// Bind with service account
	if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
		return nil, ErrProviderUnavailable
	}

	// Search for user by DN
	searchRequest := ldap.NewSearchRequest(
		externalID, // Search base is the user's DN
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1, // Size limit
		int(p.config.Timeout.Seconds()),
		false,
		"(objectClass=*)",
		p.getUserAttributes(),
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil || len(result.Entries) == 0 {
		return nil, ErrUserNotFound
	}

	userInfo := p.extractUserInfo(result.Entries[0])
	userInfo.Provider = "ldap"
	return userInfo, nil
}

// connect establishes a connection to the LDAP server
func (p *LDAPProvider) connect() (*ldap.Conn, error) {
	// Configure TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: p.config.TLS.InsecureSkipVerify, // #nosec G402 -- configurable for testing
	}

	// Load CA certificate if specified
	if p.config.TLS.CACertFile != "" {
		caCert, err := os.ReadFile(p.config.TLS.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Set minimum TLS version
	switch p.config.TLS.MinVersion {
	case "1.3":
		tlsConfig.MinVersion = tls.VersionTLS13
	default:
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	var conn *ldap.Conn
	var err error

	// Connect based on URL scheme
	if strings.HasPrefix(p.config.URL, "ldaps://") {
		conn, err = ldap.DialURL(p.config.URL, ldap.DialWithTLSConfig(tlsConfig))
	} else {
		conn, err = ldap.DialURL(p.config.URL)
		// Start TLS for non-ldaps connections if not disabled
		if err == nil && !p.config.TLS.InsecureSkipVerify {
			if err = conn.StartTLS(tlsConfig); err != nil {
				_ = conn.Close() // #nosec G104 -- close error is secondary to TLS failure
				return nil, fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}

	return conn, nil
}

// searchUser searches for a user by username
func (p *LDAPProvider) searchUser(conn *ldap.Conn, username string) (*ldap.Entry, error) {
	// Replace {username} placeholder in filter
	filter := strings.ReplaceAll(p.config.UserFilter, "{username}", ldap.EscapeFilter(username))

	searchRequest := ldap.NewSearchRequest(
		p.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, // Size limit - we only want one user
		int(p.config.Timeout.Seconds()),
		false,
		filter,
		p.getUserAttributes(),
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, ErrUserNotFound
	}

	return result.Entries[0], nil
}

// getUserAttributes returns the list of attributes to request from LDAP
func (p *LDAPProvider) getUserAttributes() []string {
	return []string{
		p.config.UserAttributes.Username,
		p.config.UserAttributes.Email,
		p.config.UserAttributes.DisplayName,
		p.config.UserAttributes.MemberOf,
	}
}

// extractUserInfo converts an LDAP entry to UserInfo
func (p *LDAPProvider) extractUserInfo(entry *ldap.Entry) *UserInfo {
	attrs := p.config.UserAttributes

	// Get groups from memberOf attribute
	groups := entry.GetAttributeValues(attrs.MemberOf)

	// Map groups to roles
	roles := p.mapGroupsToRoles(groups)
	if len(roles) == 0 {
		roles = []string{p.config.DefaultRole}
	}

	return &UserInfo{
		ExternalID:  entry.DN,
		Username:    entry.GetAttributeValue(attrs.Username),
		Email:       entry.GetAttributeValue(attrs.Email),
		DisplayName: entry.GetAttributeValue(attrs.DisplayName),
		Groups:      groups,
		Roles:       roles,
		Attributes: map[string]interface{}{
			"dn": entry.DN,
		},
	}
}

// mapGroupsToRoles converts LDAP groups to internal roles using configured mappings
func (p *LDAPProvider) mapGroupsToRoles(groups []string) []string {
	if len(p.config.GroupMappings) == 0 {
		return nil
	}

	roleSet := make(map[string]bool)
	for _, group := range groups {
		// Check for exact match first
		if role, ok := p.config.GroupMappings[group]; ok {
			roleSet[role] = true
			continue
		}

		// Check for case-insensitive match
		groupLower := strings.ToLower(group)
		for mappingGroup, role := range p.config.GroupMappings {
			if strings.ToLower(mappingGroup) == groupLower {
				roleSet[role] = true
				break
			}
		}
	}

	roles := make([]string, 0, len(roleSet))
	for role := range roleSet {
		roles = append(roles, role)
	}
	return roles
}
