package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/notes/internal/api"
	"github.com/codedogapp/bakhyard/services/notes/internal/vault"
)

// vaultVar names the vault's folder.
const vaultVar = "BAKHYARD_VAULT"

func main() {
	err := service.Run(os.Getenv, routes)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

func routes(getenv func(string) string) (http.Handler, error) {
	root, err := open(getenv)
	if err != nil {
		return nil, err
	}

	files, err := vault.Scan(root.FS())
	if err != nil {
		// Nothing was written through root, so closing it can't lose anything.
		_ = root.Close()

		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, root, files, slog.Default())

	return router, nil
}

// open opens the vault, in the folder that BAKHYARD_VAULT names, as the root that notes reads every
// file through while it runs.
func open(getenv func(string) string) (*os.Root, error) {
	folder, err := env.Required(getenv, vaultVar)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil, fmt.Errorf("open the vault: %w", err)
	}

	return root, nil
}
