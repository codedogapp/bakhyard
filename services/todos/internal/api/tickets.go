package api

import (
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

		poller.ServeHTTP(w, r)
	}
}
