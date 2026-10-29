package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1/timev1connect"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/time/internal/api"
	"github.com/codedogapp/bakhyard/services/time/internal/attention"
	"github.com/codedogapp/bakhyard/services/time/internal/week"
)

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

// routes returns time's handler, which serves its API and its sides of the time and attention
// contracts. With Jira set up, it polls Jira for the user's worklogs in the background until ctx
// ends, and adds worklogs with the same client.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
) (http.Handler, error) {
	client, err := jira.FromEnv(getenv, readSecret)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)

	// The days of the reports, and so the items' due times, follow the local zone: the Mac's, or
	// the one that the container's TZ names.
	loc := time.Local
	backends := poll(ctx, client, loc, slog.Default())
	api.Register(router, backends, loc, slog.Default())

	server := connect.NewServer()
	timev1connect.RegisterTimeServiceHandler(server, week.NewServer(backends.Reports))
	attentionv1connect.RegisterAttentionServiceHandler(server,
		attention.NewLister(backends.Reports, loc))
	connecthttp.Mount(router, server)

	return router, nil
}
