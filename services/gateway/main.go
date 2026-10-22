package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
	"github.com/codedogapp/bakhyard/services/gateway/internal/tailscale"
)

// The personal settings that let the user in (docs/plan.md §3 Auth). They come from
// mise.local.toml, never from a tracked file.
const (
	loginVar = "BAKHYARD_TAILSCALE_LOGIN"
	hostVar  = "BAKHYARD_HOST"
)

func main() {
	err := service.Run(os.Getenv, routes)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

func routes(getenv func(string) string) (http.Handler, error) {
	home, err := env.PeerAddress(getenv, "home")
	if err != nil {
		return nil, err
	}

	web, err := env.PeerAddress(getenv, "web")
	if err != nil {
		return nil, err
	}

	login, err := env.Required(getenv, loginVar)
	if err != nil {
		return nil, err
	}

	host, err := env.Required(getenv, hostVar)
	if err != nil {
		return nil, err
	}

	logger := slog.Default()
	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	// The gateway's own health check, so /healthz never reaches web (decision 59). It needs no
	// login: process-compose's probe calls the gateway directly.
	health.Register(router)
	router.Group(func(guarded chi.Router) {
		guarded.Use(tailscale.RequireLogin(login, host, logger))
		proxy.Register(guarded, "home", &url.URL{Scheme: "http", Host: home}, logger)
		proxy.RegisterFallback(guarded, "web", &url.URL{Scheme: "http", Host: web}, logger)
	})

	return router, nil
}
