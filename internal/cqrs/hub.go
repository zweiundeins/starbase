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
	Path  string // page the stream renders, for presence counts
	C     chan struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[*Sub]struct{}{}}
}

// Subscribe registers a stream. Other streams on the same path are woken,
// so presence counts ("N watching") stay live.
func (h *Hub) Subscribe(sid, tabID, path string) *Sub {
	s := &Sub{SID: sid, TabID: tabID, Path: path, C: make(chan struct{}, 1)}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.wakePath(path, s)
	h.mu.Unlock()
	return s
}

func (h *Hub) Unsubscribe(s *Sub) {
	h.mu.Lock()
	delete(h.subs, s)
	h.wakePath(s.Path, nil)
	h.mu.Unlock()
}

// wakePath wakes the streams on path except skip. Callers hold h.mu.
func (h *Hub) wakePath(path string, skip *Sub) {
	for o := range h.subs {
		if o != skip && o.Path == path {
			select {
			case o.C <- struct{}{}:
			default:
			}
		}
	}
}

// Count returns the number of open streams rendering path.
func (h *Hub) Count(path string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for s := range h.subs {
		if s.Path == path {
			n++
		}
	}
	return n
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
