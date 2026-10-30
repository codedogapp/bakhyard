package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestNotesServesTheVault(t *testing.T) {
	folder := t.TempDir()
	for _, name := range []string{"Inbox.md", ".obsidian/app.json"} {
		file := filepath.Join(folder, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	router, err := routes(t.Context(), envOf(map[string]string{vaultVar: folder}), noSecrets(t))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct{ method, target, want string }{
		{http.MethodGet, "/notes", `{"files":["Inbox.md"]}`},
		{http.MethodGet, "/notes/files/Inbox.md", "{}"},
		{http.MethodHead, "/notes/files/Inbox.md", ""},
		{http.MethodPost, "/notes/sync", `{"files":["Inbox.md"]}`},
		{http.MethodGet, "/healthz", `{"status":"ok"}`},
	} {
		req := httptest.NewRequestWithContext(t.Context(), test.method, test.target, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK || rec.Body.String() != test.want {
			t.Errorf("%s %s: got %d %q, want 200 %q", test.method, test.target, rec.Code,
				rec.Body.String(), test.want)
		}
	}
}

func TestNotesNeedsItsVault(t *testing.T) {
	folder := t.TempDir()

	file := filepath.Join(folder, "Note.md")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(folder, "Missing")
	for vault, want := range map[string]string{
		"":      "BAKHYARD_VAULT is not set",
		missing: "open the vault: open " + missing + ": no such file or directory",
		file:    "open the vault: open " + file + ": not a directory",
	} {
		_, err := routes(t.Context(), envOf(map[string]string{vaultVar: vault}), noSecrets(t))
		if err == nil || err.Error() != want {
			t.Errorf("with the vault %q: got %v, want the error %s", vault, err, want)
		}
	}
}

// envOf returns an environment that has only settings, by name.
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

// noSecrets returns a secrets reader that fails t when notes reads a secret.
func noSecrets(t *testing.T) secrets.Reader {
	t.Helper()

	return func(_ context.Context, name string) (string, error) {
		t.Errorf("read the secret %s", name)

		return "", errors.New("no secrets")
	}
}
