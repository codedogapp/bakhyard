package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/planner/internal/api"
)

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

// routes returns the handler of planner, which serves its API. With Copilot set up, it starts the
// Copilot CLI, which stops once ctx ends; stopping waits for it.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
	stopping *sync.WaitGroup,
) (http.Handler, error) {
	planner, err := startPlanner(ctx, getenv, readSecret, stopping)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, planner, slog.Default())

	return router, nil
}
