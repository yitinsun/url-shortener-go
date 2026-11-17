package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"url-shortener/apperror"
	"url-shortener/repo"
)

type memRepo struct {
	mu      sync.Mutex
	rows    []repo.URLMapping
	inserts int
	nextID  int64
	fail    bool
}

func (m *memRepo) FindByLongURLPrefix(_ context.Context, prefix string) ([]repo.URLMapping, error) {
	if m.fail {
		return nil, apperror.Unavailable("down")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []repo.URLMapping
	for _, row := range m.rows {
		got := row.LongURL
		if len([]rune(got)) > 255 {
			got = string([]rune(got)[:255])
		}
		if got == prefix {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *memRepo) FindByShortCode(_ context.Context, shortCode string) (*repo.URLMapping, error) {
	if m.fail {
		return nil, apperror.Unavailable("down")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range m.rows {
		if row.ShortCode == shortCode {
			copy := row
			return &copy, nil
		}
	}
	return nil, nil
}

func (m *memRepo) InsertWithEncodedID(_ context.Context, longURL string, _ string, createdAt time.Time, expiresAt time.Time, encode func(uint64) string) (repo.URLMapping, error) {
	if m.fail {
		return repo.URLMapping{}, apperror.Unavailable("down")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inserts++
	if m.nextID == 0 {
		m.nextID = 56800235
	}
	id := m.nextID
	m.nextID++
	mapping := repo.URLMapping{
		ID:        id,
		LongURL:   longURL,
		ShortCode: encode(uint64(id)),
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
	m.rows = append(m.rows, mapping)
	return mapping, nil
}

func (m *memRepo) DeleteExpiredBefore(context.Context, time.Time, int64) (int64, error) {
	return 0, nil
}

func fixedNow() time.Time {
	return time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
}

func TestMissingShortCodeIsNotFound(t *testing.T) {
	svc := New(&memRepo{}, "http://localhost:8080")
	_, err := svc.Resolve(context.Background(), "missing", fixedNow())
	if !apperror.IsNotFound(err) {
		t.Fatalf("got %v", err)
	}
}

func TestExpiredMappingIsNotFound(t *testing.T) {
	store := &memRepo{rows: []repo.URLMapping{{
		ID: 56800235, LongURL: "https://example.com/old", ShortCode: "abcde",
		CreatedAt: fixedNow().Add(-10 * 24 * time.Hour), ExpiresAt: fixedNow().Add(-time.Second),
	}}}
	svc := New(store, "http://localhost:8080")
	_, err := svc.Resolve(context.Background(), "abcde", fixedNow())
	if !apperror.IsNotFound(err) {
		t.Fatalf("got %v", err)
	}
}

func TestActiveLongURLDoesNotInsertAgain(t *testing.T) {
	store := &memRepo{}
	svc := New(store, "http://localhost:8080")
	ttl := int64(10)
	other := int64(99)
	first, err := svc.Create(context.Background(), "https://example.com/same", &ttl, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Create(context.Background(), "https://example.com/same", &other, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	if first.ShortCode != second.ShortCode || first.ExpiresAt != second.ExpiresAt {
		t.Fatalf("reused mapping changed: %+v %+v", first, second)
	}
	if store.inserts != 1 {
		t.Fatalf("inserts %d", store.inserts)
	}
}

func TestOmittedTTLIs730Days(t *testing.T) {
	svc := New(&memRepo{}, "http://localhost:8080")
	created, err := svc.Create(context.Background(), "https://example.com/default-ttl", nil, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	want := fixedNow().Add(730 * 24 * time.Hour)
	if !created.ExpiresAt.Equal(want) {
		t.Fatalf("expires %s want %s", created.ExpiresAt, want)
	}
}

func TestSelfLoopAndOversizeAreRejected(t *testing.T) {
	store := &memRepo{}
	svc := New(store, "http://localhost:8080")
	if _, err := svc.Create(context.Background(), "http://LOCALHOST:8080/own", nil, fixedNow()); !apperror.IsBadRequest(err) {
		t.Fatalf("self-loop %v", err)
	}
	over := "https://example.com/" + stringsRepeat(2048)
	if len(over) <= 2048 {
		t.Fatal("fixture is not oversize")
	}
	if _, err := svc.Create(context.Background(), over, nil, fixedNow()); !apperror.IsBadRequest(err) {
		t.Fatalf("oversize %v", err)
	}
	if store.inserts != 0 {
		t.Fatalf("inserts %d", store.inserts)
	}
}

func TestValidationMessages(t *testing.T) {
	svc := New(&memRepo{}, "http://localhost:8080")
	cases := []struct {
		input   string
		message string
	}{
		{"", "longUrl is required"},
		{"   ", "longUrl is required"},
		{"javascript:alert(1)", "longUrl contains a disallowed payload"},
		{"DATA:text/html,hi", "longUrl contains a disallowed payload"},
		{"https://example.com/<script>", "longUrl contains a disallowed payload"},
		{"not a url", "longUrl is not a valid URI"},
		{"ftp://example.com/file", "longUrl must be HTTP or HTTPS"},
		{"http://", "longUrl must include a host"},
	}
	for _, tc := range cases {
		_, err := svc.Create(context.Background(), tc.input, nil, fixedNow())
		if err == nil || err.Error() != tc.message {
			t.Fatalf("%q => %v", tc.input, err)
		}
	}
	zero := int64(0)
	high := int64(3651)
	one := int64(1)
	if _, err := svc.Create(context.Background(), "https://example.com/ttl", &zero, fixedNow()); err == nil || err.Error() != "ttlInDays must be between 1 and 3650" {
		t.Fatalf("low ttl %v", err)
	}
	if _, err := svc.Create(context.Background(), "https://example.com/ttl", &high, fixedNow()); err == nil || err.Error() != "ttlInDays must be between 1 and 3650" {
		t.Fatalf("high ttl %v", err)
	}
	edge, err := svc.Create(context.Background(), "https://example.com/edge", &one, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	if !edge.ExpiresAt.Equal(fixedNow().Add(24*time.Hour)) || edge.ShortURL != "http://localhost:8080/short/"+edge.ShortCode {
		t.Fatalf("%+v", edge)
	}
}

func TestExpiredLongURLIsInsertedAgain(t *testing.T) {
	store := &memRepo{rows: []repo.URLMapping{{
		ID: 10, LongURL: "https://example.com/again", ShortCode: "old",
		CreatedAt: fixedNow().Add(-72 * time.Hour), ExpiresAt: fixedNow().Add(-time.Second),
	}}, nextID: 11}
	svc := New(store, "http://localhost:8080/")
	days := int64(3650)
	created, err := svc.Create(context.Background(), "https://example.com/again", &days, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	if created.ShortCode == "old" || store.inserts != 1 {
		t.Fatalf("created %+v inserts %d", created, store.inserts)
	}
	if svc.BaseURL() != "http://localhost:8080" {
		t.Fatalf("base %q", svc.BaseURL())
	}
}

func TestRepositoryFailuresSurfaceAsUnavailable(t *testing.T) {
	svc := New(&memRepo{fail: true}, "http://localhost:8080/")
	if _, err := svc.Create(context.Background(), "https://example.com/down", nil, fixedNow()); !isUnavailable(err) {
		t.Fatalf("create %v", err)
	}
	if _, err := svc.Resolve(context.Background(), "code", fixedNow()); !isUnavailable(err) {
		t.Fatalf("resolve %v", err)
	}
	bad := New(&memRepo{}, "://bad")
	if _, err := bad.Create(context.Background(), "https://example.com/ok", nil, fixedNow()); !isUnavailable(err) {
		t.Fatalf("base %v", err)
	}
}

func TestURLsThatDifferAfter255CharactersAreDistinct(t *testing.T) {
	store := &memRepo{}
	svc := New(store, "http://localhost:8080")
	prefix := "https://example.com/" + stringsRepeat(240)
	first := prefix + "aaa"
	second := prefix + "bbb"
	if _, err := svc.Create(context.Background(), first, nil, fixedNow()); err != nil {
		t.Fatal(err)
	}
	created, err := svc.Create(context.Background(), second, nil, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	if store.inserts != 2 || created.LongURL != second {
		t.Fatalf("inserts %d created %+v", store.inserts, created)
	}
}

func isUnavailable(err error) bool {
	return err != nil && !apperror.IsBadRequest(err) && !apperror.IsNotFound(err)
}

func stringsRepeat(n int) string {
	return stringsOf('a', n)
}

func stringsOf(ch rune, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(ch)
	}
	return string(b)
}
