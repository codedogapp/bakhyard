package api_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/games/internal/api"
	"github.com/codedogapp/bakhyard/services/games/internal/exophase"
	"github.com/codedogapp/bakhyard/services/games/internal/refresh"
)

func TestRecentAnswersWhatWasFound(t *testing.T) {
	for _, test := range []struct {
		name  string
		found api.Recent
		want  string
	}{
		{"no games", api.Recent{}, `{"games":[],"searched":"2000-01-01T00:00:00Z"}`},
		{
			"a game",
			api.Recent{Games: []exophase.Game{{
				Title: "Hades", Platforms: []string{"PS5"},
				LastPlayed: time.Date(1999, 12, 31, 22, 0, 0, 0, time.UTC),
			}}},
			`{"games":[{"title":"Hades","platforms":["PS5"],"lastPlayed":"1999-12-31T22:00:00Z"}],` +
				`"searched":"2000-01-01T00:00:00Z"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// A bubble's clock starts at midnight UTC on 1 January 2000.
			synctest.Test(t, func(t *testing.T) {
				recent := refresh.New("Exophase", func(context.Context) (api.Recent, error) {
					return test.found, nil
				}, slog.New(slog.DiscardHandler))
				go recent.Run(t.Context())

				code, body := serve(t, recent, http.MethodGet)
				if code != http.StatusOK || body != test.want {
					t.Errorf("got %d %s, want 200 %s", code, body, test.want)
				}
			})
		})
	}
}

func TestRecentWithoutExophase(t *testing.T) {
	for _, test := range []struct {
		method string
		code   int
		body   string
	}{
		{http.MethodGet, http.StatusNotFound, "Exophase is not set up\n"},
		{http.MethodPost, http.StatusMethodNotAllowed, ""},
	} {
		code, body := serve(t, nil, test.method)
		if code != test.code || body != test.body {
			t.Errorf("%s: got %d %q, want %d %q", test.method, code, body, test.code, test.body)
		}
	}
}

// serve returns the status and body of the answer to a request with method for /games/recent,
// on a router where recent is registered.
func serve(t *testing.T, recent *refresh.OnRequest[api.Recent], method string) (int, string) {
	t.Helper()

	router := chi.NewRouter()
	api.Register(router, recent)

	req := httptest.NewRequestWithContext(t.Context(), method, "/games/recent", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec.Code, rec.Body.String()
}
