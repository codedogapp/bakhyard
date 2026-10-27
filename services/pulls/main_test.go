package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPullsServesWithoutBitbucket(t *testing.T) {
	router, err := routes(t.Context(), envOf(nil), keychainWith(nil))
	if err != nil {
		t.Fatal(err)
	}

	// A recorder keeps what the GET handler writes for HEAD; a server would drop it.
	for _, test := range []struct {
		method, target string
		code           int
		body           string
	}{
		{http.MethodGet, "/pulls/mine", http.StatusNotFound, "Bitbucket is not set up\n"},
		{http.MethodHead, "/pulls/mine", http.StatusNotFound, "Bitbucket is not set up\n"},
		{http.MethodPost, "/pulls/mine", http.StatusMethodNotAllowed, ""},
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
