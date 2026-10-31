package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

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
	"github.com/codedogapp/bakhyard/services/planner/internal/api"
)

// heartbeat is how often a plan's event stream says it is still open: under the idle timeouts of
// the proxies in between.
const heartbeat = 25 * time.Second

func main() {
	readSecret := secrets.New(os.Getenv, keychain.NewReader(keychain.Security).Password)

	var stopping sync.WaitGroup

	served := func(ctx context.Context, getenv func(string) string) (http.Handler, error) {
		return routes(ctx, getenv, readSecret, &stopping)
	}

	err := service.Run(os.Getenv, served)
	// Run has ended its context, so the Copilot CLI is stopping.
	stopping.Wait()

	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

// routes returns the handler of planner, which serves its API, saves plans through notes when
// BAKHYARD_PEERS lists it, and serves its side of the attention contract: a reminder to renew the
// Copilot token before it expires. With Copilot set up, it starts the Copilot CLI, which stops once
// ctx ends, and stopping waits for it; it also asks when the token expires, in the background until
// ctx ends. The event streams end with ctx too.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
	stopping *sync.WaitGroup,
) (http.Handler, error) {
	// Read every setting first, so a mistake in one doesn't start the CLI.
	notes, err := vaultClient(getenv)
	if err != nil {
		return nil, err
	}

	settings, err := copilotSettings(ctx, getenv, readSecret)
	if err != nil {
		return nil, err
	}

	githubAPI, err := copilotAPI(getenv, settings)
	if err != nil {
		return nil, err
	}

	planner, err := startPlanner(ctx, settings, stopping)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)

	logger := slog.Default()
	door := api.Door{Planner: planner, Vault: notes, Logger: logger, Heartbeat: heartbeat}
	door.Register(ctx, router)

	attentionServer := connect.NewServer()
	attentionv1connect.RegisterAttentionServiceHandler(attentionServer,
		tokenexpiry.NewLister(nil, copilotToken(ctx, githubAPI, logger)))
	connecthttp.Mount(router, attentionServer)

	return router, nil
}
