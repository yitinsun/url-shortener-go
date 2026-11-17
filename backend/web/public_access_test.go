package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDocumentedRoutesArePublic(t *testing.T) {
	access := PublicAccess{}
	for _, path := range []string{
		"/api/v1/shorten",
		"/short/abc",
		"/swagger-ui.html",
		"/swagger-ui/swagger-ui.css",
		"/v3/api-docs",
		"/v3/api-docs.yaml",
		"/v3/api-docs/extra",
	} {
		if !access.IsDocumented(path) {
			t.Fatalf("expected public %s", path)
		}
	}
	for _, path := range []string{"/", "/nope", "/actuator/health"} {
		if access.IsDocumented(path) {
			t.Fatalf("expected undocumented %s", path)
		}
	}
}

func TestAuthorizationHeaderDoesNotChallenge(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	app := AllowPublic(inner)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", nil)
	req.Header.Set("Authorization", "Bearer demo")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("challenge %q", rec.Header().Get("WWW-Authenticate"))
	}

	unknown := httptest.NewRecorder()
	app.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if unknown.Code == http.StatusUnauthorized || unknown.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("unknown challenged %d", unknown.Code)
	}
}
