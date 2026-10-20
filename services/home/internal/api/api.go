// Package api serves the Home screen as JSON over HTTP.
package api

import (
	"encoding/json/v2"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/home/internal/demo"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// Register serves GET /home on router for the user called name; other methods get 405. Due times
// are relative to now().
func Register(router chi.Router, name string, now func() time.Time) {
	router.Get("/home", func(w http.ResponseWriter, _ *http.Request) {
		at := now()
		home := screen.New(name, demo.Items(at), demo.Week(), at)

		w.Header().Set("Content-Type", "application/json")
		// The Home always encodes, so an error means the client has gone.
		_ = json.MarshalWrite(w, home)
	})
}
