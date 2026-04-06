package queue

import (
	"context"
	"log/slog"
	"time"
)

const (
	reapInterval  = 30 * time.Second
	zombieTimeout = 60 * time.Second
)

// reapLoop periodically re-enqueues zombie jobs (stale last_ping > 60s).
func (w *Worker) reapLoop(ctx context.Context) {
	ticker := time.NewTicker(reapInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := w.jobs.ReapZombies(ctx, zombieTimeout)
			if err != nil {
				slog.Error("failed to reap zombie jobs", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("reaped zombie jobs", "count", count)
			}
		}
	}
}
