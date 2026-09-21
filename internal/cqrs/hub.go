package cqrs

import "sync"

// Hub tracks open render streams and wakes them after commits.
//
// Every subscription owns a channel with capacity one. Notifications are
// non-blocking sends, so a slow stream coalesces any number of wake-ups into
// a single re-render of the latest state (frame dropping).
type Hub struct {
	mu   sync.Mutex
	subs map[*Sub]struct{}
}

type Sub struct {
	SID   string
	TabID string
	C     chan struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[*Sub]struct{}{}}
}

func (h *Hub) Subscribe(sid, tabID string) *Sub {
	s := &Sub{SID: sid, TabID: tabID, C: make(chan struct{}, 1)}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s
}

func (h *Hub) Unsubscribe(s *Sub) {
	h.mu.Lock()
	delete(h.subs, s)
	h.mu.Unlock()
}

// Notify wakes the streams of the given sessions; with all set, every stream.
func (h *Hub) Notify(all bool, sids map[string]struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		if !all {
			if _, ok := sids[s.SID]; !ok {
				continue
			}
		}
		select {
		case s.C <- struct{}{}:
		default:
		}
	}
}

// NotifyAll wakes every stream, e.g. after a dev reload of the catalog.
func (h *Hub) NotifyAll() { h.Notify(true, nil) }

// Len returns the number of open streams.
func (h *Hub) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}
