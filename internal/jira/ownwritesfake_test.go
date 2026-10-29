package jira_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

// reply is an answer of the fake Jira: a status and a body.
type reply struct {
	status int
	body   string
}

// ownJira answers as its fakeJira does, and takes new worklogs too: it answers a new worklog on
// a ticket with the reply that added holds for the ticket's key, and a request whose path failing
// holds with that status alone. It doesn't record the new worklogs in asked.
type ownJira struct {
	fakeJira

	added   map[string]reply
	failing map[string]int
}

func (o *ownJira) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		o.add(w, r)

		return
	}

	status, failing := o.failing[pathOf(r)]
	if !failing {
		o.fakeJira.ServeHTTP(w, r)

		return
	}

	o.asked = append(o.asked, r)

	w.WriteHeader(status)
}

func (o *ownJira) add(w http.ResponseWriter, r *http.Request) {
	key, _ := strings.CutPrefix(pathOf(r), "issue/")
	key, _ = strings.CutSuffix(key, "/worklog")

	answer, known := o.added[key]
	if !known {
		o.t.Errorf("unexpected worklog on %s", key)
		http.NotFound(w, r)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(answer.status)
	_, _ = w.Write([]byte(answer.body))
}

// createdOn returns Jira's answer to a new worklog on the ticket whose ID is issueID.
func createdOn(issueID string) reply {
	return reply{
		http.StatusCreated,
		`{"id":"100099","issueId":"` + issueID + `","started":"2026-10-27T09:00:00.000+0900",` +
			`"timeSpent":"1h","timeSpentSeconds":3600}`,
	}
}

// web9 returns a fake Jira that adds worklogs to WEB-9, whose ID is 10009, and whose search finds
// the tickets whose IDs are searched. I logged half an hour on 10001 and an hour on WEB-9.
func web9(t *testing.T, searched ...string) *ownJira {
	t.Helper()

	var issues []map[string]any
	for _, id := range searched {
		issues = append(issues, found(id))
	}

	return &ownJira{
		t:     t,
		pages: map[string]map[string]any{"": last(issues...)},
		worklogs: map[string]map[string]any{
			"10001@0": worklogs(1, logged(myAccount, "2026-10-28T10:00:00.000+0900", 1800)),
			"10009@0": worklogs(2,
				logged(myAccount, "2026-10-27T09:00:00.000+0900", 3600),
				logged(otherAccount, "2026-10-27T11:00:00.000+0900", 60),
			),
		},
		added: map[string]reply{"WEB-9": createdOn("10009")},
	}
}

// addTo adds an hour that starts at started to the ticket key.
func addTo(t *testing.T, client *jira.Client, key, started string) {
	t.Helper()

	err := client.AddWorklog(t.Context(), key, jira.Worklog{
		Started: at(t, started), Spent: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
}

// poll returns the worklogs of aWeek, and the requests that the fake answered for them.
func poll(t *testing.T, client *jira.Client, fake *ownJira) ([]jira.Worklog, []string) {
	t.Helper()

	fake.asked = nil
	start, end := aWeek(t)

	got, err := client.Worklogs(t.Context(), start, end)
	if err != nil {
		t.Fatal(err)
	}

	return got, fake.trail()
}

// trailOf returns the requests of a poll of aWeek that reads the worklogs of the tickets whose
// IDs are read, in order.
func trailOf(read ...string) []string {
	trail := []string{"myself ", "search/jql "}
	for _, id := range read {
		trail = append(trail, "issue/"+id+"/worklog 0")
	}

	return trail
}
