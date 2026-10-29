package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/time/internal/report"
)

// startedLayout is the layout of the time that a worklog starts in Jira's answers.
const startedLayout = "2006-01-02T15:04:05.000-0700"

// answer is time's answer to GET /time/report.
type answer struct {
	report.Report

	Searched time.Time `json:"searched"`
	Failure  string    `json:"failure"`
}

func TestTimeReportsMyWorklogsInJira(t *testing.T) {
	rec := getReport(t.Context(), t, jiraServer(t))

	var got answer

	err := json.Unmarshal(rec.Body.Bytes(), &got)
	if err != nil || rec.Code != http.StatusOK {
		t.Fatalf("got %d %s (%v), want 200 and a report", rec.Code, rec.Body, err)
	}

	var logged float64
	for _, day := range got.Days {
		logged += day.LoggedHours
	}

	// The worklog started a minute ago, which may be yesterday, or last week.
	if len(got.Days) != 7 || logged != 1.5 || got.Week.TargetHours != 40 ||
		got.Searched.IsZero() || got.Failure != "" {
		t.Errorf("got %s, want 7 days with 1.5 hours logged, a target of 40 hours, the time "+
			"of the search and no failure", rec.Body)
	}
}

func TestTimeStopsReadingWorklogsWhenItStops(t *testing.T) {
	stopped, stop := context.WithCancel(t.Context())
	stop()

	// Nothing listens on port 1, so a read that went ahead would fail.
	rec := getReport(stopped, t, "http://127.0.0.1:1")
	if rec.Code != http.StatusOK || rec.Body.String() != "{}" {
		t.Errorf("got %d %s, want 200 {}: no read, and no failure", rec.Code, rec.Body)
	}
}

func TestTimeStopsOnABadJiraSetting(t *testing.T) {
	settings := jiraSettings("https://api.atlassian.com/ex/jira/cloud-id")
	settings["BAKHYARD_JIRA_EMAIL"] = ""

	_, err := routes(t.Context(), envOf(settings), secretsWith(nil))
	if err == nil || err.Error() != "BAKHYARD_JIRA_EMAIL is not set" {
		t.Errorf("got %v, want the error BAKHYARD_JIRA_EMAIL is not set", err)
	}
}

// jiraServer returns the URL of a stand-in for Jira's REST API, for the email me@example.com with
// the token secret, where the user logged an hour and a half on one ticket, starting a minute
// before each answer, and an hour more with each later read of its worklogs.
func jiraServer(t *testing.T) string {
	t.Helper()

	var reads atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, token, ok := r.BasicAuth()
		if !ok || email != "me@example.com" || token != "secret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		switch r.URL.Path {
		case "/rest/api/3/myself":
			_, _ = w.Write([]byte(`{"accountId":"me"}`))
		case "/rest/api/3/search/jql":
			_, _ = w.Write([]byte(`{"isLast":true,"issues":[{"id":"10001"}]}`))
		case "/rest/api/3/issue/10001/worklog":
			started := time.Now().Add(-time.Minute).Format(startedLayout)
			spent := 1800 + 3600*reads.Add(1)
			_, _ = fmt.Fprintf(w, `{"total":1,"worklogs":[{"author":{"accountId":"me"},`+
				`"started":"%s","timeSpentSeconds":%d}]}`, started, spent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// getReport returns the answer to GET /time/report of time with Jira's REST API at address,
// whose routes run until ctx ends.
func getReport(ctx context.Context, t *testing.T, address string) *httptest.ResponseRecorder {
	t.Helper()

	router, err := routes(ctx, envOf(jiraSettings(address)),
		secretsWith(map[string]string{"jira-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/time/report", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

// jiraSettings returns the Jira settings, with the REST API at address.
func jiraSettings(address string) map[string]string {
	return map[string]string{
		"BAKHYARD_JIRA_SITE": "https://acme.atlassian.net", "BAKHYARD_JIRA_API": address,
		"BAKHYARD_JIRA_EMAIL": "me@example.com",
	}
}
