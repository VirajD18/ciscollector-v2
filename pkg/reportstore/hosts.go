package reportstore

import "strings"

// NormalizeHost maps loopback aliases to localhost for stable target_id.
func NormalizeHost(host string) string {
	h := strings.TrimSpace(strings.ToLower(host))
	if h == "" || h == "127.0.0.1" || h == "::1" {
		return "localhost"
	}
	return strings.TrimSpace(host)
}
