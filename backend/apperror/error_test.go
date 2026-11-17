package apperror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEachErrorMapsToItsStatus(t *testing.T) {
	bad := BadRequest("longUrl is required")
	if !IsBadRequest(bad) || IsNotFound(bad) {
		t.Fatal("bad request classification")
	}
	if bad.Error() != "longUrl is required" {
		t.Fatalf("message %q", bad.Error())
	}
	rec := httptest.NewRecorder()
	Write(rec, bad)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
	var payload body
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != 400 || payload.Error != "Bad Request" || payload.Message != "longUrl is required" || payload.ShortCode != nil {
		t.Fatalf("payload %+v", payload)
	}

	missing := httptest.NewRecorder()
	Write(missing, NotFound())
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), "<h1>Link not found</h1>") {
		t.Fatalf("not found %d %s", missing.Code, missing.Body.String())
	}
	if IsBadRequest(NotFound()) || NotFound().Error() != "short link was not found" {
		t.Fatal("not found classification")
	}

	down := httptest.NewRecorder()
	Write(down, Unavailable("down"))
	if down.Code != http.StatusServiceUnavailable || !strings.Contains(down.Body.String(), "Service Unavailable") {
		t.Fatalf("unavailable %d %s", down.Code, down.Body.String())
	}

	other := httptest.NewRecorder()
	Write(other, errString("plain"))
	if other.Code != http.StatusServiceUnavailable || !strings.Contains(other.Body.String(), "plain") {
		t.Fatalf("fallback %d %s", other.Code, other.Body.String())
	}
}

func TestFriendlyPageEscapesThePath(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteNotFound(rec, `a&b<c>"quoted"`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content type %q", rec.Header().Get("Content-Type"))
	}
	html := rec.Body.String()
	for _, want := range []string{
		"<title>Link not found</title>",
		"<h1>Link not found</h1>",
		"This short link is missing or has expired.",
		"a&amp;b&lt;c&gt;&quot;quoted&quot;",
		"POST /api/v1/shorten",
		"/swagger-ui.html",
		"/short/{code}",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
	if strings.Contains(html, "<c>") || strings.HasPrefix(html, "{") {
		t.Fatalf("unescaped or JSON page: %s", html)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
