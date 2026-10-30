package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/planner/internal/api"
)

// heartbeat is how often a plan's event stream says it is still open: under the idle timeouts of
// the proxies in between.
const heartbeat = 25 * time.Second

func main() {
	readSecret := secrets.New(os.Getenv, keychain.NewReader(keychain.Security).Password)

	var stopping sync.WaitGroup

	served := func(ctx context.Context, getenv func(string) string) (http.Handler, error) {
		return routes(ctx, getenv, readSecret, &stopping)
	}

	err := service.Run(os.Getenv, served)
	// Run has ended its context, so the Copilot CLI is stopping.
	stopping.Wait()

	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

// routes returns the handler of planner, which serves its API and saves plans through notes when
// BAKHYARD_PEERS lists it. With Copilot set up, it starts the Copilot CLI, which stops once ctx
// ends; stopping waits for it. The event streams end with ctx too.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
	stopping *sync.WaitGroup,
) (http.Handler, error) {
	// Read the peers first, so a mistake there doesn't start the CLI.
	notes, err := vaultClient(getenv)
	if err != nil {
		return nil, err
	}

	planner, err := startPlanner(ctx, getenv, readSecret, stopping)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)

	door := api.Door{Planner: planner, Vault: notes, Logger: slog.Default(), Heartbeat: heartbeat}
	door.Register(ctx, router)

	return router, nil
}
