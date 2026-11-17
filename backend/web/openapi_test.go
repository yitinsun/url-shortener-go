package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"url-shortener/service"
)

func TestOpenAPIDocumentListsOnlyTheCreateAPI(t *testing.T) {
	handler := NewRouter(service.New(failingRepo{}, "http://localhost:8080"))
	MountOpenAPI(handler, "https://short.example")

	rec := call(handler, http.MethodGet, "/v3/api-docs", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("json %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content type %q", rec.Header().Get("Content-Type"))
	}
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.0.3" {
		t.Fatalf("openapi %#v", doc["openapi"])
	}
	info, _ := doc["info"].(map[string]any)
	if info["title"] != "URL Shortener API" || info["version"] != "0.1.0" {
		t.Fatalf("info %#v", info)
	}
	servers, _ := doc["servers"].([]any)
	if len(servers) != 1 || servers[0].(map[string]any)["url"] != "https://short.example" {
		t.Fatalf("servers %#v", servers)
	}
	paths, _ := doc["paths"].(map[string]any)
	if _, ok := paths["/api/v1/shorten"]; !ok {
		t.Fatalf("paths %#v", paths)
	}
	if _, ok := paths["/short/{shortCode}"]; ok {
		t.Fatal("redirect route was published")
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "Bearer") || strings.Contains(raw, "/short/{shortCode}") {
		t.Fatalf("document leaked auth or redirect: %s", raw)
	}
	post := paths["/api/v1/shorten"].(map[string]any)["post"].(map[string]any)
	responses := post["responses"].(map[string]any)
	for _, code := range []string{"201", "400", "503"} {
		if _, ok := responses[code]; !ok {
			t.Fatalf("missing %s in %#v", code, responses)
		}
	}

	yaml := call(handler, http.MethodGet, "/v3/api-docs.yaml", "")
	if yaml.Code != http.StatusOK || !strings.Contains(yaml.Header().Get("Content-Type"), "yaml") {
		t.Fatalf("yaml %d %s", yaml.Code, yaml.Header().Get("Content-Type"))
	}
	if !strings.Contains(yaml.Body.String(), "openapi:") || !strings.Contains(yaml.Body.String(), "/api/v1/shorten") {
		t.Fatalf("yaml body %s", yaml.Body.String())
	}

	page := call(handler, http.MethodGet, "/swagger-ui.html", "")
	if page.Code != http.StatusOK || !strings.Contains(page.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("ui %d %s", page.Code, page.Header().Get("Content-Type"))
	}
	if !strings.Contains(page.Body.String(), "URL Shortener API") || !strings.Contains(page.Body.String(), "0.1.0") {
		t.Fatalf("ui body %s", page.Body.String())
	}
	asset := call(handler, http.MethodGet, "/swagger-ui/swagger-ui.css", "")
	if asset.Code != http.StatusOK || !strings.Contains(asset.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("css %d %s", asset.Code, asset.Header().Get("Content-Type"))
	}
}

func TestOpenAPIStaysOffUnlessMounted(t *testing.T) {
	handler := NewRouter(service.New(failingRepo{}, "http://localhost:8080"))
	rec := call(handler, http.MethodGet, "/v3/api-docs", "")
	if rec.Code == http.StatusOK {
		t.Fatal("openapi was mounted by default")
	}
}
