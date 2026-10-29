package main

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// author is the author of a worklog in Jira's answers.
type author struct {
	AccountID string `json:"accountId"`
}

// loggedWork is a worklog in the body that adds it to Jira, and in Jira's answers.
type loggedWork struct {
	Author           author `json:"author,omitzero"`
	Started          string `json:"started"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
}

// worklogPage is a page of a ticket's worklogs in Jira's answers.
type worklogPage struct {
	Total    int          `json:"total"`
	Worklogs []loggedWork `json:"worklogs"`
}

func TestTimeReportsTheWorklogsThatItAddsBeforeJirasSearchFindsThem(t *testing.T) {
	router, err := routes(t.Context(), envOf(jiraSettings(writableJira(t))),
		secretsWith(map[string]string{"jira-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	// Past midnight, today's date is yesterday's: still one of the report's 7 days.
	body := `{"ticket":"WEB-9","date":"` + time.Now().Format(time.DateOnly) + `","minutes":90}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/time/worklogs",
		strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d %q, want 201", rec.Code, rec.Body)
	}

	var logged float64
	for _, day := range reportOf(t, router).Days {
		logged += day.LoggedHours
	}

	if logged != 1.5 {
		t.Errorf("got %v hours logged, want the 1.5 that time added", logged)
	}
}

// writableJira returns the URL of a stand-in for Jira's REST API, for the email me@example.com
// with the token secret, which keeps the worklogs added to WEB-9, the ticket 10001, but whose
// search doesn't find them yet.
func writableJira(t *testing.T) string {
	t.Helper()

	var (
		lock  sync.Mutex
		added worklogPage
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, token, ok := r.BasicAuth()
		if !ok || email != "me@example.com" || token != "secret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		lock.Lock()
		defer lock.Unlock()

		switch r.Method + " " + r.URL.Path {
		case "GET /rest/api/3/myself":
			_, _ = w.Write([]byte(`{"accountId":"me"}`))
		case "GET /rest/api/3/search/jql":
			_, _ = w.Write([]byte(`{"isLast":true,"issues":[]}`))
		case "POST /rest/api/3/issue/WEB-9/worklog":
			worklog := loggedWork{Author: author{AccountID: "me"}}
			if json.UnmarshalRead(r.Body, &worklog) != nil {
				w.WriteHeader(http.StatusBadRequest)

				return
			}

			added.Worklogs = append(added.Worklogs, worklog)
			added.Total++

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"issueId":"10001"}`))
		case "GET /rest/api/3/issue/10001/worklog":
			_ = json.MarshalWrite(w, added)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return server.URL
}
