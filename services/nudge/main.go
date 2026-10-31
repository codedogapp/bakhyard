package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/nudge/internal/agent"
	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
	"github.com/codedogapp/bakhyard/services/nudge/internal/migrations"
	"github.com/codedogapp/bakhyard/services/nudge/internal/phone"
	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
	"github.com/codedogapp/bakhyard/services/nudge/internal/session"
)

// databaseVar holds the URL of nudge's database, without the password (process-compose.yaml).
const databaseVar = "BAKHYARD_DATABASE"

const (
	// databaseAccount is the Keychain account of the password of nudge's database, and the name of
	// its file.
	databaseAccount = "postgres-nudge"
	// heartbeat is how often an event stream says it is still open: under the idle timeouts of
	// the proxies in between.
	heartbeat = 25 * time.Second
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

// routes returns nudge's handler. The requests that it holds open end when ctx does.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
) (http.Handler, error) {
	address, err := env.PeerAddress(getenv, "notify")
	if err != nil {
		return nil, err
	}

	pool, err := openDatabase(ctx, getenv, readSecret)
	if err != nil {
		return nil, err
	}

	logger := slog.Default()
	hub := &change.Hub{}
	questions := question.NewStore(pool, hub)
	sessions := session.NewRegistry(hub)
	pusher := notifyClient(address)

	router := chi.NewRouter()
	router.Use(middleware.GetHead)
	health.Register(router)
	agent.Door{
		Questions: questions, Sessions: sessions, Pusher: pusher, Logger: logger,
	}.Register(ctx, router)
	phone.Door{
		Questions: questions, Sessions: sessions, Changes: hub, Logger: logger,
		Heartbeat: heartbeat,
	}.Register(ctx, router)

	return router, nil
}

// openDatabase opens nudge's database, at the URL in BAKHYARD_DATABASE with the password that
// readSecret reads, and runs its migrations.
func openDatabase(
	ctx context.Context, getenv func(string) string, readSecret secrets.Reader,
) (*pgxpool.Pool, error) {
	address, err := env.Required(getenv, databaseVar)
	if err != nil {
		return nil, err
	}

	password, err := readSecret(ctx, databaseAccount)
	if err != nil {
		return nil, fmt.Errorf("read the database password: %w", err)
	}

	return database.Open(ctx, address, password, migrations.FS())
}
