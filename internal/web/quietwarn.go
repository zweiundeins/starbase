package web

import (
	"log/slog"
	"sync"
	"time"
)

// quietWarn logs a refusal at most once a minute, with how many it left out:
// a refused stream asks again every few seconds.
type quietWarn struct {
	mu      sync.Mutex
	last    time.Time
	skipped int
}

func (q *quietWarn) warn(log *slog.Logger, msg string, args ...any) {
	q.mu.Lock()
	now := time.Now()
	if now.Sub(q.last) < time.Minute {
		q.skipped++
		q.mu.Unlock()
		return
	}
	skipped := q.skipped
	q.last, q.skipped = now, 0
	q.mu.Unlock()
	log.Warn(msg, append(args, "since_last", skipped)...)
}
