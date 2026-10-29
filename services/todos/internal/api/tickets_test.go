package api_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/api"
	"github.com/codedogapp/bakhyard/services/todos/internal/tickets"
)

// A bubble's clock starts at midnight UTC on 1 January 2000, when the poller searches first.
func TestGetTicketsAnswersWhatThePollerFound(t *testing.T) {
	open := []jira.Ticket{
		{
			Key: "PAY-2", Summary: `Answer "Re: <the plan>"`, Status: "In Review",
			Category: jira.InProgress, URL: "https://acme.example/browse/PAY-2",
		},
		{
			Key: "PAY-1", Summary: "Export the report", Status: "To Do", Category: jira.ToDo,
			URL: "https://acme.example/browse/PAY-1",
		},
	}

	done := jira.Ticket{
		Key: "WEB-3", Summary: "Ship it", Status: "Closed", Category: jira.Done,
		URL: "https://acme.example/browse/WEB-3",
	}

	for _, test := range []struct {
		tickets []jira.Ticket
		err     error
		done    []jira.Ticket
		want    string
	}{
		{
			open, nil, nil,
			`{"tickets":[{"key":"PAY-2","summary":"Answer \"Re: <the plan>\"",` +
				`"status":"In Review","category":"in-progress",` +
				`"url":"https://acme.example/browse/PAY-2"},` +
				`{"key":"PAY-1","summary":"Export the report","status":"To Do",` +
				`"category":"to-do","url":"https://acme.example/browse/PAY-1"}],` +
				`"searched":"2000-01-01T00:00:00Z"}`,
		},
		{[]jira.Ticket{}, nil, nil, `{"tickets":[],"searched":"2000-01-01T00:00:00Z"}`},
		{
			[]jira.Ticket{},
			nil,
			[]jira.Ticket{done},
			`{"tickets":[],"done":[{"key":"WEB-3","summary":"Ship it","status":"Closed",` +
				`"category":"done","url":"https://acme.example/browse/WEB-3"}],` +
				`"searched":"2000-01-01T00:00:00Z"}`,
		},
		{
			nil, errors.New("search Jira: 401 Unauthorized"), nil,
			`{"failure":"search Jira: 401 Unauthorized"}`,
		},
	} {
		synctest.Test(t, func(t *testing.T) {
			router := withJira(t, func(context.Context) ([]jira.Ticket, error) {
				return test.tickets, test.err
			}, test.done...)

			rec := getTickets(t.Context(), router)
			if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
				rec.Header().Get("Content-Type") != "application/json" {
				t.Errorf("got %d %s (%s), want 200 %s as JSON", rec.Code, rec.Body,
					rec.Header().Get("Content-Type"), test.want)
			}
		})
	}
}

func TestGetTicketsNeedsJira(t *testing.T) {
	router := chi.NewRouter()
	api.Register(router, nil, nil, slog.New(slog.DiscardHandler))

	rec := getTickets(t.Context(), router)
	if rec.Code != http.StatusNotFound || rec.Body.String() != "Jira is not set up\n" {
		t.Errorf("got %d %q, want a plain 404", rec.Code, rec.Body)
	}
}

// withJira returns a router whose Jira poller searches with search and finds the todos'
// tickets done, until the test ends. It serves no todos.
func withJira(
	t *testing.T,
	search func(context.Context) ([]jira.Ticket, error),
	done ...jira.Ticket,
) http.Handler {
	t.Helper()

	fake := fakeJira{search: search, done: done}

	poller := tickets.NewPoller(fake, fake.linked, slog.New(slog.DiscardHandler))
	go poller.Run(t.Context())

	router := chi.NewRouter()
	api.Register(router, nil, poller, slog.New(slog.DiscardHandler))

	return router
}

// getTickets returns router's answer to GET /todos/tickets, from a client that leaves when ctx
// ends.
func getTickets(ctx context.Context, router http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/todos/tickets", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

// fakeJira is a fake Jira, whose search is search, and whose todos name the tickets done, which
// it finds done.
type fakeJira struct {
	search func(context.Context) ([]jira.Ticket, error)
	done   []jira.Ticket
}

func (f fakeJira) Search(ctx context.Context) ([]jira.Ticket, error) {
	return f.search(ctx)
}

func (f fakeJira) Fetch(context.Context, []string) ([]jira.Ticket, error) {
	return f.done, nil
}

func (f fakeJira) linked(context.Context) ([]string, error) {
	keys := []string{}
	for _, ticket := range f.done {
		keys = append(keys, ticket.Key)
	}

	return keys, nil
}
