package api_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"testing/synctest"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/github/internal/api"
	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// targets are the routes of the lists.
var targets = []string{"/github/mine", "/github/reviews", "/github/issues"}

func TestEachRouteAnswersWhatItsPollerFound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		router := routes(started(t, 1), started(t, 2), started(t, 3))

		for i, target := range targets {
			rec := serve(t, router, http.MethodGet, target)
			sent := rec.Result().Header.Get("Content-Type")

			want := `{"items":[{"id":` + strconv.Itoa(i+1) + `,"title":"","repo":"",` +
				`"draft":false,"created":"0001-01-01T00:00:00Z",` +
				`"updated":"0001-01-01T00:00:00Z"}],"searched":"2000-01-01T00:00:00Z"}`
			if rec.Code != http.StatusOK || rec.Body.String() != want ||
				sent != "application/json" {
				t.Errorf("%s: got %d %s (%s), want 200 %s as JSON", target, rec.Code, rec.Body,
					sent, want)
			}
		}
	})
}

func TestEachRouteIsNotFoundWithoutGitHub(t *testing.T) {
	for _, target := range targets {
		rec := serve(t, routes(nil, nil, nil), http.MethodGet, target)

		want := "GitHub is not set up\n"
		if rec.Code != http.StatusNotFound || rec.Body.String() != want {
			t.Errorf("%s: got %d %q, want 404 %q", target, rec.Code, rec.Body, want)
		}
	}
}

func TestEachRouteTakesOnlyGET(t *testing.T) {
	for _, target := range targets {
		rec := serve(t, routes(nil, nil, nil), http.MethodPost, target)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: got %d, want 405", target, rec.Code)
		}
	}
}

// started returns a poller that finds the item numbered id and searches until the test ends.
func started(t *testing.T, id int) *github.Poller {
	t.Helper()

	poller := github.NewPoller("GitHub", func(context.Context) ([]github.Item, error) {
		return []github.Item{{ID: id}}, nil
	}, slog.New(slog.DiscardHandler))
	go poller.Run(t.Context())

	return poller
}

// routes returns the routes that Register gives the pollers mine, reviews and issues.
func routes(mine, reviews, issues *github.Poller) http.Handler {
	router := chi.NewRouter()
	api.Register(router, mine, reviews, issues)

	return router
}

// serve returns router's answer to a request for target with method.
func serve(t *testing.T, router http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
