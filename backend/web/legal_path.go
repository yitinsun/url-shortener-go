package web

import (
	"net/http"
	"strings"

	"url-shortener/apperror"
)

// LegalPaths is the public URL allowlist. Anything else is the shared HTML 404.
type LegalPaths struct{}

// Allows reports whether method and path may enter the handlers.
func (LegalPaths) Allows(method string, path string) bool {
	if method == http.MethodPost && path == "/api/v1/shorten" {
		return true
	}
	if method != http.MethodGet {
		return false
	}
	switch path {
	case "/swagger-ui.html", "/v3/api-docs", "/v3/api-docs.yaml":
		return true
	}
	if strings.HasPrefix(path, "/swagger-ui/") || strings.HasPrefix(path, "/v3/api-docs/") {
		return true
	}
	return isShortCode(path)
}

func isShortCode(path string) bool {
	const prefix = "/short/"
	if len(path) <= len(prefix) || path[:len(prefix)] != prefix {
		return false
	}
	code := path[len(prefix):]
	if len(code) < 1 || len(code) > 10 {
		return false
	}
	for i := 0; i < len(code); i++ {
		ch := code[i]
		switch {
		case ch >= '0' && ch <= '9', ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z':
		default:
			return false
		}
	}
	return true
}

// LegalPath is the fence outside the handlers. Illegal requests never reach a redirect or a JSON body.
func LegalPath(next http.Handler) http.Handler {
	allow := LegalPaths{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allow.Allows(r.Method, r.URL.Path) {
			apperror.WriteNotFound(w, r.URL.Path)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func friendlyMiss(w http.ResponseWriter, r *http.Request) {
	apperror.WriteNotFound(w, r.URL.Path)
}
