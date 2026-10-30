package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestGamesServesWithoutExophase(t *testing.T) {
	router, err := routes(t.Context(), envOf(nil), secretsWith(nil), exophaseAt(unasked()))
	if err != nil {
		t.Fatal(err)
	}

	// A recorder keeps what the GET handler writes for HEAD; a server would drop it.
	for _, test := range []struct {
		method, target string
		code           int
		body           string
	}{
		{http.MethodGet, "/games/recent", http.StatusNotFound, "Exophase is not set up\n"},
		{http.MethodHead, "/games/recent", http.StatusNotFound, "Exophase is not set up\n"},
		{http.MethodPost, "/games/recent", http.StatusMethodNotAllowed, ""},
		{http.MethodGet, "/healthz", http.StatusOK, `{"status":"ok"}`},
	} {
		req := httptest.NewRequestWithContext(t.Context(), test.method, test.target, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != test.code || rec.Body.String() != test.body {
			t.Errorf("%s %s: got %d %q, want %d %q", test.method, test.target, rec.Code,
				rec.Body, test.code, test.body)
		}
	}
}

func TestGamesDoesNotStartWithABadPlayer(t *testing.T) {
	router, err := routes(t.Context(), envOf(map[string]string{exophasePlayerVar: "me"}),
		secretsWith(nil), exophaseAt(unasked()))

	want := `BAKHYARD_EXOPHASE_PLAYER: "me" is not an Exophase profile id, which is digits only`
	if router != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want no handler and the error %q", router, err, want)
	}
}

// unasked returns an address of an API that the tests never ask.
func unasked() *url.URL {
	return &url.URL{Scheme: "https", Host: "api.unasked.example"}
}

// exophaseAt returns the addresses of the APIs, with Exophase's at address and the others
// unasked.
func exophaseAt(address *url.URL) apis {
	return apis{exophase: address, twitch: unasked(), igdb: unasked()}
}

// envOf returns an environment that holds only settings.
func envOf(settings map[string]string) func(string) string {
	return func(name string) string { return settings[name] }
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
