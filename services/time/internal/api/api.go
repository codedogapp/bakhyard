// Package api serves the user's time report over HTTP, and adds worklogs to Jira.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// notSetUp is the answer of every route without Jira.
const notSetUp = "Jira is not set up"

// Backends are what the routes serve from. Without Jira, both are nil.
type Backends struct {
	// Reports polls Jira for the user's worklogs and keeps the time report.
	Reports *worklogs.Poller
	// Add adds worklog to the ticket key in Jira, such as a jira.Client's AddWorklog. It must be
	// the AddWorklog of the client that Reports reads with, so that the reads find the worklogs it
	// adds before Jira's search does (docs/plan.md decision 191).
	Add func(ctx context.Context, key string, worklog jira.Worklog) error
}

// Register serves the time report and the worklogs' adds on router. GET /time/report answers
// what backends.Reports has found in Jira, once its first poll has ended; POST /time/worklogs adds
// a worklog on a day that follows loc (addWorklog). Without Jira, the client gets a 404. Other
// methods get 405.
func Register(router chi.Router, backends Backends, loc *time.Location, logger *slog.Logger) {
	router.Get("/time/report", func(w http.ResponseWriter, r *http.Request) {
		if backends.Reports == nil {
			http.Error(w, notSetUp, http.StatusNotFound)

			return
		}

		backends.Reports.ServeHTTP(w, r)
	})
	router.Post("/time/worklogs", addWorklog(backends, loc, logger))
}
