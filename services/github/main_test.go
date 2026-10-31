package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/secrets"
)

// targets are the routes of the lists.
var targets = []string{"/github/mine", "/github/reviews", "/github/issues"}

func TestGitHubServesWithoutGitHub(t *testing.T) {
	router, err := routes(t.Context(), envOf(nil), secretsWith(nil))
	if err != nil {
		t.Fatal(err)
	}

	// A recorder keeps what the GET handler writes for HEAD; a server would drop it.
	for _, target := range targets {
		for method, code := range map[string]int{
			http.MethodGet:  http.StatusNotFound,
			http.MethodHead: http.StatusNotFound,
			http.MethodPost: http.StatusMethodNotAllowed,
		} {
			rec := serve(t, router, method, target)

			want := map[int]string{http.StatusNotFound: "GitHub is not set up\n"}[code]
			if rec.Code != code || rec.Body.String() != want {
				t.Errorf("%s %s: got %d %q, want %d %q", method, target, rec.Code, rec.Body,
					code, want)
			}
		}
	}

	rec := serve(t, router, http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("/healthz: got %d %q, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body)
	}
}

func TestGitHubSearchesEachList(t *testing.T) {
	address := githubServer(t, time.Time{})

	router, err := routes(t.Context(), envOf(map[string]string{githubAPIVar: address}),
		secretsWith(map[string]string{"github-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	for target, repo := range map[string]string{
		"/github/mine": "acme/web", "/github/reviews": "acme/api", "/github/issues": "acme/docs",
	} {
		rec := serve(t, router, http.MethodGet, target)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"repo":"`+repo+`"`) {
			t.Errorf("%s: got %d %q, want an item in %s", target, rec.Code, rec.Body, repo)
		}
	}
}

func TestGitHubStopsSearchingWhenItStops(t *testing.T) {
	stopped, stop := context.WithCancel(t.Context())
	stop()

	// Nothing listens on port 1, so a search that went ahead would fail.
	router, err := routes(stopped, envOf(map[string]string{githubAPIVar: "http://127.0.0.1:1"}),
		secretsWith(map[string]string{"github-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range targets {
		rec := serve(t, router, http.MethodGet, target)
		if rec.Code != http.StatusOK || rec.Body.String() != "{}" {
			t.Errorf("%s: got %d %s, want 200 {}: no search, and no failure", target, rec.Code,
				rec.Body)
		}
	}
}

// serve returns router's answer to a request for target with method.
func serve(t *testing.T, router http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
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
