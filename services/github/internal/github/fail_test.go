package github_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// failed reports whether err is an error that starts with want and doesn't hold the token, with
// no items in got.
func failed(got []github.Item, err error, want string) bool {
	return got == nil && err != nil && strings.HasPrefix(err.Error(), want) &&
		!strings.Contains(err.Error(), token)
}

// mineWith returns what Mine gives when GitHub answers its page with answer, after full pages
// before it.
func mineWith(t *testing.T, page int, answer reply) ([]github.Item, error) {
	t.Helper()

	fake := &fakeGitHub{pages: map[string]reply{searchFor(mine, page): answer}}
	for before := 1; before < page; before++ {
		fake.pages[searchFor(mine, before)] = okay(t, resultsOf(pullsFrom(before*100-99, 100)...))
	}

	return clientOf(fake.serve(t)).Mine(t.Context())
}

// padded returns an answer that is valid JSON, with no results, and size bytes long.
func padded(size int) string {
	start, end := `{"total_count":0,"incomplete_results":false,"items":[],"pad":"`, `"}`

	return start + strings.Repeat("x", size-len(start)-len(end)) + end
}

func TestSearchFailsOnABadAnswer(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		want       string
	}{
		{
			"401", `{"message":"Bad credentials"}`, http.StatusUnauthorized,
			"search GitHub for my open PRs: 401 Unauthorized",
		},
		{
			"403", `{"message":"API rate limit exceeded"}`, http.StatusForbidden,
			"search GitHub for my open PRs: 403 Forbidden",
		},
		{
			"422", `{"message":"Validation Failed"}`, http.StatusUnprocessableEntity,
			"search GitHub for my open PRs: 422 Unprocessable Entity",
		},
		{"204", "", http.StatusNoContent, "search GitHub for my open PRs: 204 No Content"},
		{"not JSON", "<html>Unicorn!</html>", http.StatusOK, "read GitHub's answer: "},
		{
			"cut short", `{"incomplete_results":false,"items":[`, http.StatusOK,
			"read GitHub's answer: ",
		},
		{"over 4 MiB", padded(4<<20 + 1), http.StatusOK, "read GitHub's answer: "},
		{
			"incomplete", `{"total_count":1,"incomplete_results":true,"items":[]}`, http.StatusOK,
			"search GitHub for my open PRs: GitHub's results are incomplete",
		},
	} {
		for _, page := range []int{1, 2} {
			got, err := mineWith(t, page, reply{test.status, test.body})
			if !failed(got, err, test.want) {
				t.Errorf("%s on page %d: got %d items and %v, want an error that starts with %q",
					test.name, page, len(got), err, test.want)
			}
		}
	}
}

func TestEachListNamesItsSearchWhenItFails(t *testing.T) {
	for _, test := range []struct {
		list  list
		query string
		want  string
	}{
		{(*github.Client).Mine, mine, "search GitHub for my open PRs: 401 Unauthorized"},
		{(*github.Client).Reviews, reviews, "search GitHub for my review requests: 401 Unauthorized"},
		{(*github.Client).Issues, issues, "search GitHub for my issues: 401 Unauthorized"},
	} {
		fake := &fakeGitHub{pages: map[string]reply{
			searchFor(test.query, 1): {http.StatusUnauthorized, `{"message":"Bad credentials"}`},
		}}

		got, err := test.list(clientOf(fake.serve(t)), t.Context())
		if !failed(got, err, test.want) {
			t.Errorf("got %v, %v, want an error that starts with %q", got, err, test.want)
		}
	}
}

func TestSearchReadsAnAnswerOfFourMiB(t *testing.T) {
	got, err := mineWith(t, 1, reply{http.StatusOK, padded(4 << 20)})
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("got %v, %v, want an empty list", got, err)
	}
}

func TestSearchFailsOnAnItemWithNoRepository(t *testing.T) {
	for _, address := range []string{
		"", "https://api.github.com", "https://api.github.com/users/acme",
		"https://api.github.com/repos/acme", "https://api.github.com/repos//web",
		"https://api.github.com/repos/acme/", "https://api.github.com/repos/acme/web/issues",
		"%zz",
	} {
		found := pullIn("web", 1)
		found["repository_url"] = address

		got, err := mineWith(t, 1, okay(t, resultsOf(found)))
		if !failed(got, err, "read GitHub's answer: no repository in ") {
			t.Errorf("%q: got %v, %v, want an error that names the address", address, got, err)
		}
	}
}

func TestSearchFailsOnABadItem(t *testing.T) {
	for field, value := range map[string]any{
		"number": "7", "updated_at": "yesterday", "id": "1001", "draft": "no",
	} {
		found := pullIn("web", 1)
		found[field] = value

		got, err := mineWith(t, 1, okay(t, resultsOf(found)))
		if !failed(got, err, "read GitHub's answer: ") {
			t.Errorf("%s %v: got %v, %v, want an error that names the answer", field, value,
				got, err)
		}
	}
}

func TestSearchFailsWithoutGitHub(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	api, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	got, err := clientOf(api).Mine(t.Context())
	if !failed(got, err, "search GitHub for my open PRs: ") {
		t.Errorf("got %v, %v, want an error that names the search", got, err)
	}
}

func TestSearchStopsWhenItsContextEnds(t *testing.T) {
	fake := &fakeGitHub{}
	api := fake.serve(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got, err := clientOf(api).Mine(ctx)
	if got != nil || !errors.Is(err, context.Canceled) || len(fake.requests()) != 0 {
		t.Errorf("got %v, %v after %q, want context.Canceled before any request", got, err,
			fake.requests())
	}
}
