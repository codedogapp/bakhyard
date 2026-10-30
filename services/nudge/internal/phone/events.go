package phone

import (
	"context"
	"net/http"
	"time"

	"github.com/codedogapp/bakhyard/internal/eventstream"
)

// events streams "changed" to the page whenever a question or a session changes, so that it reads
// the state again. It lasts until the page leaves or the service stops. HEAD gets the stream's
// headers alone: a stream would hold its connection, and the next request on it would wait.
func (d Door) events(stopping context.Context, w http.ResponseWriter, r *http.Request) {
	// Subscribe before the first write, so that no change slips between the page's state and the
	// stream.
	changes, end := d.Changes.Subscribe()
	defer end()

	heartbeat := time.NewTicker(d.Heartbeat)
	defer heartbeat.Stop()

	stream, err := eventstream.Open(w)
	if err != nil || r.Method == http.MethodHead {
		return
	}

	for err == nil {
		select {
		case <-changes:
			err = stream.Send("changed", struct{}{})
		case <-heartbeat.C:
			err = stream.Beat()
		case <-r.Context().Done():
			return
		case <-stopping.Done():
			return
		}
	}
}
