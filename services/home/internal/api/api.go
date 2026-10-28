// Package api serves the Home screen as JSON over HTTP.
package api

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// Register serves GET /home on router for the user called name; other methods get 405. gather
// asks home's peers for what needs the user and how asking each went, and the items due before
// now(), read once they're in, are late.
func Register(
	router chi.Router,
	name string,
	now func() time.Time,
	gather func(context.Context) ([]screen.Item, []screen.SourceStatus),
) {
	router.Get("/home", func(w http.ResponseWriter, r *http.Request) {
		items, sources := gather(r.Context())
		home := screen.New(name, items, sources, now())

		w.Header().Set("Content-Type", "application/json")
		// The Home always encodes, so an error means the client has gone.
		_ = json.MarshalWrite(w, home)
	})
}
