package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/todos/internal/api"
	"github.com/codedogapp/bakhyard/services/todos/internal/attention"
	"github.com/codedogapp/bakhyard/services/todos/internal/migrations"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// databaseVar holds the URL of todos' database, without the password (process-compose.yaml).
const databaseVar = "BAKHYARD_DATABASE"

// databaseAccount is the Keychain account of the password of todos' database, and the name of
// its file (decisions 79 and 154).
const databaseAccount = "postgres-todos"

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

// routes returns todos' handler, which serves its API and its side of the attention contract.
// With Jira set up, it polls Jira in the background until ctx ends.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
) (http.Handler, error) {
	client, err := jira.FromEnv(getenv, readSecret)
	if err != nil {
		return nil, err
	}

	store, err := open(getenv, readSecret)
	if err != nil {
		return nil, err
	}

	logger := slog.Default()
	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)

	tickets := poll(ctx, client, store, logger)
	api.Register(router, store, tickets, logger)

	attentionServer := connect.NewServer()
	attentionv1connect.RegisterAttentionServiceHandler(attentionServer,
		attention.NewLister(store.List, store.Dismissed, tickets))
	connecthttp.Mount(router, attentionServer)

	return router, nil
}

// open opens todos' database, at the URL in BAKHYARD_DATABASE with the password that readSecret
// reads, runs its migrations and returns the store of the todos in it.
func open(getenv func(string) string, readSecret secrets.Reader) (*todo.Store, error) {
	address, err := env.Required(getenv, databaseVar)
	if err != nil {
		return nil, err
	}

	secret, err := readSecret(context.Background(), databaseAccount)
	if err != nil {
		return nil, fmt.Errorf("read the database password: %w", err)
	}

	pool, err := database.Open(context.Background(), address, secret, migrations.FS())
	if err != nil {
		return nil, err
	}

	return todo.NewStore(pool), nil
}
