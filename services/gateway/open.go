package main

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// openRoutes returns the routes of the gateway built for the end-to-end tests (decision 107):
// those of routes, with no lock. No request needs the Tailscale identity or a session, and no
// route signs the user in or out, so it reads BAKHYARD_PEERS alone.
func openRoutes(getenv func(string) string) (http.Handler, error) {
	upstream, err := peers(getenv)
	if err != nil {
		return nil, err
	}

	open := func(next http.Handler) http.Handler { return next }
	noSigning := func(chi.Router) {
		// Nobody signs in or out of a gateway without locks.
	}
	guard := locks{identified: open, signedIn: open, signing: noSigning}

	return wire(upstream, guard, slog.Default()), nil
}
