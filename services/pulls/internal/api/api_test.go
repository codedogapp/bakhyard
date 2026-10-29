package api_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/pulls/internal/api"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

func TestEachRouteAnswersWhatItsPollerFound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mine := started(t, bitbucket.NewPoller, bitbucket.Pull{ID: 1})
		reviews := started(t, bitbucket.NewReviewsPoller, bitbucket.Pull{ID: 2})

		for target, id := range map[string]string{"/pulls/mine": "1", "/pulls/reviews": "2"} {
			rec := serve(t, routes(mine, reviews), http.MethodGet, target)
			sent := rec.Result().Header.Get("Content-Type")

			want := `{"pulls":[{"id":` + id + `,"title":"","repo":"","draft":false,` +
				`"created":"0001-01-01T00:00:00Z","updated":"0001-01-01T00:00:00Z"}],` +
				`"searched":"2000-01-01T00:00:00Z"}`
			if rec.Code != http.StatusOK || rec.Body.String() != want ||
				sent != "application/json" {
				t.Errorf("%s: got %d %s (%s), want 200 %s as JSON", target, rec.Code, rec.Body,
					sent, want)
			}
		}
	})
}

func TestEachRouteIsNotFoundWithoutBitbucket(t *testing.T) {
	for _, target := range []string{"/pulls/mine", "/pulls/reviews"} {
		rec := serve(t, routes(nil, nil), http.MethodGet, target)

		want := "Bitbucket is not set up\n"
		if rec.Code != http.StatusNotFound || rec.Body.String() != want {
			t.Errorf("%s: got %d %q, want 404 %q", target, rec.Code, rec.Body, want)
		}
	}
}

func TestEachRouteTakesOnlyGET(t *testing.T) {
	for _, target := range []string{"/pulls/mine", "/pulls/reviews"} {
		rec := serve(t, routes(nil, nil), http.MethodPost, target)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: got %d, want 405", target, rec.Code)
		}
	}
}

// started returns a poller, made by start, that finds pull and polls until the test ends.
func started(
	t *testing.T,
	start func(func(context.Context) ([]bitbucket.Pull, error), *slog.Logger) *bitbucket.Poller,
	pull bitbucket.Pull,
) *bitbucket.Poller {
	t.Helper()

	poller := start(func(context.Context) ([]bitbucket.Pull, error) {
		return []bitbucket.Pull{pull}, nil
	}, slog.New(slog.DiscardHandler))
	go poller.Run(t.Context())

	return poller
}

// routes returns the routes that Register gives the pollers mine and reviews.
func routes(mine, reviews *bitbucket.Poller) http.Handler {
	router := chi.NewRouter()
	api.Register(router, mine, reviews)

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
