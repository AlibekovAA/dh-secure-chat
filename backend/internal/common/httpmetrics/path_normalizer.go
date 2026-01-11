package httpmetrics

import (
	"regexp"
	"strings"
)

var (
	uuidRegex = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
)

func NormalizePath(path string) string {
	if path == "" {
		return "/"
	}

	normalized := uuidRegex.ReplaceAllString(path, "{id}")

	parts := strings.Split(normalized, "/")
	for i, part := range parts {
		if part != "" && (strings.HasPrefix(part, "{") || isNumeric(part)) {
			parts[i] = "{param}"
		}
	}

	result := strings.Join(parts, "/")
	if result == "" {
		return "/"
	}

	return result
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
