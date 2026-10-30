// Package api serves the user's lists from GitHub over HTTP.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// Register serves the lists on router: GET /github/mine answers what mine has found on GitHub,
// GET /github/reviews what reviews has found, and GET /github/issues what issues has found, each
// nil when GitHub isn't set up. Other methods get 405.
func Register(router chi.Router, mine, reviews, issues *github.Poller) {
	router.Get("/github/mine", list(mine))
	router.Get("/github/reviews", list(reviews))
	router.Get("/github/issues", list(issues))
}

// list returns the handler of a GET that answers what poller has found, once its first search has
// ended. Without GitHub, poller is nil and the client gets a 404.
func list(poller *github.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if poller == nil {
			http.Error(w, "GitHub is not set up", http.StatusNotFound)

			return
		}

		poller.ServeHTTP(w, r)
	}
}
