package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/codedogapp/bakhyard/internal/secrets"
)

// testSecret is the client secret in the test gateway's secrets.
const testSecret = "client-secret"

func TestTheGatewaySendsTheUserToGoogle(t *testing.T) {
	router, _ := gateway(t, "http://127.0.0.1:2")

	// Signing in needs no session.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, signedIn(t, http.MethodGet, "/auth/google/login"))

	target, err := url.Parse(rec.Header().Get("Location"))
	if rec.Code != http.StatusFound || err != nil || target.Host != "accounts.google.com" {
		t.Fatalf("got %d to %q, want 302 to Google", rec.Code, rec.Header().Get("Location"))
	}

	query := target.Query()
	if query.Get("client_id") != "client" ||
		query.Get("redirect_uri") != "https://"+testHost+"/auth/google/callback" {
		t.Errorf("got the client %q and the redirect URL %q, want the gateway's",
			query.Get("client_id"), query.Get("redirect_uri"))
	}
}

func TestTheSignInRoutesServeGETAndHEADOnly(t *testing.T) {
	router, _ := gateway(t, "http://127.0.0.1:2")

	// The routes' 405 comes after the lock, which answers 403 without a login.
	for method, want := range map[string]int{
		http.MethodHead: http.StatusFound,
		http.MethodPost: http.StatusMethodNotAllowed,
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, signedIn(t, method, "/auth/google/login"))

		if rec.Code != want {
			t.Errorf("%s /auth/google/login: got %d, want %d", method, rec.Code, want)
		}
	}
}

func TestTheGatewayNeedsTheClientSecret(t *testing.T) {
	vars, _ := gatewayEnv(t, "http://127.0.0.1:2")
	locked := errors.New("the Keychain is locked")

	_, err := routes(
		func(key string) string { return vars[key] },
		func(context.Context, string) (string, error) { return "", locked },
	)
	if !errors.Is(err, locked) || err.Error() != "read the Google client secret: "+locked.Error() {
		t.Errorf("got %v, want the error read the Google client secret: %v", err, locked)
	}
}

// secretsWith returns a fake secrets reader that has only items, by name.
func secretsWith(items map[string]string) secrets.Reader {
	return func(_ context.Context, name string) (string, error) {
		secret, ok := items[name]
		if !ok {
			return "", fmt.Errorf("no secret %s", name)
		}

		return secret, nil
	}
}
