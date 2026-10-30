package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestPlannerServesWithoutCopilot(t *testing.T) {
	var stopping sync.WaitGroup

	// No home, token or CLI: none of them is needed without the host.
	router, err := routes(t.Context(), envOf(nil), secretsWith(nil), &stopping)
	if err != nil {
		t.Fatal(err)
	}

	// A recorder keeps what the GET handler writes for HEAD; a server would drop it.
	for _, route := range []struct{ method, target string }{
		{http.MethodGet, "/planner/plans/x"},
		{http.MethodHead, "/planner/plans/x"},
		{http.MethodPost, "/planner/plans"},
	} {
		rec := serve(t, router, route.method, route.target)
		if rec.Code != http.StatusNotFound || rec.Body.String() != "Copilot is not set up\n" {
			t.Errorf("%s %s: got %d %q, want 404 Copilot is not set up", route.method,
				route.target, rec.Code, rec.Body)
		}
	}

	rec := serve(t, router, http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("/healthz: got %d %q, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body)
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
