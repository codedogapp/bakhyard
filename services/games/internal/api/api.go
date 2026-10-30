// Package api serves the games I played lately over HTTP.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/games/internal/exophase"
	"github.com/codedogapp/bakhyard/services/games/internal/refresh"
)

// Recent is what GET /games/recent answers beside when the latest poll to succeed began and why
// the latest poll failed: the games I played in the last 14 days, the last played first.
type Recent struct {
	Games []Game `json:"games"`
}

// Game is a game in the answer: what Exophase says of it, and its cover.
type Game struct {
	exophase.Game

	// Cover is the address of the game's cover on IGDB, or "" when it has none or IGDB isn't set
	// up.
	Cover string `json:"cover,omitzero"`
}

// Register serves GET /games/recent on router, which answers what recent has found on Exophase,
// or 404 when Exophase isn't set up and recent is nil. Other methods get 405.
func Register(router chi.Router, recent *refresh.OnRequest[Recent]) {
	router.Get("/games/recent", func(w http.ResponseWriter, r *http.Request) {
		if recent == nil {
			http.Error(w, "Exophase is not set up", http.StatusNotFound)

			return
		}

		recent.ServeHTTP(w, r)
	})
}
