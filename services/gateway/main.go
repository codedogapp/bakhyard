package main

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/server"
	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
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

	home, err := env.PeerAddress(os.Getenv, "home")
	if err != nil {
		return err
	}

	web, err := env.PeerAddress(os.Getenv, "web")
	if err != nil {
		return err
	}

	logger := slog.Default()
	router := chi.NewRouter()
	proxy.Register(router, "home", &url.URL{Scheme: "http", Host: home}, logger)
	proxy.RegisterFallback(router, "web", &url.URL{Scheme: "http", Host: web}, logger)

	return server.Run(ctx, addr, router)
}
