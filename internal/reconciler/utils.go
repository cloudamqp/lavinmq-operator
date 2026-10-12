package reconciler

import "strings"

// SanitizeHostnameForPath sanitizes a hostname for use in filesystem paths
// Converts wildcards, dots, and ensures Unix path compatibility
// Example: "*.example.com" -> "wildcard-example-com"
func SanitizeHostnameForPath(hostname string) string {
	sanitized := strings.ReplaceAll(hostname, ".", "-")
	sanitized = strings.ReplaceAll(sanitized, "*", "wildcard")
	sanitized = strings.ToLower(sanitized)
	sanitized = strings.Trim(sanitized, "-")

	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		sanitized = strings.TrimRight(sanitized, "-")
	}

	return sanitized
}
