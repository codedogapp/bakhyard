package jira_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/internal/secrets"
)

// The Jira site and REST API in the settings of most tests.
const (
	siteURL = "https://acme.atlassian.net"
	apiURL  = "https://api.atlassian.com/ex/jira/cloud-id"
)

func TestFromEnvSearchesTheJiraInTheSettings(t *testing.T) {
	client, err := jira.FromEnv(envOf(settingsAt(standIn(t))),
		secretsWith(map[string]string{"jira-token": token}))
	if err != nil {
		t.Fatal(err)
	}

	got, err := client.Search(t.Context())
	if err != nil || len(got) != 1 || got[0].URL != siteURL+"/browse/PAY-1" {
		t.Errorf("got %v, %v, want PAY-1 on %s", got, err, siteURL)
	}
}

func TestFromEnvIsOffWithoutTheSettings(t *testing.T) {
	client, err := jira.FromEnv(envOf(nil), secretsWith(nil))
	if client != nil || err != nil {
		t.Errorf("got %v, %v, want no client and no error", client, err)
	}
}

func TestFromEnvRejects(t *testing.T) {
	for _, test := range []rejection{
		{map[string]string{"BAKHYARD_JIRA_SITE": siteURL}, "BAKHYARD_JIRA_API is not set"},
		{map[string]string{"BAKHYARD_JIRA_API": apiURL}, "BAKHYARD_JIRA_SITE is not set"},
		{map[string]string{"BAKHYARD_JIRA_EMAIL": email}, "BAKHYARD_JIRA_SITE is not set"},
		{settingsWith("BAKHYARD_JIRA_SITE", ""), "BAKHYARD_JIRA_SITE is not set"},
		{settingsWith("BAKHYARD_JIRA_API", ""), "BAKHYARD_JIRA_API is not set"},
		{settingsWith("BAKHYARD_JIRA_EMAIL", ""), "BAKHYARD_JIRA_EMAIL is not set"},
		// internal/env tests which URLs pass; these show that both of Jira's URLs must.
		notHTTPS("BAKHYARD_JIRA_SITE", "http://acme.atlassian.net"),
		notHTTPS("BAKHYARD_JIRA_API", "http://example.com"),
	} {
		// Without the token, so the settings must be checked before it's read.
		client, err := jira.FromEnv(envOf(test.settings), secretsWith(nil))
		if client != nil || err == nil || err.Error() != test.want {
			t.Errorf("got %v, %v, want the error %s", client, err, test.want)
		}
	}
}

func TestFromEnvNeedsTheToken(t *testing.T) {
	client, err := jira.FromEnv(envOf(settingsAt(apiURL)), secretsWith(nil))

	want := "read the Jira token: no secret jira-token"
	if client != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want the error %s", client, err, want)
	}
}

// standIn returns the URL of a stand-in for Jira's REST API that has one open ticket, PAY-1, for
// email with token.
func standIn(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != email || password != token {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		answer(t, w, last(issue("PAY-1", "Pay", "To Do", "new")))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// settingsAt returns the Jira settings, with the REST API at address.
func settingsAt(address string) map[string]string {
	return map[string]string{
		"BAKHYARD_JIRA_SITE": siteURL, "BAKHYARD_JIRA_API": address, "BAKHYARD_JIRA_EMAIL": email,
	}
}

// settingsWith returns the Jira settings, with name set to value.
func settingsWith(name, value string) map[string]string {
	settings := settingsAt(apiURL)
	settings[name] = value

	return settings
}

// rejection is a row of TestFromEnvRejects: the settings, and the error they give.
type rejection struct {
	settings map[string]string
	want     string
}

// notHTTPS returns the rejection of value in the setting name as not an https URL.
func notHTTPS(name, value string) rejection {
	return rejection{settingsWith(name, value), name + `: "` + value + `" is not an https URL`}
}

// envOf returns an environment that holds only settings.
func envOf(settings map[string]string) func(string) string {
	return func(name string) string { return settings[name] }
}

// secretsWith returns a Reader of the secrets in items.
func secretsWith(items map[string]string) secrets.Reader {
	return func(_ context.Context, name string) (string, error) {
		secret, ok := items[name]
		if !ok {
			return "", fmt.Errorf("no secret %s", name)
		}

		return secret, nil
	}
}
