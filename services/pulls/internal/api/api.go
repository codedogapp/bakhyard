// Package api serves the user's pull requests over HTTP.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// Register serves the user's open pull requests on router: GET /pulls/mine answers what mine
// has found on Bitbucket, nil when Bitbucket isn't set up. Other methods get 405.
func Register(router chi.Router, mine *bitbucket.Poller) {
	router.Get("/pulls/mine", listMine(mine))
}

// listMine returns the handler of GET /pulls/mine, which answers what poller has found, once its
// first poll has ended. Without Bitbucket, poller is nil and the client gets a 404.
func listMine(poller *bitbucket.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if poller == nil {
			http.Error(w, "Bitbucket is not set up", http.StatusNotFound)

			return
		}

		poller.ServeHTTP(w, r)
	}
}
