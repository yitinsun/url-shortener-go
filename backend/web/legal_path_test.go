package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAllowlistMatchesThePlaybook(t *testing.T) {
	allow := LegalPaths{}
	cases := []struct {
		method  string
		path    string
		allowed bool
	}{
		{http.MethodPost, "/api/v1/shorten", true},
		{http.MethodGet, "/api/v1/shorten", false},
		{http.MethodGet, "/short/b9K2x1", true},
		{http.MethodGet, "/short/0", true},
		{http.MethodGet, "/short/1234567890", true},
		{http.MethodGet, "/short/", false},
		{http.MethodGet, "/short/12345678901", false},
		{http.MethodGet, "/short/abc_def", false},
		{http.MethodPost, "/short/abc", false},
		{http.MethodGet, "/b9K2x1", false},
		{http.MethodGet, "/", false},
		{http.MethodGet, "/index.html", false},
		{http.MethodGet, "/foo", false},
		{http.MethodGet, "/swagger-ui.html", true},
		{http.MethodGet, "/swagger-ui/swagger-ui.css", true},
		{http.MethodGet, "/swagger-ui", false},
		{http.MethodGet, "/v3/api-docs", true},
		{http.MethodGet, "/v3/api-docs.yaml", true},
		{http.MethodGet, "/v3/api-docs/extra", true},
		{http.MethodPut, "/v3/api-docs", false},
		{http.MethodGet, "/actuator/health", false},
		{http.MethodGet, "/metrics", false},
	}
	for _, tc := range cases {
		if allow.Allows(tc.method, tc.path) != tc.allowed {
			t.Fatalf("%s %s allowed=%v", tc.method, tc.path, tc.allowed)
		}
	}
}

func TestLegalPathBlocksIllegalRequestsBeforeTheHandler(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("reached"))
	})
	app := LegalPath(inner)

	created := httptest.NewRecorder()
	app.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/shorten", nil))
	if created.Code != http.StatusOK || created.Body.String() != "reached" {
		t.Fatalf("post %d %s", created.Code, created.Body.String())
	}

	redirect := httptest.NewRecorder()
	app.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/short/abc", nil))
	if redirect.Code != http.StatusOK || redirect.Body.String() != "reached" {
		t.Fatalf("get short %d %s", redirect.Code, redirect.Body.String())
	}

	wrong := httptest.NewRecorder()
	app.ServeHTTP(wrong, httptest.NewRequest(http.MethodGet, "/api/v1/shorten", nil))
	assertBlocked(t, wrong, "/api/v1/shorten")

	unknown := httptest.NewRecorder()
	app.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/b9K2x1", nil))
	assertBlocked(t, unknown, "/b9K2x1")

	quotedReq := httptest.NewRequest(http.MethodGet, "/quote", nil)
	quotedReq.URL.Path = `/a&b<c">`
	quoted := httptest.NewRecorder()
	app.ServeHTTP(quoted, quotedReq)
	if quoted.Code != http.StatusNotFound {
		t.Fatalf("quoted %d", quoted.Code)
	}
	html := quoted.Body.String()
	if strings.Contains(html, "<c>") || !strings.Contains(html, "&amp;") || !strings.Contains(html, "&quot;") {
		t.Fatalf("path was not escaped: %s", html)
	}
}

func assertBlocked(t *testing.T, rec *httptest.ResponseRecorder, path string) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
	html := rec.Body.String()
	for _, want := range []string{"<h1>Link not found</h1>", "<p>" + path + "</p>", "POST /api/v1/shorten", "/swagger-ui.html", "/short/{code}"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
	if strings.Contains(html, "reached") || strings.Contains(html, "Method Not Allowed") || strings.HasPrefix(strings.TrimSpace(html), "404") {
		t.Fatalf("framework body leaked: %s", html)
	}
}
