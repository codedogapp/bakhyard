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
	"github.com/codedogapp/bakhyard/internal/tokenexpiry"
	"github.com/codedogapp/bakhyard/services/github/internal/api"
	"github.com/codedogapp/bakhyard/services/github/internal/attention"
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

// routes returns the handler of github, which serves its API and its side of the attention
// contract, with a reminder to renew the GitHub token before it expires. With GitHub set up, it
// searches GitHub, and asks when the token expires, in the background until ctx ends.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
) (http.Handler, error) {
	client, err := githubClient(getenv, readSecret)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)

	logger := slog.Default()
	mine, reviews, issues := poll(ctx, client, logger)
	api.Register(router, mine, reviews, issues)

	token := githubToken(watchToken(ctx, client, logger))
	attentionServer := connect.NewServer()
	attentionv1connect.RegisterAttentionServiceHandler(attentionServer,
		tokenexpiry.NewLister(attention.NewLister(mine, reviews, issues), token))
	connecthttp.Mount(router, attentionServer)

	return router, nil
}
