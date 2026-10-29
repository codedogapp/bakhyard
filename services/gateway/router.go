package main

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
	"github.com/codedogapp/bakhyard/services/gateway/internal/secure"
	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
)

// locks guard the gateway's routes. identified lets in only the user's Tailscale identity,
// signedIn only a request with a live session, and signing adds the routes that sign the user in
// and out.
type locks struct {
	identified, signedIn func(http.Handler) http.Handler
	signing              func(chi.Router)
}

// wire returns the gateway's router, which forwards to upstream behind guard.
func wire(upstream upstreams, guard locks, logger *slog.Logger) http.Handler {
	router := chi.NewRouter()
	// Every response gets the security headers, refusals included (decision 95). Then refuse what
	// other sites' pages send here with a method other than GET, HEAD or OPTIONS: the browser adds
	// the user's cookies (decision 88). Then answer HEAD wherever GET is served.
	router.Use(secure.Headers, http.NewCrossOriginProtection().Handler, middleware.GetHead)
	// The gateway's own health check, so /healthz never reaches web (decision 59). It needs no
	// login: process-compose's probe calls the gateway directly.
	health.Register(router)
	// Signing in and out needs the Tailscale identity only.
	router.Group(func(signingIn chi.Router) {
		signingIn.Use(guard.identified)
		guard.signing(signingIn)
	})
	// The API needs a session as well (decision 86).
	router.Group(func(api chi.Router) {
		api.Use(guard.identified, guard.signedIn, session.Hide)
		proxy.Register(api, "home", upstream.home, logger)
		proxy.Register(api, "notes", upstream.notes, logger)
		proxy.Register(api, "notify", upstream.notify, logger)
		proxy.Register(api, "nudge", upstream.nudge, logger)
		proxy.Register(api, "pulls", upstream.pulls, logger)
		proxy.Register(api, "time", upstream.time, logger)
		proxy.Register(api, "todos", upstream.todos, logger)
		proxy.RegisterUnclaimed(api)
	})
	// Agents ask through the tailnet with the Tailscale identity only: they have no Google session
	// (decision 164).
	router.Group(func(agents chi.Router) {
		agents.Use(guard.identified, session.Hide)
		proxy.RegisterAgent(agents, "nudge", upstream.nudge, logger)
	})
	// The app's files hold no data, so they need the Tailscale identity only. This group is the
	// root router's child, so that its web fallback, chi's NotFound handler, reaches the root router.
	router.Group(func(app chi.Router) {
		app.Use(guard.identified, session.Hide)
		proxy.RegisterFallback(app, "web", upstream.web, logger)
	})

	return router
}
