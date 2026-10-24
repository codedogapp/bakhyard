package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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

	router, err := routes(vaultAt(folder))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct{ method, target, want string }{
		{http.MethodGet, "/notes", `{"files":["Inbox.md"]}`},
		{http.MethodGet, "/notes/files/Inbox.md", "{}"},
		{http.MethodHead, "/notes/files/Inbox.md", ""},
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
		_, err := routes(vaultAt(vault))
		if err == nil || err.Error() != want {
			t.Errorf("with the vault %q: got %v, want the error %s", vault, err, want)
		}
	}
}

// vaultAt returns an environment whose BAKHYARD_VAULT is folder.
func vaultAt(folder string) func(string) string {
	return func(name string) string { return map[string]string{vaultVar: folder}[name] }
}
