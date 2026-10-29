package jira_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

const (
	// myAccount and otherAccount are the account IDs of the user and of another user.
	myAccount    = "account-me"
	otherAccount = "account-someone"
)

// fakeJira answers as Jira does for the account myAccount: the search pages by their tokens, and each
// ticket's worklog pages by the ticket's escaped ID and their startAt, such as "10001@0".
type fakeJira struct {
	t        *testing.T
	pages    map[string]map[string]any
	worklogs map[string]map[string]any
	asked    []*http.Request
}

func (f *fakeJira) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.asked = append(f.asked, r)
	query := r.URL.Query()

	var (
		body  map[string]any
		known bool
	)

	switch path := pathOf(r); path {
	case "myself":
		body, known = map[string]any{
			"accountId":   myAccount,
			"displayName": "Me",
			"active":      true,
		}, true
	case "search/jql":
		body, known = f.pages[query.Get("nextPageToken")]
	default:
		id, _ := strings.CutPrefix(path, "issue/")
		id, _ = strings.CutSuffix(id, "/worklog")
		body, known = f.worklogs[id+"@"+query.Get("startAt")]
	}

	if !known {
		f.t.Errorf("unexpected request for %s", r.URL)
		http.NotFound(w, r)

		return
	}

	answer(f.t, w, body)
}

// trail returns each request that the fake answered as its path, then its page token or startAt.
func (f *fakeJira) trail() []string {
	var trail []string

	for _, r := range f.asked {
		query := r.URL.Query()
		trail = append(trail, pathOf(r)+" "+query.Get("nextPageToken")+query.Get("startAt"))
	}

	return trail
}

// pathOf returns the path of r under Jira's REST API, escaped.
func pathOf(r *http.Request) string {
	return strings.TrimPrefix(r.URL.EscapedPath(), apiPath+"/rest/api/3/")
}

// found returns a ticket as a search for IDs alone finds it.
func found(id string) map[string]any {
	return map[string]any{"id": id}
}

// logged returns a worklog as Jira's answers hold it, with fields the client doesn't read too.
func logged(author, started string, seconds int64) map[string]any {
	account := map[string]any{
		"accountId": author, "displayName": "Someone", "active": true,
		"avatarUrls": map[string]any{"48x48": "https://example.invalid/avatar.png"},
	}

	return map[string]any{
		"id": "100028", "issueId": "10002", "self": "https://example.invalid/worklog/100028",
		"author": account, "updateAuthor": account, "started": started,
		"created": started, "updated": started, "timeSpent": "1h", "timeSpentSeconds": seconds,
	}
}

// worklogs returns a page of a ticket's worklogs, of total worklogs in all its pages.
func worklogs(total int, entries ...map[string]any) map[string]any {
	return map[string]any{"maxResults": 1000, "total": total, "worklogs": entries}
}

// aWeek returns the start and end of a week in Tokyo.
func aWeek(t *testing.T) (start, end time.Time) {
	t.Helper()

	return at(t, "2026-10-26T00:00:00+09:00"), at(t, "2026-11-02T00:00:00+09:00")
}

func at(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// same reports whether got and want hold the same worklogs in the same order.
func same(got, want []jira.Worklog) bool {
	return slices.EqualFunc(got, want, func(a, b jira.Worklog) bool {
		return a.Started.Equal(b.Started) && a.Spent == b.Spent
	})
}
