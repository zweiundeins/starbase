package web

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"starbase/internal/commands"
	"starbase/internal/queries"
)

// boardCache shares the encoded board between all viewers of one version:
// every stream re-renders after a paint, but only the first reads the cells.
type boardCache struct {
	version int64
	cells   string
}

// boardState returns the encoded board and its total painted pixels.
func (s *Server) boardState(ctx context.Context, r *queries.Reader) (cells string, pixels int64, err error) {
	version, pixels, err := r.BoardMeta(ctx, commands.BoardName)
	if err != nil {
		return "", 0, err
	}
	if c := s.board.Load(); c != nil && c.version == version {
		return c.cells, pixels, nil
	}
	cells, err = r.BoardCells(ctx, commands.BoardName, commands.BoardSize)
	if err != nil {
		return "", 0, err
	}
	s.board.Store(&boardCache{version: version, cells: cells})
	return cells, pixels, nil
}

// limiter is a per-session token bucket for painting: an edge concern that
// protects the shared board from floods, not domain state.
type limiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	at     time.Time
}

func newLimiter(rate, burst float64) *limiter {
	return &limiter{rate: rate, burst: burst, buckets: map[string]*bucket{}}
}

// allow takes n tokens for key, if available.
func (l *limiter) allow(key string, n int, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) > 10000 { // forget idle sessions
			for k, v := range l.buckets {
				if now.Sub(v.at) > time.Minute {
					delete(l.buckets, k)
				}
			}
		}
		b = &bucket{tokens: l.burst, at: now}
		l.buckets[key] = b
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.at).Seconds()*l.rate)
	b.at = now
	if b.tokens < float64(n) {
		return false
	}
	b.tokens -= float64(n)
	return true
}

// cmdPaint paints cells of the shared board (anyone may paint, rate-limited).
func (s *Server) cmdPaint(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Color int   `json:"color"`
		Cells []int `json:"cells"`
	}
	if err := datastar.ReadSignals(r, &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	now := time.Now()
	if !s.paintLimit.allow(sessionID(r), len(p.Cells), now) || !s.paintLimitIP.allow(clientIP(r), len(p.Cells), now) {
		http.Error(w, "painting too fast", http.StatusTooManyRequests)
		return
	}
	s.send(w, r, commands.PaintPixels{Board: commands.BoardName, Color: p.Color, Cells: p.Cells})
}
