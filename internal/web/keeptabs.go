package web

import (
	"context"
	"sync"
	"time"

	"starbase/internal/commands"
)

// openTabs counts the render streams under way per tab, so that the tab
// rows of open pages outlive the sweep (commands.TabTTL).
type openTabs struct {
	mu sync.Mutex
	n  map[commands.TabKey]int
}

func (o *openTabs) add(k commands.TabKey) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.n == nil {
		o.n = map[commands.TabKey]int{}
	}
	o.n[k]++
}

func (o *openTabs) remove(k commands.TabKey) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.n[k]--; o.n[k] <= 0 {
		delete(o.n, k)
	}
}

// keepTabs marks the rows of the tabs whose stream is open as touched, every
// interval, in one command that wakes no view.
func (s *Server) keepTabs(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		s.tabs.mu.Lock()
		var keep []commands.TabKey
		for k := range s.tabs.n {
			if s.hub.HasStream(streamID(k.SID, k.Tab)) {
				keep = append(keep, k)
			}
		}
		s.tabs.mu.Unlock()
		if len(keep) == 0 {
			continue
		}
		if err := s.writer.Do(ctx, commands.KeepTabs{Tabs: keep}); err != nil && ctx.Err() == nil {
			s.log.Warn("keeping open tabs", "err", err)
		}
	}
}
