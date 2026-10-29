// Package api serves the user's time report over HTTP.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// Register serves the time report on router: GET /time/report answers what reports has found in
// Jira, once its first poll has ended. Without Jira, reports is nil and the client gets a 404.
// Other methods get 405.
func Register(router chi.Router, reports *worklogs.Poller) {
	router.Get("/time/report", func(w http.ResponseWriter, r *http.Request) {
		if reports == nil {
			http.Error(w, "Jira is not set up", http.StatusNotFound)

			return
		}

		reports.ServeHTTP(w, r)
	})
}
