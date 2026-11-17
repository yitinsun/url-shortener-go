package schedule

import (
	"context"
	"sync"
	"testing"
	"time"

	"url-shortener/apperror"
	"url-shortener/repo"
)

func TestCleanupBindsACutoffAndUsesPlaceholders(t *testing.T) {
	if count(repo.DeleteExpiredSQL, '?') != 2 {
		t.Fatalf("sql %q", repo.DeleteExpiredSQL)
	}
	if contains(repo.DeleteExpiredSQL, "Sprintf") || contains(repo.DeleteExpiredSQL, "+") {
		t.Fatalf("sql was built dynamically: %q", repo.DeleteExpiredSQL)
	}
	cutoff := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	store := &recordingRepo{}
	if _, err := DeleteExpired(context.Background(), store, cutoff); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.calls) != 1 || !store.calls[0].cutoff.Equal(cutoff) || store.calls[0].limit != DeleteBatch {
		t.Fatalf("calls %+v", store.calls)
	}
}

func TestStartCoversRemovedRowsErrorsAndAQuietPass(t *testing.T) {
	store := &recordingRepo{results: []result{{n: 4}, {err: apperror.Unavailable("db down")}, {n: 0}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	Start(ctx, store, 5*time.Millisecond)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		store.mu.Lock()
		n := len(store.calls)
		store.mu.Unlock()
		if n >= 3 {
			cancel()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("cleanup ran %d times", len(store.calls))
}

type call struct {
	cutoff time.Time
	limit  int64
}

type result struct {
	n   int64
	err error
}

type recordingRepo struct {
	mu      sync.Mutex
	calls   []call
	results []result
}

func (r *recordingRepo) FindByLongURLPrefix(context.Context, string) ([]repo.URLMapping, error) {
	return nil, nil
}

func (r *recordingRepo) FindByShortCode(context.Context, string) (*repo.URLMapping, error) {
	return nil, nil
}

func (r *recordingRepo) InsertWithEncodedID(context.Context, string, string, time.Time, time.Time, func(uint64) string) (repo.URLMapping, error) {
	return repo.URLMapping{}, apperror.Unavailable("unused")
}

func (r *recordingRepo) DeleteExpiredBefore(_ context.Context, cutoff time.Time, limit int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call{cutoff: cutoff, limit: limit})
	if len(r.results) == 0 {
		return 0, nil
	}
	last := r.results[len(r.results)-1]
	r.results = r.results[:len(r.results)-1]
	return last.n, last.err
}

func count(value string, ch byte) int {
	n := 0
	for i := 0; i < len(value); i++ {
		if value[i] == ch {
			n++
		}
	}
	return n
}

func contains(value string, part string) bool {
	return len(part) > 0 && len(value) >= len(part) && (func() bool {
		for i := 0; i+len(part) <= len(value); i++ {
			if value[i:i+len(part)] == part {
				return true
			}
		}
		return false
	})()
}
