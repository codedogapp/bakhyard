// Package health answers a service's health check, GET /healthz, which says that the process is
// alive and serving (docs/plan.md §4.7). Every service registers it.
package health

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const body = `{"status":"ok"}`

// Register serves GET /healthz on router; other methods get 405.
func Register(router chi.Router) {
	router.Get("/healthz", healthz)
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// A failed write means the client has gone; there is no one left to tell.
	_, _ = io.WriteString(w, body)
}
