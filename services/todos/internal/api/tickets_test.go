package api_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/todos/internal/api"
	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
)

// onePage is a Jira search's only page, with two tickets.
const onePage = `{"isLast":true,"issues":[
	{"id":"10002","key":"PAY-2","fields":{"summary":"Answer \"Re: <the plan>\"",
		"status":{"name":"In Review","statusCategory":{"key":"indeterminate"}}}},
	{"id":"10001","key":"PAY-1","fields":{"summary":"Export the report",
		"status":{"name":"To Do","statusCategory":{"key":"new"}}}}]}`

func TestGetTicketsListsTheOpenTickets(t *testing.T) {
	for _, test := range []struct{ page, want string }{
		{
			onePage,
			`{"tickets":[{"key":"PAY-2","summary":"Answer \"Re: <the plan>\"",` +
				`"status":"In Review","category":"in-progress",` +
				`"url":"https://acme.example/browse/PAY-2"},` +
				`{"key":"PAY-1","summary":"Export the report","status":"To Do",` +
				`"category":"to-do","url":"https://acme.example/browse/PAY-1"}]}`,
		},
		{`{"isLast":true,"issues":[]}`, `{"tickets":[]}`},
	} {
		router, _ := withJira(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(test.page))
		})

		rec := getTickets(t, router)
		if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
			rec.Header().Get("Content-Type") != "application/json" {
			t.Errorf("got %d %s (%s), want 200 %s as JSON", rec.Code, rec.Body,
				rec.Header().Get("Content-Type"), test.want)
		}
	}
}

func TestGetTicketsFailsWhenJiraFails(t *testing.T) {
	router, logs := withJira(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	rec := getTickets(t, router)
	if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" {
		t.Errorf("got %d %q, want a plain 502", rec.Code, rec.Body)
	}

	want := `msg="list the tickets" err="search Jira: 500 Internal Server Error"`
	if !strings.Contains(logs.String(), want) {
		t.Errorf("got the logs %q, want %s", logs, want)
	}
}

func TestGetTicketsNeedsJira(t *testing.T) {
	router := chi.NewRouter()
	api.Register(router, nil, nil, slog.New(slog.DiscardHandler))

	rec := getTickets(t, router)
	if rec.Code != http.StatusNotFound || rec.Body.String() != "Jira is not set up\n" {
		t.Errorf("got %d %q, want a plain 404", rec.Code, rec.Body)
	}
}

// withJira returns a router whose Jira's REST API answers with handler, and the logs that the
// router writes. It serves no todos.
func withJira(t *testing.T, handler http.HandlerFunc) (http.Handler, *bytes.Buffer) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	site, siteErr := url.Parse("https://acme.example")
	address, apiErr := url.Parse(server.URL)

	if siteErr != nil || apiErr != nil {
		t.Fatal(siteErr, apiErr)
	}

	var logs bytes.Buffer

	router := chi.NewRouter()
	api.Register(router, nil, jira.New(jira.Settings{Site: site, API: address}),
		slog.New(slog.NewTextHandler(&logs, nil)))

	return router, &logs
}

// getTickets returns router's answer to GET /todos/tickets.
func getTickets(t *testing.T, router http.Handler) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/todos/tickets", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
