package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/home/internal/api"
	"github.com/codedogapp/bakhyard/services/home/internal/feed"
	"github.com/codedogapp/bakhyard/services/home/internal/week"
)

func main() {
	err := service.Run(os.Getenv, routes)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

func routes(_ context.Context, getenv func(string) string) (http.Handler, error) {
	peers, err := env.Peers(getenv)
	if err != nil {
		return nil, err
	}

	homeFeed := feed.New(peers, slog.Default())
	homeWeek := week.New(peers, slog.Default())

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, getenv("BAKHYARD_NAME"), time.Now,
		api.Peers{Items: homeFeed.Gather, Week: homeWeek.Ask})

	return router, nil
}
