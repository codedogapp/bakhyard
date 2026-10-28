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
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/notes/internal/api"
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

func routes(_ context.Context, getenv func(string) string) (http.Handler, error) {
	folder, err := env.Required(getenv, vaultVar)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)

	err = api.Register(router, folder, slog.Default())
	if err != nil {
		return nil, err
	}

	return router, nil
}
