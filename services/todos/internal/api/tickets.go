package api

import (
	"encoding/json/v2"
	"net/http"

	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
)

// listTickets returns the handler of GET /todos/tickets, which answers what poller has found in
// Jira, once its first poll has ended. Without Jira, poller is nil and the client gets a 404.
func listTickets(poller *jira.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if poller == nil {
			http.Error(w, "Jira is not set up", http.StatusNotFound)

			return
		}

		latest, err := poller.Latest(r.Context())
		if err != nil {
			// The client has gone before the first poll ended.
			return
		}

		w.Header().Set("Content-Type", "application/json")
		// Jira's JSON was valid UTF-8, and the poller makes each failure valid UTF-8 too, so the
		// snapshot always encodes: an error means the client has gone.
		_ = json.MarshalWrite(w, latest)
	}
}
