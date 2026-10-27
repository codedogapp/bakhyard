package poll

import (
	"encoding/json/v2"
	"net/http"
)

// ServeHTTP answers what the polls have found, as JSON, once the first poll has ended. A client
// that leaves before then gets nothing. What the look-up finds must encode as JSON; the poller
// makes each failure valid UTF-8 itself.
func (p *Poller[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	latest, err := p.Latest(r.Context())
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// The snapshot encodes, so an error means the client has gone.
	_ = json.MarshalWrite(w, latest)
}
