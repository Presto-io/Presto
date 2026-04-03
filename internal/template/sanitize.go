package template

import (
	"net/url"
	"os"
	"strings"
)

// SanitizeURL removes sensitive query parameters from URLs for logging.
// Example: https://example.com?token=secret → https://example.com?token=***
func SanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Invalid URL, return as-is
	}

	// Remove sensitive query parameters
	sensitiveParams := []string{"token", "key", "signature", "auth", "api_key"}
	query := u.Query()
	for _, param := range sensitiveParams {
		if query.Has(param) {
			query.Set(param, "***")
		}
	}
	u.RawQuery = query.Encode()

	return u.String()
}

// SanitizePath removes user home directory from file paths for logging.
// Example: /Users/alice/.presto → ~/.presto
func SanitizePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path // Fallback: no sanitization
	}
	if home != "" && strings.Contains(path, home) {
		return strings.ReplaceAll(path, home, "~")
	}
	return path
}
