package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/pulls/internal/api"
	"github.com/codedogapp/bakhyard/services/pulls/internal/attention"
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

// routes returns the handler of pulls, which serves its API and its side of the attention
// contract. With Bitbucket set up, it polls Bitbucket in the background until ctx ends.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
) (http.Handler, error) {
	client, err := bitbucketClient(getenv, readSecret)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)

	mine, reviews := poll(ctx, client, slog.Default())
	api.Register(router, mine, reviews)

	attentionServer := connect.NewServer()
	attentionv1connect.RegisterAttentionServiceHandler(attentionServer, attention.NewLister(mine))
	connecthttp.Mount(router, attentionServer)

	return router, nil
}
