package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/web/internal/spa"
)

// Vite builds the app into dist/app; dist/.gitkeep keeps the embed valid before that.
const appDir = "dist/app"

//go:embed all:dist
var dist embed.FS

func main() {
	err := service.Run(os.Getenv, routes)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

func routes(_ func(string) string) (http.Handler, error) {
	app, err := fs.Sub(dist, appDir)
	if err != nil {
		return nil, fmt.Errorf("open the embedded app in %s: %w", appDir, err)
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	// The process is alive even before the app is built, so the health check doesn't need it.
	health.Register(router)
	spa.Register(router, app)

	return router, nil
}
