package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestTimeServesWithoutJira(t *testing.T) {
	router, err := routes(t.Context(), envOf(nil), secretsWith(nil))
	if err != nil {
		t.Fatal(err)
	}

	// A recorder keeps what the GET handler writes for HEAD; a server would drop it.
	for _, test := range []struct {
		method, target string
		code           int
		body           string
	}{
		{http.MethodGet, "/time/report", http.StatusNotFound, "Jira is not set up\n"},
		{http.MethodHead, "/time/report", http.StatusNotFound, "Jira is not set up\n"},
		{http.MethodPost, "/time/report", http.StatusMethodNotAllowed, ""},
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
