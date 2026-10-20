package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
)

func main() {
	err := service.Run(os.Getenv, routes)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

func routes(getenv func(string) string) (http.Handler, error) {
	home, err := env.PeerAddress(getenv, "home")
	if err != nil {
		return nil, err
	}

	web, err := env.PeerAddress(getenv, "web")
	if err != nil {
		return nil, err
	}

	logger := slog.Default()
	router := chi.NewRouter()
	proxy.Register(router, "home", &url.URL{Scheme: "http", Host: home}, logger)
	proxy.RegisterFallback(router, "web", &url.URL{Scheme: "http", Host: web}, logger)

	return router, nil
}
