package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateProviderURL checks that rawURL is well-formed, uses http(s), and
// resolves to an allowlisted host. Called at config load for each local
// provider (Ollama, ComfyUI, Kokoro) so a misconfigured OLLAMA_URL can't
// redirect an internal request to an attacker's box (SSRF).
//
// The defaultAllowed list covers the common local-dev targets. Ops can
// extend it via CLOTHO_LOCAL_PROVIDER_HOSTS (comma-separated "host:port"
// or bare hostnames).
func ValidateProviderURL(label, rawURL string, extraAllowed []string) error {
	if rawURL == "" {
		return nil // provider disabled; nothing to check
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s: invalid URL %q: %w", label, rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s: scheme must be http or https, got %q", label, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%s: missing host in %q", label, rawURL)
	}

	host := u.Hostname()
	hostLower := strings.ToLower(host)

	allowed := append([]string{
		"localhost",
		"127.0.0.1",
		"::1",
		"host.docker.internal",
	}, extraAllowed...)

	for _, a := range allowed {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		// Match bare host OR "host:port" — users paste either form.
		if hostLower == a {
			return nil
		}
		if strings.HasPrefix(a, hostLower+":") || a == hostLower+":"+u.Port() {
			return nil
		}
	}

	// Block obvious SSRF-target patterns even if a user forgets to update
	// the allowlist — 0.0.0.0 shouldn't be a reachable provider host.
	if hostLower == "0.0.0.0" {
		return fmt.Errorf("%s: host 0.0.0.0 is not a valid provider target", label)
	}

	// Parsed IPs must be loopback OR explicitly allowlisted.
	if ip := net.ParseIP(hostLower); ip != nil {
		if ip.IsLoopback() {
			return nil
		}
		return fmt.Errorf("%s: IP %s is not in the local-provider allowlist (set CLOTHO_LOCAL_PROVIDER_HOSTS to extend)", label, hostLower)
	}

	return fmt.Errorf("%s: host %q is not in the local-provider allowlist (set CLOTHO_LOCAL_PROVIDER_HOSTS to extend)", label, hostLower)
}

// ParseAllowedHosts splits a comma-separated env var into a cleaned list.
func ParseAllowedHosts(raw string) []string {
	if raw = strings.TrimSpace(raw); raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
