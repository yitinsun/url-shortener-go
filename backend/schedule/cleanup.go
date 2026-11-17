package schedule

import (
	"context"
	"log/slog"
	"time"

	"url-shortener/repo"
)

// DeleteBatch is the maximum number of expired rows removed on one pass.
const DeleteBatch int64 = 500

// CleanupInterval is the greenfield cadence.
const CleanupInterval = time.Hour

// DeleteExpired runs the fixed, parameterized delete.
func DeleteExpired(ctx context.Context, store repo.Repository, cutoff time.Time) (int64, error) {
	return store.DeleteExpiredBefore(ctx, cutoff, DeleteBatch)
}

// Start runs DeleteExpired on every interval until ctx is cancelled.
// The first pass waits one full interval, matching a skipped initial tick.
func Start(ctx context.Context, store repo.Repository, every time.Duration) {
	go func() {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().UTC()
				removed, err := DeleteExpired(ctx, store, cutoff)
				switch {
				case err != nil:
					slog.Error("expired mapping cleanup failed", "error", err)
				case removed > 0:
					slog.Info("removed expired url mappings", "removed", removed)
				}
			}
		}
	}()
}
