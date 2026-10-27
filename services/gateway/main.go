package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/keychain"
	"github.com/codedogapp/bakhyard/internal/service"
	"github.com/codedogapp/bakhyard/services/gateway/internal/migrations"
	"github.com/codedogapp/bakhyard/services/gateway/internal/pin"
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

	err := service.Run(os.Getenv, served(password))
	if err != nil {
		slog.Error("failed", "err", err)
		os.Exit(1)
	}
}

// served returns the routes this build serves: the real gateway's, or the open ones of the build
// for the end-to-end tests (decision 107), which it warns of.
func served(password passwordReader) service.Routes {
	if e2e {
		slog.Warn("authentication is off: this gateway is built for the end-to-end tests")

		return func(_ context.Context, getenv func(string) string) (http.Handler, error) {
			return openRoutes(getenv)
		}
	}

	return func(_ context.Context, getenv func(string) string) (http.Handler, error) {
		return routes(getenv, password)
	}
}

func routes(getenv func(string) string, password passwordReader) (http.Handler, error) {
	upstream, err := peers(getenv)
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

	sessions, pins, err := stores(getenv, password)
	if err != nil {
		return nil, err
	}

	logger := slog.Default()
	guard := locks{
		identified: tailscale.RequireLogin(login, host, logger),
		signedIn:   session.Require(sessions, logger),
		// Signing out takes POST alone, so that a link or a prefetch can't sign the user out.
		signing: func(router chi.Router) {
			signin.New(settings, sessions, pins, logger).Register(router)
			router.Method(http.MethodPost, "/auth/logout", session.SignOut(sessions, logger))
		},
	}

	return wire(upstream, guard, logger), nil
}

// upstreams are the URLs of the services that the gateway forwards to.
type upstreams struct {
	home, notes, pulls, todos, web *url.URL
}

// peers returns the URLs of home, notes, pulls, todos and web from BAKHYARD_PEERS, or an error
// that names every one of them it lacks.
func peers(getenv func(string) string) (upstreams, error) {
	home, homeErr := peer(getenv, "home")
	notes, notesErr := peer(getenv, "notes")
	pulls, pullsErr := peer(getenv, "pulls")
	todos, todosErr := peer(getenv, "todos")
	web, webErr := peer(getenv, "web")

	err := errors.Join(homeErr, notesErr, pullsErr, todosErr, webErr)
	if err != nil {
		return upstreams{}, err
	}

	return upstreams{home: home, notes: notes, pulls: pulls, todos: todos, web: web}, nil
}

// peer returns the URL of the peer name, from BAKHYARD_PEERS.
func peer(getenv func(string) string, name string) (*url.URL, error) {
	address, err := env.PeerAddress(getenv, name)
	if err != nil {
		return nil, err
	}

	return &url.URL{Scheme: "http", Host: address}, nil
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

// stores opens the gateway's database with the password from the Keychain, and returns the stores
// that keep the sessions and the pins in it.
func stores(
	getenv func(string) string, password passwordReader,
) (*session.Store, *pin.Store, error) {
	address, err := env.Required(getenv, databaseVar)
	if err != nil {
		return nil, nil, err
	}

	secret, err := password(context.Background(), secretService, databaseAccount)
	if err != nil {
		return nil, nil, fmt.Errorf("read the database password: %w", err)
	}

	pool, err := database.Open(context.Background(), address, secret, migrations.FS())
	if err != nil {
		return nil, nil, err
	}

	return session.NewStore(pool), pin.NewStore(pool), nil
}
