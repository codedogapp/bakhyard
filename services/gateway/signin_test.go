package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// testSecret is the client secret in the test gateway's Keychain.
const testSecret = "client-secret"

func TestTheGatewaySendsTheUserToGoogle(t *testing.T) {
	rec := httptest.NewRecorder()
	gateway(
		t,
		"http://127.0.0.1:2",
	).ServeHTTP(rec, signedIn(t, http.MethodGet, "/auth/google/login"))

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
	router := gateway(t, "http://127.0.0.1:2")

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
		func(context.Context, string, string) (string, error) { return "", locked },
	)
	if !errors.Is(err, locked) || err.Error() != "read the Google client secret: "+locked.Error() {
		t.Errorf("got %v, want the error read the Google client secret: %v", err, locked)
	}
}

// keychainWith returns a fake Keychain reader that has only items, by account, under the
// service bakhyard.
func keychainWith(items map[string]string) passwordReader {
	return func(_ context.Context, service, account string) (string, error) {
		secret, ok := items[account]
		if service != "bakhyard" || !ok {
			return "", fmt.Errorf("no Keychain item %s/%s", service, account)
		}

		return secret, nil
	}
}
