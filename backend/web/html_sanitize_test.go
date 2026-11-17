package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInspectionFollowsThePostJSONContract(t *testing.T) {
	if !inspects(http.MethodPost, "/api/v1/shorten", "application/json; charset=utf-8") {
		t.Fatal("charset json")
	}
	if !inspects(http.MethodPost, "/api/v1/shorten", "APPLICATION/JSON") {
		t.Fatal("upper json")
	}
	if inspects(http.MethodGet, "/api/v1/shorten", "application/json") ||
		inspects(http.MethodPost, "/short/abc", "application/json") ||
		inspects(http.MethodPost, "/api/v1/shorten", "") ||
		inspects(http.MethodPost, "/api/v1/shorten", "text/plain") {
		t.Fatal("inspection scope")
	}
	for _, body := range []string{"", "   ", "not-json", "{}", `{"longUrl":1}`, `{"longUrl":"https://example.com/path"}`} {
		if findingInBody(body) != nil {
			t.Fatalf("unexpected finding for %q", body)
		}
	}
	finding := findingInBody(`{"longUrl":"<img src=x>"}`)
	if finding == nil || finding.Error() != "longUrl contains disallowed HTML" {
		t.Fatalf("finding %v", finding)
	}
}

func TestHtmlSanitizeRejectsMarkupAndKeepsAPlainURL(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	})
	app := HtmlSanitize(inner)

	clean := httptest.NewRecorder()
	cleanReq := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", strings.NewReader(`{"longUrl":"https://example.com/path"}`))
	cleanReq.Header.Set("Content-Type", "application/json")
	app.ServeHTTP(clean, cleanReq)
	if clean.Code != http.StatusOK || !strings.Contains(clean.Body.String(), "https://example.com/path") {
		t.Fatalf("clean %d %s", clean.Code, clean.Body.String())
	}

	markup := httptest.NewRecorder()
	markupReq := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", strings.NewReader(`{"longUrl":"<script>alert(1)</script>"}`))
	markupReq.Header.Set("Content-Type", "application/json")
	app.ServeHTTP(markup, markupReq)
	if markup.Code != http.StatusBadRequest {
		t.Fatalf("markup status %d", markup.Code)
	}
	body := markup.Body.String()
	if !strings.Contains(body, "disallowed HTML") || !strings.Contains(body, `"status":400`) || !strings.Contains(body, `"shortCode":null`) {
		t.Fatalf("markup body %s", body)
	}

	other := httptest.NewRecorder()
	otherReq := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", strings.NewReader(`{"longUrl":"<b>x</b>"}`))
	otherReq.Header.Set("Content-Type", "text/plain")
	app.ServeHTTP(other, otherReq)
	if other.Code != http.StatusOK {
		t.Fatalf("non-json %d", other.Code)
	}
}
