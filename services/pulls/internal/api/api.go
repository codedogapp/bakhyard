// Package api serves the user's pull requests over HTTP.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// Register serves the open pull requests on router: GET /pulls/mine answers what mine has found
// on Bitbucket, and GET /pulls/reviews what reviews has found, each nil when Bitbucket isn't set
// up. Other methods get 405.
func Register(router chi.Router, mine, reviews *bitbucket.Poller) {
	router.Get("/pulls/mine", listPulls(mine))
	router.Get("/pulls/reviews", listPulls(reviews))
}

// listPulls returns the handler of a GET that answers what poller has found, once its first poll
// has ended. Without Bitbucket, poller is nil and the client gets a 404.
func listPulls(poller *bitbucket.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if poller == nil {
			http.Error(w, "Bitbucket is not set up", http.StatusNotFound)

			return
		}

		poller.ServeHTTP(w, r)
	}
}
