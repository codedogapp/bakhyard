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
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/games/internal/api"
)

func main() {
	// The APIs that games asks: the same for everyone, so not settings (docs/plan.md decision 300).
	addresses := apis{
		exophase: &url.URL{Scheme: "https", Host: "api.exophase.com"},
		twitch:   &url.URL{Scheme: "https", Host: "id.twitch.tv"},
		igdb:     &url.URL{Scheme: "https", Host: "api.igdb.com"},
	}
	readSecret := secrets.New(os.Getenv, keychain.NewReader(keychain.Security).Password)
	served := func(ctx context.Context, getenv func(string) string) (http.Handler, error) {
		return routes(ctx, getenv, readSecret, addresses)
	}

	err := service.Run(os.Getenv, served)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

// apis are the addresses of the APIs that games asks.
type apis struct {
	// exophase is Exophase's, where the games list comes from.
	exophase *url.URL
	// twitch is Twitch's OAuth server, which gives the tokens that IGDB takes.
	twitch *url.URL
	// igdb is IGDB's, where the covers come from.
	igdb *url.URL
}

// routes returns the handler of games, which serves its API. With Exophase set up, it lists the
// games I played lately when a request finds them stale, until ctx ends, and with a Twitch app set
// up, whose secret readSecret reads, it finds their covers on IGDB.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
	addresses apis,
) (http.Handler, error) {
	client, err := exophaseClient(getenv, addresses.exophase)
	if err != nil {
		return nil, err
	}

	cache, err := coverCache(ctx, getenv, readSecret, addresses)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, poll(ctx, client, cache, slog.Default()))

	return router, nil
}
