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

func TestMineAnswersWhatThePollerFound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poller := bitbucket.NewPoller(func(context.Context) ([]bitbucket.Pull, error) {
			return []bitbucket.Pull{}, nil
		}, slog.New(slog.DiscardHandler))
		go poller.Run(t.Context())

		rec := serve(t, poller, http.MethodGet)
		sent := rec.Result().Header.Get("Content-Type")

		want := `{"pulls":[],"searched":"2000-01-01T00:00:00Z"}`
		if rec.Code != http.StatusOK || rec.Body.String() != want || sent != "application/json" {
			t.Errorf("got %d %s (%s), want 200 %s as JSON", rec.Code, rec.Body, sent, want)
		}
	})
}

func TestMineIsNotFoundWithoutBitbucket(t *testing.T) {
	rec := serve(t, nil, http.MethodGet)

	want := "Bitbucket is not set up\n"
	if rec.Code != http.StatusNotFound || rec.Body.String() != want {
		t.Errorf("got %d %q, want 404 %q", rec.Code, rec.Body, want)
	}
}

func TestMineTakesOnlyGET(t *testing.T) {
	rec := serve(t, nil, http.MethodPost)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", rec.Code)
	}
}

// serve returns the answer to a request for /pulls/mine with method, from the routes that
// Register gives the poller mine.
func serve(t *testing.T, mine *bitbucket.Poller, method string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	api.Register(router, mine)

	req := httptest.NewRequestWithContext(t.Context(), method, "/pulls/mine", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
