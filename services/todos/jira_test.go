package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/internal/secrets"
)

// The Jira site and REST API in the settings of most tests.
const (
	jiraSite = "https://acme.atlassian.net"
	jiraAPI  = "https://api.atlassian.com/ex/jira/cloud-id"
)

func TestTodosListsTheJiraTickets(t *testing.T) {
	rec := getTickets(t.Context(), t, jiraServer(t))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"key":"PAY-1"`) ||
		strings.Contains(rec.Body.String(), `"done"`) {
		t.Errorf("got %d %q, want PAY-1 and no done tickets", rec.Code, rec.Body)
	}
}

func TestTodosListsTheDoneTicketsThatItsTodosName(t *testing.T) {
	rec := getTickets(t.Context(), t, jiraServer(t), "WEB-3", "PAY-1")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"done":[{"key":"WEB-3"`) {
		t.Errorf("got %d %q, want WEB-3 done", rec.Code, rec.Body)
	}
}

func TestTodosStopsSearchingJiraWhenItStops(t *testing.T) {
	stopped, stop := context.WithCancel(t.Context())
	stop()

	// Nothing listens on port 1, so a search that went ahead would fail.
	rec := getTickets(stopped, t, "http://127.0.0.1:1")
	if rec.Code != http.StatusOK || rec.Body.String() != "{}" {
		t.Errorf("got %d %s, want 200 {}: no search, and no failure", rec.Code, rec.Body)
	}
}

func TestTodosStopsOnABadJiraSetting(t *testing.T) {
	_, err := routes(t.Context(), envOf(jiraWith("BAKHYARD_JIRA_EMAIL", "")), secretsWith(nil))
	if err == nil || err.Error() != "BAKHYARD_JIRA_EMAIL is not set" {
		t.Errorf("got %v, want the error BAKHYARD_JIRA_EMAIL is not set", err)
	}
}

func TestTodosStopsOnABadTokenDate(t *testing.T) {
	_, err := routes(t.Context(), envOf(map[string]string{jiraExpiresVar: "2027-02-29"}),
		secretsWith(nil))

	want := `BAKHYARD_JIRA_TOKEN_EXPIRES: "2027-02-29" is not a date, YYYY-MM-DD`
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want the error %s", err, want)
	}
}

// jiraServer returns the URL of a stand-in for Jira's REST API that has one open ticket, PAY-1,
// and one done, WEB-3, for the email me@example.com with the token secret. A fetch finds WEB-3
// whatever it asks for.
func jiraServer(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, token, ok := r.BasicAuth()
		if !ok || email != "me@example.com" || token != "secret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if strings.HasSuffix(r.URL.Path, "/rest/api/3/issue/bulkfetch") {
			_, _ = w.Write([]byte(`{"issues":[{"key":"WEB-3","fields":{"summary":"Ship",` +
				`"status":{"name":"Done","statusCategory":{"key":"done"}}}}]}`))

			return
		}

		_, _ = w.Write([]byte(`{"isLast":true,"issues":[{"key":"PAY-1","fields":{` +
			`"summary":"Pay","status":{"name":"To Do","statusCategory":{"key":"new"}}}}]}`))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// getTickets returns the answer to GET /todos/tickets of todos with Jira's REST API at address,
// whose routes run until ctx ends. Before they start, it adds a todo for each of the keys.
func getTickets(
	ctx context.Context, t *testing.T, address string, keys ...string,
) *httptest.ResponseRecorder {
	t.Helper()

	getenv, readSecret := jiraEnv(t, address)

	for _, key := range keys {
		addTodo(t, getenv, readSecret, key)
	}

	router, err := routes(ctx, getenv, readSecret)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/todos/tickets", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

// jiraEnv returns todos' environment, with Jira's REST API at address and the database in a
// schema of its own, and its secrets, with the token of jiraServer.
func jiraEnv(t *testing.T, address string) (func(string) string, secrets.Reader) {
	t.Helper()

	schema := databasetest.NewSchema(t)
	settings := jiraSettings(address)
	settings[databaseVar] = schema.URL

	return envOf(settings), secretsWith(map[string]string{
		"postgres-todos": schema.Password, "jira-token": "secret",
	})
}

// addTodo adds a todo for the ticket key, through routes that stop at once.
func addTodo(t *testing.T, getenv func(string) string, readSecret secrets.Reader, key string) {
	t.Helper()

	stopped, stop := context.WithCancel(t.Context())
	stop()

	router, err := routes(stopped, getenv, readSecret)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/todos",
		strings.NewReader(`{"title":"Ship","ticket":"`+key+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d %q, want the todo for %s", rec.Code, rec.Body, key)
	}
}

// jiraSettings returns the Jira settings, with the REST API at address.
func jiraSettings(address string) map[string]string {
	return map[string]string{
		"BAKHYARD_JIRA_SITE": jiraSite, "BAKHYARD_JIRA_API": address,
		"BAKHYARD_JIRA_EMAIL": "me@example.com",
	}
}

// jiraWith returns the Jira settings, with name set to value.
func jiraWith(name, value string) map[string]string {
	settings := jiraSettings(jiraAPI)
	settings[name] = value

	return settings
}

// envOf returns an environment that holds only settings.
func envOf(settings map[string]string) func(string) string {
	return func(name string) string { return settings[name] }
}
