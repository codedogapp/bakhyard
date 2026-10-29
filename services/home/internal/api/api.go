// Package api serves the Home screen as JSON over HTTP.
package api

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// Peers is how Register asks home's peers what the Home shows.
type Peers struct {
	// Items asks for what needs the user, and how asking each peer went.
	Items func(context.Context) ([]screen.Item, []screen.SourceStatus)
	// Week asks time for this week's hours, and says how asking went.
	Week func(context.Context) (*screen.Week, screen.State)
}

// Register serves GET /home on router for the user called name; other methods get 405. It asks
// peers for the items and the week at once, and the items due before now(), read once both are
// in, are late.
func Register(router chi.Router, name string, now func() time.Time, peers Peers) {
	router.Get("/home", func(w http.ResponseWriter, r *http.Request) {
		var (
			asking    sync.WaitGroup
			week      *screen.Week
			weekState screen.State
		)

		asking.Go(func() { week, weekState = peers.Week(r.Context()) })

		items, sources := peers.Items(r.Context())

		asking.Wait()

		home := screen.New(name, items, sources, now())
		home.Week, home.WeekState = week, weekState

		w.Header().Set("Content-Type", "application/json")
		// The Home always encodes, so an error means the client has gone.
		_ = json.MarshalWrite(w, home)
	})
}
