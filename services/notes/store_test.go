package main

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotesSavesThroughTheVaultsRepository(t *testing.T) {
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "Inbox.md"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	settings := map[string]string{
		vaultVar: folder, repoVar: giteaStandIn(t).URL + "/api/v1/repos/me/vault",
	}

	router, err := routes(t.Context(), envOf(settings),
		secretsWith(map[string]string{tokenAccount: "s3cret"}))
	if err != nil {
		t.Fatal(err)
	}

	list := serve(
		router,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/notes", nil),
	)
	if want := `{"files":["Inbox.md"],"editable":true}`; list.Body.String() != want {
		t.Errorf("the list: got %q, want %q", list.Body.String(), want)
	}

	save := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/notes/files/Inbox.md",
		strings.NewReader("# Saved"))
	// The SHA-256 of "{}".
	save.Header.Set("If-Match",
		`"44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"`)

	if rec := serve(router, save); rec.Code != http.StatusNoContent {
		t.Errorf("the save: got %d %q, want 204", rec.Code, rec.Body.String())
	}
}

func TestNotesNeedsASecureRepositoryAndItsToken(t *testing.T) {
	folder := t.TempDir()

	for _, test := range []struct {
		repo   string
		tokens map[string]string
		want   string
	}{
		{
			"http://gitea.example/api/v1/repos/me/vault",
			map[string]string{tokenAccount: "s3cret"},
			`BAKHYARD_VAULT_REPO: "http://gitea.example/api/v1/repos/me/vault" ` +
				"is not an https URL",
		},
		{
			"https://gitea.example/api/v1/repos/me/vault", nil,
			"read the vault's write token: no secret vault-write-token",
		},
	} {
		_, err := routes(
			t.Context(),
			envOf(map[string]string{vaultVar: folder, repoVar: test.repo}),
			secretsWith(test.tokens),
		)
		if err == nil || err.Error() != test.want {
			t.Errorf("with the repository %s: got %v, want the error %s", test.repo, err, test.want)
		}
	}
}

// giteaStandIn returns a stand-in for Gitea's API that holds Inbox.md with "{}", in the blob
// "b1", in the repository me/vault, and takes an update of it in place of that blob, from the
// token s3cret alone.
func giteaStandIn(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/repos/me/vault/contents-ext/Inbox.md",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "token s3cret" {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			_, _ = io.WriteString(w, `{"file_contents":{"type":"file","sha":"b1",`+
				`"encoding":"base64","content":"e30="}}`)
		})
	mux.HandleFunc("PUT /api/v1/repos/me/vault/contents/Inbox.md",
		func(w http.ResponseWriter, r *http.Request) {
			var change struct {
				SHA string `json:"sha"`
			}
			if r.Header.Get("Authorization") != "token s3cret" ||
				json.UnmarshalRead(r.Body, &change) != nil || change.SHA != "b1" {
				w.WriteHeader(http.StatusUnprocessableEntity)
			}
		})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

// serve returns router's answer to req.
func serve(router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
