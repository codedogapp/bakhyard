package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/health"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/gateway/internal/database"
	"github.com/codedogapp/bakhyard/services/gateway/internal/keychain"
	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
	"github.com/codedogapp/bakhyard/services/gateway/internal/signin"
	"github.com/codedogapp/bakhyard/services/gateway/internal/tailscale"
)

// The personal settings that let the user in (docs/plan.md §3 Auth). They come from
// mise.local.toml, never from a tracked file.
const (
	loginVar    = "BAKHYARD_TAILSCALE_LOGIN"
	hostVar     = "BAKHYARD_HOST"
	clientIDVar = "BAKHYARD_GOOGLE_CLIENT_ID"
	emailVar    = "BAKHYARD_GOOGLE_EMAIL"
)

// databaseVar holds the URL of the gateway's database, without the password
// (process-compose.yaml).
const databaseVar = "BAKHYARD_DATABASE"

// The login Keychain items that hold the gateway's secrets (decision 79).
const (
	secretService       = "bakhyard"
	clientSecretAccount = "google-client-secret"
	databaseAccount     = "postgres-gateway"
)

// passwordReader reads a password from the login Keychain: keychain.Reader.Password, or a fake
// in tests.
type passwordReader func(ctx context.Context, serviceName, account string) (string, error)

func main() {
	password := keychain.NewReader(keychain.Security).Password

	err := service.Run(os.Getenv, func(getenv func(string) string) (http.Handler, error) {
		return routes(getenv, password)
	})
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

func routes(getenv func(string) string, password passwordReader) (http.Handler, error) {
	home, err := env.PeerAddress(getenv, "home")
	if err != nil {
		return nil, err
	}

	web, err := env.PeerAddress(getenv, "web")
	if err != nil {
		return nil, err
	}

	login, err := env.Required(getenv, loginVar)
	if err != nil {
		return nil, err
	}

	host, err := env.Required(getenv, hostVar)
	if err != nil {
		return nil, err
	}

	settings, err := googleSettings(getenv, host, password)
	if err != nil {
		return nil, err
	}

	sessions, err := sessionStore(getenv, password)
	if err != nil {
		return nil, err
	}

	logger := slog.Default()

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	// The gateway's own health check, so /healthz never reaches web (decision 59). It needs no
	// login: process-compose's probe calls the gateway directly.
	health.Register(router)
	router.Group(func(guarded chi.Router) {
		guarded.Use(tailscale.RequireLogin(login, host, logger))
		signin.New(settings, sessions, logger).Register(guarded)
		proxy.Register(guarded, "home", &url.URL{Scheme: "http", Host: home}, logger)
		proxy.RegisterFallback(guarded, "web", &url.URL{Scheme: "http", Host: web}, logger)
	})

	return router, nil
}

// googleSettings returns the settings of the flow that signs the user in with Google and sends
// them back to host. It reads the client secret with password.
func googleSettings(
	getenv func(string) string, host string, password passwordReader,
) (signin.Settings, error) {
	clientID, err := env.Required(getenv, clientIDVar)
	if err != nil {
		return signin.Settings{}, err
	}

	email, err := env.Required(getenv, emailVar)
	if err != nil {
		return signin.Settings{}, err
	}

	secret, err := password(context.Background(), secretService, clientSecretAccount)
	if err != nil {
		return signin.Settings{}, fmt.Errorf("read the Google client secret: %w", err)
	}

	return signin.Settings{
		Provider:     signin.Google(),
		ClientID:     clientID,
		ClientSecret: secret,
		Email:        email,
		Host:         host,
	}, nil
}

// sessionStore opens the gateway's database with the password from the Keychain, and returns the
// store that keeps the sessions in it.
func sessionStore(getenv func(string) string, password passwordReader) (*session.Store, error) {
	address, err := env.Required(getenv, databaseVar)
	if err != nil {
		return nil, err
	}

	secret, err := password(context.Background(), secretService, databaseAccount)
	if err != nil {
		return nil, fmt.Errorf("read the database password: %w", err)
	}

	pool, err := database.Open(context.Background(), address, secret)
	if err != nil {
		return nil, err
	}

	return session.NewStore(pool), nil
}
