package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/server"
	"github.com/codedogapp/bakhyard/services/web/internal/spa"
)

// Vite builds the app into dist/app; dist/.gitkeep keeps the embed valid before that.
const appDir = "dist/app"

//go:embed all:dist
var dist embed.FS

func main() {
	err := run()
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr, err := env.ListenAddress(os.Getenv)
	if err != nil {
		return err
	}

	app, err := fs.Sub(dist, appDir)
	if err != nil {
		return fmt.Errorf("open the embedded app in %s: %w", appDir, err)
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	spa.Register(router, app)

	return server.Run(ctx, addr, router)
}
