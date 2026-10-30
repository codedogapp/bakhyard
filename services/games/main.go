package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/games/internal/api"
)

func main() {
	// Exophase's API, where the games list comes from: the same for everyone, so not a setting
	// (docs/plan.md decision 300).
	exophaseAPI := &url.URL{Scheme: "https", Host: "api.exophase.com"}
	served := func(ctx context.Context, getenv func(string) string) (http.Handler, error) {
		return routes(ctx, getenv, exophaseAPI)
	}

	err := service.Run(os.Getenv, served)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

// routes returns the handler of games, which serves its API. With Exophase set up, it lists the
// games I played lately from exophaseAPI when a request finds them stale, until ctx ends.
func routes(
	ctx context.Context,
	getenv func(string) string,
	exophaseAPI *url.URL,
) (http.Handler, error) {
	client, err := exophaseClient(getenv, exophaseAPI)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, poll(ctx, client, slog.Default()))

	return router, nil
}
