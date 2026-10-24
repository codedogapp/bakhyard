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
	files, err := scan(getenv)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, files)

	return router, nil
}

// scan lists the files of the vault in the folder that BAKHYARD_VAULT names.
func scan(getenv func(string) string) ([]string, error) {
	folder, err := env.Required(getenv, vaultVar)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil, fmt.Errorf("open the vault: %w", err)
	}
	// Nothing was written through root, so closing it can't lose anything.
	defer func() { _ = root.Close() }()

	return vault.Scan(root.FS())
}
