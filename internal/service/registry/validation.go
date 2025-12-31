package registry

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SSRFError represents an SSRF validation error
type SSRFError struct {
	URL     string
	Reason  string
	Details string
}

func (e *SSRFError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("SSRF protection: %s - %s (%s)", e.Reason, e.URL, e.Details)
	}
	return fmt.Sprintf("SSRF protection: %s - %s", e.Reason, e.URL)
}

// ValidateServerURL validates that a server URL is safe and not targeting internal resources
func ValidateServerURL(serverURL string) error {
	if serverURL == "" {
		return &SSRFError{URL: serverURL, Reason: "empty URL"}
	}

	// Check for control characters (CRLF injection prevention)
	if strings.ContainsAny(serverURL, "\r\n\t") {
		return &SSRFError{URL: serverURL, Reason: "URL contains control characters"}
	}

	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return &SSRFError{URL: serverURL, Reason: "invalid URL format", Details: err.Error()}
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "invalid scheme",
			Details: fmt.Sprintf("only http and https are allowed, got: %s", parsedURL.Scheme),
		}
	}

	// Reject URLs with credentials (user:password@host)
	if parsedURL.User != nil {
		return &SSRFError{
			URL:    serverURL,
			Reason: "credentials in URL not allowed",
		}
	}

	// Extract host (without port)
	host := parsedURL.Hostname()
	if host == "" {
		return &SSRFError{URL: serverURL, Reason: "missing host"}
	}

	// Check for localhost variations
	if isLocalhost(host) {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "localhost not allowed",
			Details: "use a proper hostname or IP address",
		}
	}

	// Check if host is an IP address
	ip := net.ParseIP(host)
	if ip != nil {
		if err := validateIP(ip, serverURL); err != nil {
			return err
		}
	} else {
		// Host is a domain name - resolve and check
		if err := validateHostname(host, serverURL); err != nil {
			return err
		}
	}

	return nil
}

// isLocalhost checks if the host is a localhost variant
func isLocalhost(host string) bool {
	host = strings.ToLower(host)
	localhostVariants := []string{
		"localhost",
		"localhost.localdomain",
		"local",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
	}

	for _, variant := range localhostVariants {
		if host == variant {
			return true
		}
	}

	// Check for localhost subdomains
	if strings.HasSuffix(host, ".localhost") {
		return true
	}

	return false
}

// validateIP checks if an IP address is in a private/reserved range
func validateIP(ip net.IP, serverURL string) error {
	// Check cloud metadata service IPs FIRST (before link-local check)
	// AWS/GCP metadata (169.254.169.254) is in the link-local range but needs special handling
	if isCloudMetadataIP(ip) {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "cloud metadata service IP not allowed",
			Details: fmt.Sprintf("%s - this IP is used for cloud instance metadata", ip.String()),
		}
	}

	// Check loopback
	if ip.IsLoopback() {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "loopback address not allowed",
			Details: ip.String(),
		}
	}

	// Check private networks
	if ip.IsPrivate() {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "private IP address not allowed",
			Details: fmt.Sprintf("%s is in a private range (10.x.x.x, 172.16-31.x.x, 192.168.x.x)", ip.String()),
		}
	}

	// Check link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "link-local address not allowed",
			Details: ip.String(),
		}
	}

	// Check unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "unspecified address not allowed",
			Details: ip.String(),
		}
	}

	// Check multicast
	if ip.IsMulticast() {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "multicast address not allowed",
			Details: ip.String(),
		}
	}

	// Check documentation/example ranges
	if isDocumentationIP(ip) {
		return &SSRFError{
			URL:     serverURL,
			Reason:  "documentation IP range not allowed",
			Details: ip.String(),
		}
	}

	return nil
}

// validateHostname resolves a hostname and validates all resulting IPs
func validateHostname(host, serverURL string) error {
	// DNS rebinding protection: resolve the hostname
	ips, err := net.LookupIP(host)
	if err != nil {
		// Allow hostnames that don't resolve - they might be internal K8s services
		// The actual connection will fail if the hostname is invalid
		return nil //nolint:nilerr // intentionally ignoring DNS lookup errors
	}

	// Validate all resolved IPs
	for _, ip := range ips {
		if err := validateIP(ip, serverURL); err != nil {
			return &SSRFError{
				URL:     serverURL,
				Reason:  "hostname resolves to blocked IP",
				Details: fmt.Sprintf("%s resolves to %s", host, ip.String()),
			}
		}
	}

	return nil
}

// isDocumentationIP checks if IP is in documentation/example ranges
func isDocumentationIP(ip net.IP) bool {
	// TEST-NET-1: 192.0.2.0/24
	// TEST-NET-2: 198.51.100.0/24
	// TEST-NET-3: 203.0.113.0/24
	// 2001:db8::/32 (IPv6 documentation)

	documentationRanges := []string{
		"192.0.2.0/24",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"2001:db8::/32",
	}

	for _, cidr := range documentationRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// isCloudMetadataIP checks if IP is a cloud provider's metadata service
func isCloudMetadataIP(ip net.IP) bool {
	// AWS/GCP/Azure metadata service: 169.254.169.254
	// AWS IMDSv2 alternative: fd00:ec2::254
	// Azure also uses: 168.63.129.16

	metadataIPs := []string{
		"169.254.169.254",
		"168.63.129.16",
		"fd00:ec2::254",
	}

	ipStr := ip.String()
	for _, metaIP := range metadataIPs {
		if ipStr == metaIP {
			return true
		}
	}

	return false
}

// ValidateServerURLForInternalOnly validates that a URL points to an internal network
// This is the inverse of ValidateServerURL - used when we ONLY want internal URLs
func ValidateServerURLForInternalOnly(serverURL string, allowedNetworks []string) error {
	if serverURL == "" {
		return &SSRFError{URL: serverURL, Reason: "empty URL"}
	}

	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return &SSRFError{URL: serverURL, Reason: "invalid URL format", Details: err.Error()}
	}

	host := parsedURL.Hostname()
	if host == "" {
		return &SSRFError{URL: serverURL, Reason: "missing host"}
	}

	// Allow Kubernetes service discovery names
	if isKubernetesServiceName(host) {
		return nil
	}

	// Check against allowed networks
	ip := net.ParseIP(host)
	if ip != nil {
		for _, cidr := range allowedNetworks {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if network.Contains(ip) {
				return nil
			}
		}
		return &SSRFError{
			URL:     serverURL,
			Reason:  "IP not in allowed networks",
			Details: fmt.Sprintf("allowed: %v", allowedNetworks),
		}
	}

	// For non-IP hosts, check if they resolve to allowed networks
	ips, err := net.LookupIP(host)
	if err != nil {
		// Can't resolve - might be a K8s service name, allow it
		return nil //nolint:nilerr // intentionally ignoring DNS lookup errors
	}

	for _, resolvedIP := range ips {
		allowed := false
		for _, cidr := range allowedNetworks {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if network.Contains(resolvedIP) {
				allowed = true
				break
			}
		}
		if !allowed {
			return &SSRFError{
				URL:     serverURL,
				Reason:  "resolved IP not in allowed networks",
				Details: fmt.Sprintf("%s resolved to %s", host, resolvedIP.String()),
			}
		}
	}

	return nil
}

// isKubernetesServiceName checks if the hostname looks like a K8s service name
func isKubernetesServiceName(host string) bool {
	// K8s service names typically end with:
	// - .svc.cluster.local
	// - .svc
	// - Or are simple service names within the same namespace

	k8sSuffixes := []string{
		".svc.cluster.local",
		".svc",
		".cluster.local",
	}

	host = strings.ToLower(host)
	for _, suffix := range k8sSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}

	return false
}
