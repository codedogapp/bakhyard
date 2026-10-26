package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/databasetest"
)

// The Jira site and REST API in the settings of most tests.
const (
	jiraSite = "https://acme.atlassian.net"
	jiraAPI  = "https://api.atlassian.com/ex/jira/cloud-id"
)

func TestJiraClientSearchesTheJiraInTheSettings(t *testing.T) {
	client, err := jiraClient(envOf(jiraSettings(jiraServer(t))),
		keychainWith(map[string]string{"jira-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	got, err := client.Search(t.Context())
	if err != nil || len(got) != 1 || got[0].URL != jiraSite+"/browse/PAY-1" {
		t.Errorf("got %v, %v, want PAY-1 on %s", got, err, jiraSite)
	}
}

func TestJiraClientTakesTheseURLs(t *testing.T) {
	for _, address := range []string{
		jiraAPI, "HTTPS://API.ATLASSIAN.COM/", "http://127.0.0.1:8080", "http://[::1]:8080",
	} {
		client, err := jiraClient(envOf(jiraWith(jiraAPIVar, address)),
			keychainWith(map[string]string{"jira-token": "secret"}))
		if client == nil || err != nil {
			t.Errorf("%s: got %v, %v, want a client", address, client, err)
		}
	}
}

func TestJiraClientIsOffWithoutTheSettings(t *testing.T) {
	client, err := jiraClient(envOf(nil), keychainWith(nil))
	if client != nil || err != nil {
		t.Errorf("got %v, %v, want no client and no error", client, err)
	}
}

func TestJiraClientRejects(t *testing.T) {
	for _, test := range []rejection{
		{map[string]string{jiraSiteVar: jiraSite}, "BAKHYARD_JIRA_API is not set"},
		{map[string]string{jiraAPIVar: jiraAPI}, "BAKHYARD_JIRA_SITE is not set"},
		{map[string]string{jiraEmailVar: "me@example.com"}, "BAKHYARD_JIRA_SITE is not set"},
		{jiraWith(jiraSiteVar, ""), "BAKHYARD_JIRA_SITE is not set"},
		{jiraWith(jiraAPIVar, ""), "BAKHYARD_JIRA_API is not set"},
		{jiraWith(jiraEmailVar, ""), "BAKHYARD_JIRA_EMAIL is not set"},
		notHTTPS(jiraSiteVar, "http://acme.atlassian.net"),
		notHTTPS(jiraAPIVar, "http://example.com"),
		notHTTPS(jiraAPIVar, "http://localhost:8080"),
		notHTTPS(jiraAPIVar, "http://192.168.0.2:8080"),
		notHTTPS(jiraAPIVar, "ftp://127.0.0.1"),
		notHTTPS(jiraAPIVar, "https://"),
		notHTTPS(jiraAPIVar, "https:api.atlassian.com"),
		notHTTPS(jiraAPIVar, "api.atlassian.com"),
		notHTTPS(jiraAPIVar, "https://api.atlassian.com/%zz"),
	} {
		client, err := jiraClient(envOf(test.settings),
			keychainWith(map[string]string{"jira-token": "secret"}))
		if client != nil || err == nil || err.Error() != test.want {
			t.Errorf("got %v, %v, want the error %s", client, err, test.want)
		}
	}
}

func TestJiraClientNeedsTheToken(t *testing.T) {
	client, err := jiraClient(envOf(jiraSettings(jiraAPI)), keychainWith(nil))

	want := "read the Jira token: no Keychain item bakhyard/jira-token"
	if client != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want the error %s", client, err, want)
	}
}

func TestTodosListsTheJiraTickets(t *testing.T) {
	schema := databasetest.NewSchema(t)
	settings := jiraSettings(jiraServer(t))
	settings[databaseVar] = schema.URL

	router, err := routes(envOf(settings), keychainWith(map[string]string{
		"postgres-todos": schema.Password, "jira-token": "secret",
	}))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/todos/tickets", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"key":"PAY-1"`) {
		t.Errorf("got %d %q, want PAY-1", rec.Code, rec.Body)
	}
}

func TestTodosStopsOnABadJiraSetting(t *testing.T) {
	_, err := routes(envOf(jiraWith(jiraEmailVar, "")), keychainWith(nil))
	if err == nil || err.Error() != "BAKHYARD_JIRA_EMAIL is not set" {
		t.Errorf("got %v, want the error BAKHYARD_JIRA_EMAIL is not set", err)
	}
}

// jiraServer returns the URL of a stand-in for Jira's REST API that has one open ticket, PAY-1,
// for the email me@example.com with the token secret.
func jiraServer(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, token, ok := r.BasicAuth()
		if !ok || email != "me@example.com" || token != "secret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		_, _ = w.Write([]byte(`{"isLast":true,"issues":[{"key":"PAY-1","fields":{` +
			`"summary":"Pay","status":{"name":"To Do","statusCategory":{"key":"new"}}}}]}`))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// jiraSettings returns the Jira settings, with the REST API at address.
func jiraSettings(address string) map[string]string {
	return map[string]string{
		jiraSiteVar: jiraSite, jiraAPIVar: address, jiraEmailVar: "me@example.com",
	}
}

// jiraWith returns the Jira settings, with name set to value.
func jiraWith(name, value string) map[string]string {
	settings := jiraSettings(jiraAPI)
	settings[name] = value

	return settings
}

// rejection is a row of TestJiraClientRejects: the settings, and the error they give.
type rejection struct {
	settings map[string]string
	want     string
}

// notHTTPS returns the rejection of value in the setting name as not an https URL.
func notHTTPS(name, value string) rejection {
	return rejection{jiraWith(name, value), name + `: "` + value + `" is not an https URL`}
}

// envOf returns an environment that holds only settings.
func envOf(settings map[string]string) func(string) string {
	return func(name string) string { return settings[name] }
}
