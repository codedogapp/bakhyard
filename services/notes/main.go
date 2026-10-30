package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/notes/internal/api"
)

// vaultVar names the vault's folder.
const vaultVar = "BAKHYARD_VAULT"

func main() {
	readSecret := secrets.New(os.Getenv, keychain.NewReader(keychain.Security).Password)
	served := func(ctx context.Context, getenv func(string) string) (http.Handler, error) {
		return routes(ctx, getenv, readSecret)
	}

	err := service.Run(os.Getenv, served)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

// routes returns the handler of notes, which serves the vault in its folder, saves its notes once
// the vault's repository is set, with the token that readSecret reads, and links to their history
// once the vault's history is set.
func routes(
	ctx context.Context, getenv func(string) string, readSecret secrets.Reader,
) (http.Handler, error) {
	folder, err := env.Required(getenv, vaultVar)
	if err != nil {
		return nil, err
	}

	history, err := vaultHistory(getenv)
	if err != nil {
		return nil, err
	}

	store, err := vaultStore(ctx, getenv, readSecret)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)

	vault := api.Vault{Folder: folder, Store: store, History: history}

	err = api.Register(router, vault, slog.Default())
	if err != nil {
		return nil, err
	}

	return router, nil
}
