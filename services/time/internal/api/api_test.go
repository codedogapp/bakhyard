package api_test

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/time/internal/api"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

func TestReportAnswersWhatThePollerFound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports := worklogs.NewPoller(func(context.Context, time.Time, time.Time) (
			[]jira.Worklog, error,
		) {
			return []jira.Worklog{{Started: time.Now(), Spent: time.Hour}}, nil
		}, time.UTC, slog.New(slog.DiscardHandler))
		go reports.Run(t.Context())

		snapshot, err := reports.Latest(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		want, err := json.Marshal(snapshot)
		if err != nil {
			t.Fatal(err)
		}

		rec := serve(t, routes(api.Backends{Reports: reports}), http.MethodGet)
		sent := rec.Result().Header.Get("Content-Type")

		if rec.Code != http.StatusOK || rec.Body.String() != string(want) ||
			sent != "application/json" {
			t.Errorf("got %d %s (%s), want 200 %s as JSON", rec.Code, rec.Body, sent, want)
		}
	})
}

func TestReportIsNotFoundWithoutJira(t *testing.T) {
	rec := serve(t, routes(api.Backends{}), http.MethodGet)

	want := "Jira is not set up\n"
	if rec.Code != http.StatusNotFound || rec.Body.String() != want {
		t.Errorf("got %d %q, want 404 %q", rec.Code, rec.Body, want)
	}
}

func TestReportTakesOnlyGET(t *testing.T) {
	rec := serve(t, routes(api.Backends{}), http.MethodPost)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", rec.Code)
	}
}

// routes returns the routes that Register gives backends, with days in UTC.
func routes(backends api.Backends) http.Handler {
	router := chi.NewRouter()
	api.Register(router, backends, time.UTC, slog.New(slog.DiscardHandler))

	return router
}

// serve returns router's answer to a request for the report with method.
func serve(t *testing.T, router http.Handler, method string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, "/time/report", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
