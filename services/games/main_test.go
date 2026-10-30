package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGamesServesWithoutExophase(t *testing.T) {
	router, err := routes(t.Context(), envOf(nil), unasked())
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
		unasked())

	want := `BAKHYARD_EXOPHASE_PLAYER: "me" is not an Exophase profile id, which is digits only`
	if router != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want no handler and the error %q", router, err, want)
	}
}

// unasked returns an address of Exophase's API that the tests never ask.
func unasked() *url.URL {
	return &url.URL{Scheme: "https", Host: "api.exophase.example"}
}

// envOf returns an environment that holds only settings.
func envOf(settings map[string]string) func(string) string {
	return func(name string) string { return settings[name] }
}
