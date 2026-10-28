package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestTodosListsItsTodosWhoseTicketIsDoneAsAttentionItems(t *testing.T) {
	getenv, readSecret := jiraEnv(t, jiraServer(t))
	addTodo(t, getenv, readSecret, "WEB-3")
	addTodo(t, getenv, readSecret, "PAY-1")

	got := listItems(t, served(t, getenv, readSecret))

	want := []string{"1 | Ship | WEB-3 · Done in Jira | SOURCE_JIRA"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTodosListsItsJiraInboxUntilItsTicketIsDismissed(t *testing.T) {
	getenv, readSecret := jiraEnv(t, jiraServer(t))
	addTodo(t, getenv, readSecret, "WEB-3")
	router := served(t, getenv, readSecret)

	want := []string{
		"PAY-1 | Pay | PAY-1 · In the Jira inbox | SOURCE_JIRA",
		"1 | Ship | WEB-3 · Done in Jira | SOURCE_JIRA",
	}
	if got := listItems(t, router); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/todos/dismissals/PAY-1",
		http.NoBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d %q, want PAY-1 dismissed", rec.Code, rec.Body)
	}

	if got := listItems(t, router); !slices.Equal(got, want[1:]) {
		t.Errorf("after the dismissal, got %q, want %q", got, want[1:])
	}
}

func TestTodosListsItsHighTodosAsAttentionItemsWithoutJira(t *testing.T) {
	database, items := todosEnv(t)
	router := served(t, database, secretsWith(items))

	for _, call := range []struct{ method, target, body string }{
		{http.MethodPost, "/todos", `{"title":"Call the bank"}`},
		{http.MethodPatch, "/todos/1", `{"priority":"high"}`},
		{http.MethodPost, "/todos", `{"title":"Buy milk"}`},
	} {
		req := httptest.NewRequestWithContext(t.Context(), call.method, call.target,
			strings.NewReader(call.body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("%s %s: got %d %q", call.method, call.target, rec.Code, rec.Body)
		}
	}

	want := []string{"1 | Call the bank | High priority | no source"}
	if got := listItems(t, router); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// served returns todos' routes, with the settings in getenv and the secrets that readSecret
// reads, which run until the test ends.
func served(t *testing.T, getenv func(string) string, readSecret secrets.Reader) http.Handler {
	t.Helper()

	router, err := routes(t.Context(), getenv, readSecret)
	if err != nil {
		t.Fatal(err)
	}

	return router
}

// listItems returns the items that router lists over the attention contract, called with the
// gRPC protocol, one line each: their ID, title, detail and source.
func listItems(t *testing.T, router http.Handler) []string {
	t.Helper()

	// gRPC needs HTTP/2; between services it runs without TLS (h2c), as internal/server allows.
	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	server := httptest.NewUnstartedServer(router)
	server.Config.Protocols = h2c
	server.Start()
	t.Cleanup(server.Close)

	transport := &http.Transport{Protocols: h2c}
	t.Cleanup(transport.CloseIdleConnections)

	httpClient := &http.Client{Transport: transport}
	client := attentionv1connect.NewAttentionServiceClient(
		connect.NewClient(connecthttp.NewTransport(httpClient, server.URL, connecthttp.WithGRPC())))

	res, err := client.ListItems(t.Context(), &attentionv1.ListItemsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	var lines []string

	for _, item := range res.GetItems() {
		source := "no source"
		if item.HasSource() {
			source = item.GetSource().String()
		}

		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", item.GetId(), item.GetTitle(),
			item.GetDetail(), source))
	}

	return lines
}
