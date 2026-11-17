package web

import (
	"net/http"
	"strings"
)

// PublicAccess documents the routes that stay public. This demo issues no authentication challenge.
type PublicAccess struct{}

// IsDocumented reports a route named by the public-access playbook.
func (PublicAccess) IsDocumented(path string) bool {
	switch path {
	case "/api/v1/shorten", "/swagger-ui.html", "/v3/api-docs", "/v3/api-docs.yaml":
		return true
	}
	return strings.HasPrefix(path, "/short/") ||
		strings.HasPrefix(path, "/swagger-ui/") ||
		strings.HasPrefix(path, "/v3/api-docs/")
}

// AllowPublic lets every request through. CSRF, form login, HTTP basic, and logout are not enabled.
func AllowPublic(next http.Handler) http.Handler {
	access := PublicAccess{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = access.IsDocumented(r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
