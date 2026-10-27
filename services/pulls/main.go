package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/pulls/internal/api"
)

// secretService names the login Keychain items that hold the service's secrets (decision 79).
const secretService = "bakhyard"

// passwordReader reads a password from the login Keychain: keychain.Reader.Password, or a fake
// in tests.
type passwordReader func(ctx context.Context, serviceName, account string) (string, error)

func main() {
	password := keychain.NewReader(keychain.Security).Password
	served := func(ctx context.Context, getenv func(string) string) (http.Handler, error) {
		return routes(ctx, getenv, password)
	}

	err := service.Run(os.Getenv, served)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

// routes returns the handler of pulls. With Bitbucket set up, it polls Bitbucket in the
// background until ctx ends.
func routes(
	ctx context.Context,
	getenv func(string) string,
	password passwordReader,
) (http.Handler, error) {
	client, err := bitbucketClient(getenv, password)
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
