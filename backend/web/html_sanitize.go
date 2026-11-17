package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"url-shortener/apperror"
)

// HtmlSanitize runs after LegalPath and rejects HTML in longUrl on POST /api/v1/shorten.
func HtmlSanitize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !inspects(r.Method, r.URL.Path, r.Header.Get("Content-Type")) {
			next.ServeHTTP(w, r)
			return
		}
		data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			r.Body = io.NopCloser(bytes.NewReader(nil))
			next.ServeHTTP(w, r)
			return
		}
		if finding := findingInBody(string(data)); finding != nil {
			apperror.Write(w, finding)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(data))
		next.ServeHTTP(w, r)
	})
}

func inspects(method string, path string, contentType string) bool {
	return method == http.MethodPost &&
		path == "/api/v1/shorten" &&
		strings.Contains(strings.ToLower(contentType), "application/json")
}

func findingInBody(body string) *apperror.Error {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(body), &root); err != nil {
		return nil
	}
	raw, ok := root["longUrl"]
	if !ok {
		return nil
	}
	longURL, ok := raw.(string)
	if !ok {
		return nil
	}
	if LooksUnsafe(longURL) {
		return apperror.DisallowedHTML()
	}
	return nil
}
