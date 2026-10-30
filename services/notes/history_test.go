package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotesLinksToTheVaultsHistory(t *testing.T) {
	const history = "https://gitea.example/me/vault/commits/branch/main"

	for _, setting := range []string{history, history + "/"} {
		settings := map[string]string{vaultVar: t.TempDir(), historyVar: setting}

		router, err := routes(t.Context(), envOf(settings), noSecrets(t))
		if err != nil {
			t.Fatal(err)
		}

		list := serve(
			router,
			httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/notes", nil),
		)
		if want := `{"files":[],"history":"` + history + `"}`; list.Body.String() != want {
			t.Errorf("with the history %q: got %q, want %q", setting, list.Body.String(), want)
		}
	}
}

func TestNotesNeedsAnHTTPSHistory(t *testing.T) {
	folder := t.TempDir()

	for _, history := range []string{
		"http://gitea.example/me/vault/commits/branch/main",
		"javascript:alert(1)",
		"gitea.example/me/vault/commits/branch/main",
		"https:///me/vault/commits/branch/main",
		"https://gitea.example/me/vault/commits/branch/main?page=2",
		"https://gitea.example/me/vault/commits/branch/main#top",
		" https://gitea.example/me/vault/commits/branch/main",
	} {
		_, err := routes(t.Context(),
			envOf(map[string]string{vaultVar: folder, historyVar: history}), noSecrets(t))

		want := fmt.Sprintf(
			"BAKHYARD_VAULT_HISTORY: %q is not an https URL with no query or fragment", history)
		if err == nil || err.Error() != want {
			t.Errorf("with the history %q: got %v, want the error %s", history, err, want)
		}
	}
}
