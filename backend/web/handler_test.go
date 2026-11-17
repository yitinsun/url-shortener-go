package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"url-shortener/apperror"
	"url-shortener/repo"
	"url-shortener/service"
)

func TestHandlersRejectBadJSONAndMapDatabaseFailures(t *testing.T) {
	svc := service.New(&failingRepo{}, "http://localhost:8080")
	handler := NewRouter(svc)

	bad := call(handler, http.MethodPost, "/api/v1/shorten", "{")
	if bad.Code != http.StatusBadRequest || !strings.Contains(bad.Body.String(), "not valid JSON") {
		t.Fatalf("bad json %d %s", bad.Code, bad.Body.String())
	}

	down := call(handler, http.MethodPost, "/api/v1/shorten", `{"longUrl":"https://example.com/down"}`)
	if down.Code != http.StatusServiceUnavailable {
		t.Fatalf("create down %d %s", down.Code, down.Body.String())
	}

	missing := call(handler, http.MethodGet, "/short/missing", "")
	if missing.Code != http.StatusServiceUnavailable {
		t.Fatalf("redirect down %d %s", missing.Code, missing.Body.String())
	}
}

type failingRepo struct{}

func (failingRepo) FindByLongURLPrefix(context.Context, string) ([]repo.URLMapping, error) {
	return nil, apperror.Unavailable("down")
}
func (failingRepo) FindByShortCode(context.Context, string) (*repo.URLMapping, error) {
	return nil, apperror.Unavailable("down")
}
func (failingRepo) InsertWithEncodedID(context.Context, string, string, time.Time, time.Time, func(uint64) string) (repo.URLMapping, error) {
	return repo.URLMapping{}, apperror.Unavailable("down")
}
func (failingRepo) DeleteExpiredBefore(context.Context, time.Time, int64) (int64, error) {
	return 0, apperror.Unavailable("down")
}

func call(handler http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" || method == http.MethodPost {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json %s: %v", rec.Body.String(), err)
	}
	return payload
}
