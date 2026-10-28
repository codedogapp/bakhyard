// Package change tells whoever watches nudge's state that it changed. It carries no content: a
// watcher reads the state again.
package change

import "sync"

// Hub tells its subscribers about changes. The zero value is a hub without subscribers, safe for
// use by several goroutines at once.
type Hub struct {
	mu       sync.Mutex
	watchers map[chan struct{}]struct{}
}

// Subscribe returns a channel that gets a signal after each change published from now on, and the
// function that ends the subscription. Changes that come faster than the subscriber reads them
// merge into one signal.
func (h *Hub) Subscribe() (<-chan struct{}, func()) {
	watcher := make(chan struct{}, 1)

	h.mu.Lock()
	if h.watchers == nil {
		h.watchers = map[chan struct{}]struct{}{}
	}

	h.watchers[watcher] = struct{}{}
	h.mu.Unlock()

	return watcher, func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		delete(h.watchers, watcher)
	}
}

// Publish signals every subscriber.
func (h *Hub) Publish() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for watcher := range h.watchers {
		select {
		case watcher <- struct{}{}:
		default:
		}
	}
}
