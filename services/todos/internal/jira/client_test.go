package jira_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
)

func TestSearchAsksForTheOpenTickets(t *testing.T) {
	var asked *http.Request

	client := clientOf(t, func(w http.ResponseWriter, r *http.Request) {
		asked = r

		answer(t, w, last(issue("PAY-1", "Export the report", "In Review", "indeterminate")))
	})

	got, err := client.Search(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []jira.Ticket{{
		Key: "PAY-1", Summary: "Export the report", Status: "In Review",
		Category: jira.InProgress, URL: "https://acme.example/browse/PAY-1",
	}}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	query := asked.URL.Query()
	user, password, ok := asked.BasicAuth()

	for _, check := range []struct{ name, got, want string }{
		{"method", asked.Method, http.MethodGet},
		{"path", asked.URL.Path, apiPath + "/rest/api/3/search/jql"},
		{
			"jql", query.Get("jql"),
			"assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC",
		},
		{"fields", query.Get("fields"), "summary,status"},
		{"maxResults", query.Get("maxResults"), "100"},
		{"nextPageToken", strings.Join(query["nextPageToken"], ","), ""},
		{"user", user, email},
		{"password", password, token},
		{"Accept", asked.Header.Get("Accept"), "application/json"},
	} {
		if check.got != check.want {
			t.Errorf("%s: got %q, want %q", check.name, check.got, check.want)
		}
	}

	if !ok || len(query) != 3 {
		t.Errorf("got Basic auth %t and the query %v, want Basic auth and three parameters", ok,
			query)
	}
}

func TestSearchReadsEveryPage(t *testing.T) {
	pages := map[string]map[string]any{
		"":   next("p2", issue("PAY-3", "Three", "To Do", "new")),
		"p2": next("p3", issue("PAY-2", "Two", "To Do", "new")),
		"p3": last(issue("PAY-1", "One", "To Do", "new")),
	}

	var tokens []string

	client := clientOf(t, func(w http.ResponseWriter, r *http.Request) {
		pageToken := r.URL.Query().Get("nextPageToken")
		tokens = append(tokens, pageToken)
		answer(t, w, pages[pageToken])
	})

	got, err := client.Search(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if keys := keysOf(got); !slices.Equal(keys, []string{"PAY-3", "PAY-2", "PAY-1"}) ||
		!slices.Equal(tokens, []string{"", "p2", "p3"}) {
		t.Errorf("got the tickets %v with the page tokens %q, want three pages in order", keys,
			tokens)
	}
}

func TestSearchStopsAtTheLastPage(t *testing.T) {
	for _, body := range []string{
		`{"issues":[],"isLast":true,"nextPageToken":"more"}`,
		`{"issues":[],"isLast":false}`,
		`{"issues":[],"isLast":false,"nextPageToken":null}`,
		`{"issues":null}`,
	} {
		requests := 0
		client := clientOf(t, func(w http.ResponseWriter, _ *http.Request) {
			requests++
			_, _ = w.Write([]byte(body))
		})

		got, err := client.Search(t.Context())
		if err != nil || got == nil || len(got) != 0 || requests != 1 {
			t.Errorf("answered %s: got %v, %v after %d requests, want an empty list after one",
				body, got, err, requests)
		}
	}
}

func TestSearchGivesUpAfterTenPages(t *testing.T) {
	requests := 0
	client := clientOf(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++

		answer(t, w, next("again", issue("PAY-1", "One", "To Do", "new")))
	})

	got, err := client.Search(t.Context())
	if got != nil || err == nil || err.Error() != "search Jira: more than 10 pages" ||
		requests != 10 {
		t.Errorf("got %v, %v after %d requests, want an error after 10", got, err, requests)
	}
}

func TestSearchSortsTheStatusesIntoCategories(t *testing.T) {
	client := clientOf(t, func(w http.ResponseWriter, _ *http.Request) {
		answer(t, w, last(
			issue("A-1", "", "Open", "new"),
			issue("A-2", "", "Blocked", "indeterminate"),
			issue("A-3", "", "Closed", "done"),
			issue("A-4", "", "Odd", "undefined"),
			issue("A-5", "", "Newer", "in-flight"),
			issue("A-6", "", "Bare", ""),
		))
	})

	got, err := client.Search(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	var categories []jira.Category
	for _, ticket := range got {
		categories = append(categories, ticket.Category)
	}

	want := []jira.Category{
		jira.ToDo, jira.InProgress, jira.Done, jira.Unknown, jira.Unknown, jira.Unknown,
	}
	if !slices.Equal(categories, want) {
		t.Errorf("got %q, want %q", categories, want)
	}
}

func TestSearchLinksEachTicketToItsPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		answer(t, w, last(issue("PAY-1", "", "", ""), issue("A/B?c#d%", "", "", "")))
	}))
	t.Cleanup(server.Close)

	client := jira.New(jira.Settings{
		Site: parse(t, "https://acme.example/jira/"), API: parse(t, server.URL),
	})

	got, err := client.Search(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	var links []string
	for _, ticket := range got {
		links = append(links, ticket.URL)
	}

	want := []string{
		"https://acme.example/jira/browse/PAY-1",
		"https://acme.example/jira/browse/A%2FB%3Fc%23d%25",
	}
	if !slices.Equal(links, want) {
		t.Errorf("got %q, want %q", links, want)
	}
}
