package api

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
)

// ticketList is the body of GET /todos/tickets: the user's open Jira tickets.
type ticketList struct {
	Tickets []jira.Ticket `json:"tickets"`
}

// listTickets returns the handler of GET /todos/tickets, which searches Jira with client, nil
// when Jira isn't set up. Without Jira, the client gets a 404; when the search fails, a 502, and
// logger gets the cause.
func listTickets(client *jira.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			http.Error(w, "Jira is not set up", http.StatusNotFound)

			return
		}

		found, err := client.Search(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "list the tickets", "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		// Jira's JSON was valid UTF-8, so the tickets always encode: an error means the client
		// has gone.
		_ = json.MarshalWrite(w, ticketList{Tickets: found})
	}
}
