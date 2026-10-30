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

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/notify/internal/api"
	"github.com/codedogapp/bakhyard/services/notify/internal/history"
	"github.com/codedogapp/bakhyard/services/notify/internal/migrations"
	"github.com/codedogapp/bakhyard/services/notify/internal/push"
	"github.com/codedogapp/bakhyard/services/notify/internal/subscription"
	"github.com/codedogapp/bakhyard/services/notify/internal/vapid"
)

// databaseVar holds the URL of notify's database, without the password (process-compose.yaml).
const databaseVar = "BAKHYARD_DATABASE"

const (
	// databaseAccount is the Keychain account of the password of notify's database, and the name
	// of its file.
	databaseAccount = "postgres-notify"
	// vapidAccount is the account of the app's VAPID private key, base64url.
	vapidAccount = "vapid-key"
	// pushTimeout bounds one push to a push service.
	pushTimeout = 15 * time.Second
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

// routes returns notify's handler.
func routes(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
) (http.Handler, error) {
	stored, err := readSecret(ctx, vapidAccount)
	if err != nil {
		return nil, fmt.Errorf("read the VAPID key: %w", err)
	}

	key, err := vapid.Parse(stored)
	if err != nil {
		return nil, err
	}

	address, err := env.Required(getenv, databaseVar)
	if err != nil {
		return nil, err
	}

	password, err := readSecret(ctx, databaseAccount)
	if err != nil {
		return nil, fmt.Errorf("read the database password: %w", err)
	}

	pool, err := database.Open(ctx, address, password, migrations.FS())
	if err != nil {
		return nil, err
	}

	logger := slog.Default()
	devices := subscription.NewStore(pool)
	sender := &push.Sender{
		Key: key, Devices: devices, Client: &http.Client{Timeout: pushTimeout}, Logger: logger,
	}

	router := chi.NewRouter()
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, key.Public, api.Backends{
		Devices: devices, Sender: sender, Alerts: history.NewStore(pool),
	}, logger)

	return router, nil
}
