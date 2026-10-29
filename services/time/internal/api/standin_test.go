package api_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/time/internal/api"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// standIn is a stand-in for the user's worklogs in Jira: add keeps each worklog that it's given,
// unless the stand-in has a failure or the context has ended, and read answers those added so far.
type standIn struct {
	mu      sync.Mutex
	failure error
	keys    []string
	added   []jira.Worklog
	reads   int
}

func (s *standIn) add(ctx context.Context, key string, worklog jira.Worklog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if s.failure != nil {
		return s.failure
	}

	s.keys = append(s.keys, key)
	s.added = append(s.added, worklog)

	return nil
}

func (s *standIn) read(ctx context.Context, _, _ time.Time) ([]jira.Worklog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	s.reads++

	return slices.Clone(s.added), nil
}

// calls returns the keys that add took, the worklogs it added, and the number of reads.
func (s *standIn) calls() ([]string, []jira.Worklog, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.keys), slices.Clone(s.added), s.reads
}

// backends returns the backends on the stand-in, whose report's days follow loc.
func (s *standIn) backends(loc *time.Location) api.Backends {
	return api.Backends{
		Reports: worklogs.NewPoller(s.read, loc, slog.New(slog.DiscardHandler)),
		Add:     s.add,
	}
}

// timeAPI is time's API, with the logs it writes.
type timeAPI struct {
	router http.Handler
	logs   *bytes.Buffer
}

// newTimeAPI returns the API that Register serves from backends, with days in loc.
func newTimeAPI(backends api.Backends, loc *time.Location) timeAPI {
	logs := &bytes.Buffer{}
	router := chi.NewRouter()
	api.Register(router, backends, loc, slog.New(slog.NewTextHandler(logs, nil)))

	return timeAPI{router: router, logs: logs}
}

// post returns the API's answer to POST /time/worklogs with body, sent with ctx.
func (a timeAPI) post(ctx context.Context, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/time/worklogs",
		strings.NewReader(body))
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, req)

	return rec
}
