package phone

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// events streams "changed" to the page whenever a question or a session changes, so that it reads
// the state again. It lasts until the page leaves or the service stops.
func (d Door) events(stopping context.Context, w http.ResponseWriter, r *http.Request) {
	flusher := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")

	// Subscribe before the first write, so that no change slips between the page's state and the
	// stream.
	changes, end := d.Changes.Subscribe()
	defer end()

	heartbeat := time.NewTicker(d.Heartbeat)
	defer heartbeat.Stop()

	if !send(w, flusher, ": open\n\n") {
		return
	}

	for {
		var message string

		select {
		case <-changes:
			message = "event: changed\ndata: {}\n\n"
		case <-heartbeat.C:
			message = ": still here\n\n"
		case <-r.Context().Done():
			return
		case <-stopping.Done():
			return
		}

		if !send(w, flusher, message) {
			return
		}
	}
}

// send writes message and flushes it, and reports whether the page is still there.
func send(w http.ResponseWriter, flusher *http.ResponseController, message string) bool {
	if _, err := fmt.Fprint(w, message); err != nil {
		return false
	}

	return flusher.Flush() == nil
}
