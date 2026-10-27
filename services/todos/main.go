package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/todos/internal/api"
	"github.com/codedogapp/bakhyard/services/todos/internal/migrations"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// databaseVar holds the URL of todos' database, without the password (process-compose.yaml).
const databaseVar = "BAKHYARD_DATABASE"

// The login Keychain item that holds the password of todos' database (decision 79).
const (
	secretService   = "bakhyard"
	databaseAccount = "postgres-todos"
)

// passwordReader reads a password from the login Keychain: keychain.Reader.Password, or a fake
// in tests.
type passwordReader func(ctx context.Context, serviceName, account string) (string, error)

func main() {
	keys := keychain.NewReader(keychain.Security)
	served := func(ctx context.Context, getenv func(string) string) (http.Handler, error) {
		return routes(ctx, getenv, keys.Password)
	}

	err := service.Run(os.Getenv, served)
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

// routes returns todos' handler. With Jira set up, it polls Jira in the background until ctx
// ends.
func routes(
	ctx context.Context,
	getenv func(string) string,
	password passwordReader,
) (http.Handler, error) {
	client, err := jiraClient(getenv, password)
	if err != nil {
		return nil, err
	}

	store, err := open(getenv, password)
	if err != nil {
		return nil, err
	}

	logger := slog.Default()
	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, store, poll(ctx, client, store, logger), logger)

	return router, nil
}

// open opens todos' database, at the URL in BAKHYARD_DATABASE with the password from the
// Keychain, runs its migrations and returns the store of the todos in it.
func open(getenv func(string) string, password passwordReader) (*todo.Store, error) {
	address, err := env.Required(getenv, databaseVar)
	if err != nil {
		return nil, err
	}

	secret, err := password(context.Background(), secretService, databaseAccount)
	if err != nil {
		return nil, fmt.Errorf("read the database password: %w", err)
	}

	pool, err := database.Open(context.Background(), address, secret, migrations.FS())
	if err != nil {
		return nil, err
	}

	return todo.NewStore(pool), nil
}
