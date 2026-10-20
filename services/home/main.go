package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/server"
	"github.com/codedogapp/bakhyard/services/home/internal/api"
	"github.com/codedogapp/bakhyard/services/home/internal/health"
)

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

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, os.Getenv("BAKHYARD_NAME"), time.Now)

	return server.Run(ctx, addr, router)
}
