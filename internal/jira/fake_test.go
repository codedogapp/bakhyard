package jira_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/internal/jira"
)

const (
	email = "you@example.com"
	token = "secret-token"
	// apiPath is where the fake Jira's REST API is, as Atlassian's is under /ex/jira/<cloudId>.
	apiPath = "/ex/jira/cloud-1"
)

// clientOf returns a client whose Jira site is https://acme.example and whose API, under
// apiPath, answers with handler, one request at a time.
func clientOf(t *testing.T, handler http.HandlerFunc) *jira.Client {
	t.Helper()

	var turn sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		turn.Lock()
		defer turn.Unlock()

		handler(w, r)
	}))
	t.Cleanup(server.Close)

	return jira.New(jira.Settings{
		Site:  parse(t, "https://acme.example"),
		API:   parse(t, server.URL+apiPath),
		Email: email,
		Token: token,
	})
}

func parse(t *testing.T, address string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// issue returns a ticket as Jira's search results hold it, with fields the client didn't ask
// for too.
func issue(key, summary, status, category string) map[string]any {
	return map[string]any{
		"expand": "", "id": "10001", "self": "https://example.invalid/rest/api/3/issue/10001",
		"key": key,
		"fields": map[string]any{
			"summary": summary,
			"status": map[string]any{
				"name": status, "id": "3", "iconUrl": "https://example.invalid/status.png",
				"statusCategory": map[string]any{"key": category, "id": 4, "colorName": "yellow"},
			},
		},
	}
}

// next returns a page of search results that another page follows, whose token is pageToken.
func next(pageToken string, issues ...map[string]any) map[string]any {
	return map[string]any{"issues": issues, "nextPageToken": pageToken, "isLast": false}
}

// last returns the last page of search results.
func last(issues ...map[string]any) map[string]any {
	return map[string]any{"issues": issues, "isLast": true}
}

func answer(t *testing.T, w http.ResponseWriter, body map[string]any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")

	err := json.MarshalWrite(w, body)
	if err != nil {
		t.Error(err)
	}
}

func keysOf(tickets []jira.Ticket) []string {
	var keys []string
	for _, ticket := range tickets {
		keys = append(keys, ticket.Key)
	}

	return keys
}
