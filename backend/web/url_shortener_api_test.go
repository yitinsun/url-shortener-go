package web

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"url-shortener/config"
	"url-shortener/repo"
	"url-shortener/service"
)

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	container, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase("shortener_test"),
		tcmysql.WithUsername("root"),
		tcmysql.WithPassword("test"),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = container.Terminate(context.Background()) }()

	host, err := container.Host(ctx)
	if err != nil {
		panic(err)
	}
	mapped, err := container.MappedPort(ctx, "3306/tcp")
	if err != nil {
		panic(err)
	}
	portText := strconv.Itoa(int(mapped.Num()))
	cfg, err := config.FromLookup(func(name string) string {
		switch name {
		case "APP_ENV":
			return "test"
		case "APP_BASE_URL":
			return "http://localhost:8080"
		case "DB_HOST":
			return host
		case "DB_PORT":
			return portText
		case "DB_NAME":
			return "shortener_test"
		case "DB_USER":
			return "root"
		case "DB_PASSWORD":
			return "test"
		default:
			return ""
		}
	})
	if err != nil {
		panic(err)
	}
	if cfg.DBName == "shortener_dev" || cfg.DBName == "shortener_prod" {
		panic("tests must not use shortener_dev or shortener_prod")
	}
	db, err := repo.Open(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	again, err := repo.Open(ctx, cfg)
	if err != nil {
		panic(err)
	}
	again.Close()

	router := NewRouter(service.New(repo.NewMySQL(db), cfg.BaseURL))
	MountOpenAPI(router, cfg.BaseURL)
	sharedDB = db
	sharedHandler = LegalPath(AllowPublic(HtmlSanitize(router)))
	return m.Run()
}

var (
	sharedDB      *sql.DB
	sharedHandler http.Handler
)

func TestPostCreatesAShortLinkAndReusesIt(t *testing.T) {
	payload := `{"longUrl":"https://example.com/api-design/api-design-url-shortening-service/"}`
	rec := call(sharedHandler, http.MethodPost, "/api/v1/shorten", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	code, _ := body["shortCode"].(string)
	if len(code) < 5 || len(code) > 7 {
		t.Fatalf("shortCode %q", code)
	}
	shortURL, _ := body["shortUrl"].(string)
	if !strings.HasPrefix(shortURL, "http://localhost:8080/short/") {
		t.Fatalf("shortUrl %q", shortURL)
	}
	if body["expiresAt"] == nil || body["longUrl"] != "https://example.com/api-design/api-design-url-shortening-service/" {
		t.Fatalf("body %#v", body)
	}

	again := call(sharedHandler, http.MethodPost, "/api/v1/shorten", payload)
	if again.Code != http.StatusCreated {
		t.Fatalf("reuse %d %s", again.Code, again.Body.String())
	}
	reused := decodeJSON(t, again)
	if reused["shortCode"] != code {
		t.Fatalf("reused %#v want %s", reused, code)
	}
}

func TestPostRejectsAnInvalidURLAndASelfLoop(t *testing.T) {
	rec := call(sharedHandler, http.MethodPost, "/api/v1/shorten", `{"longUrl":"not-a-valid-url"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid %d %s", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	status, _ := body["status"].(float64)
	message, _ := body["message"].(string)
	if status != 400 || message == "" {
		t.Fatalf("error body %#v", body)
	}

	loop := call(sharedHandler, http.MethodPost, "/api/v1/shorten", `{"longUrl":"http://localhost:8080/own"}`)
	if loop.Code != http.StatusBadRequest {
		t.Fatalf("loop %d %s", loop.Code, loop.Body.String())
	}
	loopBody := decodeJSON(t, loop)
	if _, ok := loopBody["shortCode"]; !ok || loopBody["shortCode"] != nil {
		t.Fatalf("shortCode %#v", loopBody["shortCode"])
	}
}

func TestGetRedirectsAnActiveCodeAndHidesMissingOrExpired(t *testing.T) {
	longURL := "https://example.com/redirect-target"
	rec := call(sharedHandler, http.MethodPost, "/api/v1/shorten", `{"longUrl":"`+longURL+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	code := decodeJSON(t, rec)["shortCode"].(string)

	active := call(sharedHandler, http.MethodGet, "/short/"+code, "")
	if active.Code != http.StatusFound {
		t.Fatalf("redirect %d %s", active.Code, active.Body.String())
	}
	if active.Header().Get("Location") != longURL {
		t.Fatalf("location %q", active.Header().Get("Location"))
	}

	missing := call(sharedHandler, http.MethodGet, "/short/unknown", "")
	assertFriendlyHTML(t, missing, "/short/unknown")

	if _, err := sharedDB.Exec(
		"INSERT INTO url_mappings (long_url, short_code, created_at, expires_at) VALUES (?, ?, ?, ?)",
		"https://example.com/expired", "expired1", time.Now().UTC().Add(-48*time.Hour), time.Now().UTC().Add(-time.Hour),
	); err != nil {
		t.Fatal(err)
	}
	expired := call(sharedHandler, http.MethodGet, "/short/expired1", "")
	assertFriendlyHTML(t, expired, "/short/expired1")

	removed, err := repo.NewMySQL(sharedDB).DeleteExpiredBefore(context.Background(), time.Now().UTC(), 500)
	if err != nil {
		t.Fatal(err)
	}
	if removed < 1 {
		t.Fatalf("deleted %d", removed)
	}
}

func TestLegalPathsFenceTheRunningAPI(t *testing.T) {
	for _, path := range []string{"/b9K2x1", "/", "/nope"} {
		rec := call(sharedHandler, http.MethodGet, path, "")
		assertFriendlyHTML(t, rec, path)
		if strings.Contains(rec.Body.String(), "404 page not found") {
			t.Fatalf("framework 404 for %s: %s", path, rec.Body.String())
		}
	}
	wrongMethod := call(sharedHandler, http.MethodGet, "/api/v1/shorten", "")
	assertFriendlyHTML(t, wrongMethod, "/api/v1/shorten")

	docs := call(sharedHandler, http.MethodGet, "/v3/api-docs", "")
	if docs.Code != http.StatusOK {
		t.Fatalf("docs %d %s", docs.Code, docs.Body.String())
	}
	body := docs.Body.String()
	if !strings.Contains(body, "/api/v1/shorten") || strings.Contains(body, "/short/{shortCode}") {
		t.Fatalf("docs body %s", body)
	}
}

func TestHTMLInLongURLIsRejectedAndAQueryAmpersandIsNot(t *testing.T) {
	for _, raw := range []string{
		`{"longUrl":"<script>alert(1)</script>"}`,
		`{"longUrl":"https://example.com/<img src=x>"}`,
	} {
		rec := call(sharedHandler, http.MethodPost, "/api/v1/shorten", raw)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("html %d %s", rec.Code, rec.Body.String())
		}
		body := decodeJSON(t, rec)
		if body["message"] != "longUrl contains disallowed HTML" || body["status"] != float64(400) {
			t.Fatalf("payload %#v", body)
		}
		if rec.Header().Get("WWW-Authenticate") != "" {
			t.Fatal("auth challenge on public post")
		}
	}

	payload := call(sharedHandler, http.MethodPost, "/api/v1/shorten", `{"longUrl":"javascript:alert(1)"}`)
	if payload.Code != http.StatusBadRequest || decodeJSON(t, payload)["message"] != "longUrl contains a disallowed payload" {
		t.Fatalf("payload message %d %s", payload.Code, payload.Body.String())
	}

	query := call(sharedHandler, http.MethodPost, "/api/v1/shorten", `{"longUrl":"https://example.com/search?q=go&lang=en"}`)
	if query.Code != http.StatusCreated {
		t.Fatalf("query url %d %s", query.Code, query.Body.String())
	}

	missing := call(sharedHandler, http.MethodGet, "/nope", "")
	if missing.Code != http.StatusNotFound || missing.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("unknown %d %q", missing.Code, missing.Header().Get("WWW-Authenticate"))
	}
}

func assertFriendlyHTML(t *testing.T, rec *httptest.ResponseRecorder, path string) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content type %q", rec.Header().Get("Content-Type"))
	}
	html := rec.Body.String()
	for _, want := range []string{"<h1>Link not found</h1>", path, "POST /api/v1/shorten", "/swagger-ui.html", "/short/{code}"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(html), "{") {
		t.Fatalf("json body %s", html)
	}
}
