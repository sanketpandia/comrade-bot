package auth

import (
	"fmt"
	"net/url"
	"strings"

	"infinite-experiment/politburo/internal/routes"
)

const DefaultRedirect = routes.Dashboard

func NormalizeRedirect(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return DefaultRedirect, nil
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.Contains(path, "://") {
		return "", fmt.Errorf("redirect must be a relative path")
	}
	return path, nil
}

func FormatLoginURL(baseURL, token string) string {
	return strings.TrimRight(baseURL, "/") + routes.AuthLogin + "?token=" + url.QueryEscape(token)
}
