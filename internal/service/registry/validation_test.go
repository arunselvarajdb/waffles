package registry

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateServerURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorReason string
	}{
		// Valid URLs
		{
			name:        "valid public URL",
			url:         "https://api.example.com/mcp",
			expectError: false,
		},
		{
			name:        "valid HTTP URL",
			url:         "http://mcp-server.example.com:8080",
			expectError: false,
		},
		{
			name:        "valid URL with path",
			url:         "https://example.com/api/v1/mcp",
			expectError: false,
		},

		// Invalid schemes
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
			errorReason: "empty URL",
		},
		{
			name:        "FTP scheme",
			url:         "ftp://example.com/file",
			expectError: true,
			errorReason: "invalid scheme",
		},
		{
			name:        "file scheme",
			url:         "file:///etc/passwd",
			expectError: true,
			errorReason: "invalid scheme",
		},
		{
			name:        "javascript scheme",
			url:         "javascript:alert(1)",
			expectError: true,
			errorReason: "invalid scheme",
		},

		// Localhost variants
		{
			name:        "localhost",
			url:         "http://localhost:8080",
			expectError: true,
			errorReason: "localhost not allowed",
		},
		{
			name:        "localhost with subdomain",
			url:         "http://foo.localhost:8080",
			expectError: true,
			errorReason: "localhost not allowed",
		},
		{
			name:        "127.0.0.1",
			url:         "http://127.0.0.1:8080",
			expectError: true,
			errorReason: "localhost not allowed",
		},
		{
			name:        "0.0.0.0",
			url:         "http://0.0.0.0:8080",
			expectError: true,
			errorReason: "localhost not allowed",
		},

		// Loopback IPs
		{
			name:        "127.0.0.2 loopback",
			url:         "http://127.0.0.2:8080",
			expectError: true,
			errorReason: "loopback address not allowed",
		},
		{
			name:        "IPv6 loopback",
			url:         "http://[::1]:8080",
			expectError: true,
			errorReason: "localhost not allowed",
		},

		// Private IP ranges
		{
			name:        "10.x.x.x private",
			url:         "http://10.0.0.1:8080",
			expectError: true,
			errorReason: "private IP address not allowed",
		},
		{
			name:        "172.16.x.x private",
			url:         "http://172.16.0.1:8080",
			expectError: true,
			errorReason: "private IP address not allowed",
		},
		{
			name:        "192.168.x.x private",
			url:         "http://192.168.1.1:8080",
			expectError: true,
			errorReason: "private IP address not allowed",
		},

		// Link-local
		{
			name:        "169.254.x.x link-local",
			url:         "http://169.254.1.1:8080",
			expectError: true,
			errorReason: "link-local address not allowed",
		},

		// Cloud metadata
		{
			name:        "AWS/GCP metadata",
			url:         "http://169.254.169.254/latest/meta-data",
			expectError: true,
			errorReason: "cloud metadata service IP not allowed",
		},
		{
			name:        "Azure metadata",
			url:         "http://168.63.129.16/metadata",
			expectError: true,
			errorReason: "cloud metadata service IP not allowed",
		},

		// Documentation IPs
		{
			name:        "TEST-NET-1",
			url:         "http://192.0.2.1:8080",
			expectError: true,
			errorReason: "documentation IP range not allowed",
		},
		{
			name:        "TEST-NET-2",
			url:         "http://198.51.100.1:8080",
			expectError: true,
			errorReason: "documentation IP range not allowed",
		},

		// IPv4-mapped IPv6 addresses (bypass attempts)
		{
			name:        "IPv4-mapped IPv6 loopback",
			url:         "http://[::ffff:127.0.0.1]:8080",
			expectError: true,
			errorReason: "loopback address not allowed",
		},
		{
			name:        "IPv4-mapped IPv6 private",
			url:         "http://[::ffff:10.0.0.1]:8080",
			expectError: true,
			errorReason: "private IP address not allowed",
		},
		{
			name:        "IPv4-mapped IPv6 metadata",
			url:         "http://[::ffff:169.254.169.254]:8080",
			expectError: true,
			errorReason: "cloud metadata service IP not allowed",
		},

		// Credentials in URL (should be rejected)
		{
			name:        "URL with credentials",
			url:         "http://user:pass@example.com/api",
			expectError: true,
			errorReason: "credentials in URL not allowed",
		},

		// Control characters (CRLF injection)
		{
			name:        "URL with newline",
			url:         "http://example.com/path\r\nHost: evil.com",
			expectError: true,
			errorReason: "URL contains control characters",
		},
		{
			name:        "URL with tab",
			url:         "http://example.com/path\tevil",
			expectError: true,
			errorReason: "URL contains control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerURL(tt.url)

			if tt.expectError {
				require.Error(t, err)
				var ssrfErr *SSRFError
				require.ErrorAs(t, err, &ssrfErr, "expected SSRFError type")
				assert.Contains(t, ssrfErr.Reason, tt.errorReason)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSSRFError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *SSRFError
		expected string
	}{
		{
			name: "with details",
			err: &SSRFError{
				URL:     "http://localhost",
				Reason:  "localhost not allowed",
				Details: "use a proper hostname",
			},
			expected: "SSRF protection: localhost not allowed - http://localhost (use a proper hostname)",
		},
		{
			name: "without details",
			err: &SSRFError{
				URL:    "http://localhost",
				Reason: "localhost not allowed",
			},
			expected: "SSRF protection: localhost not allowed - http://localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"localhost.localdomain", true},
		{"local", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"0.0.0.0", true},
		{"foo.localhost", true},
		{"sub.localhost", true},
		{"example.com", false},
		{"localhost.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			assert.Equal(t, tt.expected, isLocalhost(tt.host))
		})
	}
}

func TestIsKubernetesServiceName(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"my-service.default.svc.cluster.local", true},
		{"my-service.svc.cluster.local", true},
		{"my-service.default.svc", true},
		{"my-service.cluster.local", true},
		{"example.com", false},
		{"my-service", false},
		{"svc.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			assert.Equal(t, tt.expected, isKubernetesServiceName(tt.host))
		})
	}
}

func TestValidateServerURLForInternalOnly(t *testing.T) {
	allowedNetworks := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		// Allowed internal IPs
		{
			name:        "10.x.x.x allowed",
			url:         "http://10.0.0.1:8080",
			expectError: false,
		},
		{
			name:        "172.16.x.x allowed",
			url:         "http://172.16.0.1:8080",
			expectError: false,
		},
		{
			name:        "192.168.x.x allowed",
			url:         "http://192.168.1.1:8080",
			expectError: false,
		},

		// K8s service names allowed
		{
			name:        "K8s service name",
			url:         "http://my-mcp-server.default.svc.cluster.local:8080",
			expectError: false,
		},

		// Public IPs not allowed
		{
			name:        "public IP not allowed",
			url:         "http://8.8.8.8:8080",
			expectError: true,
		},

		// Empty URL
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerURLForInternalOnly(tt.url, allowedNetworks)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsCloudMetadataIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"169.254.169.254", true},
		{"168.63.129.16", true},
		{"169.254.1.1", false},
		{"8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip)
			assert.Equal(t, tt.expected, isCloudMetadataIP(ip))
		})
	}
}

func TestIsDocumentationIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.0.2.1", true},
		{"192.0.2.255", true},
		{"198.51.100.1", true},
		{"203.0.113.1", true},
		{"8.8.8.8", false},
		{"192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip)
			assert.Equal(t, tt.expected, isDocumentationIP(ip))
		})
	}
}

func TestValidateServerURLWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		config      SSRFConfig
		expectError bool
		errorReason string
	}{
		// SSRF disabled - allow everything
		{
			name:        "SSRF disabled allows localhost",
			url:         "http://localhost:8080",
			config:      SSRFConfig{Enabled: false},
			expectError: false,
		},
		{
			name:        "SSRF disabled allows private IPs",
			url:         "http://10.0.0.1:8080",
			config:      SSRFConfig{Enabled: false},
			expectError: false,
		},

		// Allowed hosts
		{
			name: "allowed host bypasses checks",
			url:  "http://host.docker.internal:8080",
			config: SSRFConfig{
				Enabled:      true,
				AllowedHosts: []string{"host.docker.internal"},
			},
			expectError: false,
		},
		{
			name: "allowed host case insensitive",
			url:  "http://HOST.DOCKER.INTERNAL:8080",
			config: SSRFConfig{
				Enabled:      true,
				AllowedHosts: []string{"host.docker.internal"},
			},
			expectError: false,
		},

		// Allow localhost
		{
			name: "localhost allowed by config",
			url:  "http://localhost:8080",
			config: SSRFConfig{
				Enabled:        true,
				AllowLocalhost: true,
			},
			expectError: false,
		},
		{
			name: "loopback allowed when localhost enabled",
			url:  "http://127.0.0.1:8080",
			config: SSRFConfig{
				Enabled:        true,
				AllowLocalhost: true,
			},
			expectError: false,
		},

		// Allow private networks
		{
			name: "private IP allowed by config",
			url:  "http://192.168.1.1:8080",
			config: SSRFConfig{
				Enabled:              true,
				AllowPrivateNetworks: true,
			},
			expectError: false,
		},
		{
			name: "10.x.x.x allowed by config",
			url:  "http://10.0.0.1:8080",
			config: SSRFConfig{
				Enabled:              true,
				AllowPrivateNetworks: true,
			},
			expectError: false,
		},

		// Allowed CIDRs
		{
			name: "IP in allowed CIDR",
			url:  "http://172.20.0.5:8080",
			config: SSRFConfig{
				Enabled:      true,
				AllowedCIDRs: []string{"172.20.0.0/16"},
			},
			expectError: false,
		},
		{
			name: "IP not in allowed CIDR still blocked",
			url:  "http://172.20.0.5:8080",
			config: SSRFConfig{
				Enabled:      true,
				AllowedCIDRs: []string{"172.21.0.0/16"},
			},
			expectError: true,
			errorReason: "private IP address not allowed",
		},

		// Cloud metadata still blocked even with private networks allowed
		{
			name: "cloud metadata blocked even with private networks",
			url:  "http://169.254.169.254/latest/meta-data",
			config: SSRFConfig{
				Enabled:              true,
				AllowPrivateNetworks: true,
			},
			expectError: true,
			errorReason: "cloud metadata service IP not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerURLWithConfig(tt.url, tt.config)

			if tt.expectError {
				require.Error(t, err)
				var ssrfErr *SSRFError
				require.ErrorAs(t, err, &ssrfErr)
				assert.Contains(t, ssrfErr.Reason, tt.errorReason)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsAllowedHost(t *testing.T) {
	tests := []struct {
		host         string
		allowedHosts []string
		expected     bool
	}{
		{"host.docker.internal", []string{"host.docker.internal"}, true},
		{"HOST.DOCKER.INTERNAL", []string{"host.docker.internal"}, true},
		{"example.com", []string{"host.docker.internal"}, false},
		{"kubernetes.default.svc", []string{"host.docker.internal", "kubernetes.default.svc"}, true},
		{"example.com", nil, false},
		{"example.com", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			assert.Equal(t, tt.expected, isAllowedHost(tt.host, tt.allowedHosts))
		})
	}
}

func TestIsInAllowedCIDR(t *testing.T) {
	tests := []struct {
		ip           string
		allowedCIDRs []string
		expected     bool
	}{
		{"172.20.0.5", []string{"172.20.0.0/16"}, true},
		{"172.21.0.5", []string{"172.20.0.0/16"}, false},
		{"10.0.0.1", []string{"10.0.0.0/8"}, true},
		{"192.168.1.1", []string{"10.0.0.0/8", "192.168.0.0/16"}, true},
		{"8.8.8.8", []string{"10.0.0.0/8"}, false},
		{"10.0.0.1", nil, false},
		{"10.0.0.1", []string{}, false},
		{"10.0.0.1", []string{"invalid-cidr"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip)
			assert.Equal(t, tt.expected, isInAllowedCIDR(ip, tt.allowedCIDRs))
		})
	}
}
