package github_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"slices"
	"testing"

	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// list is one of the client's lists: Mine, Reviews or Issues.
type list func(*github.Client, context.Context) ([]github.Item, error)

func TestEachListSearchesGitHubAsTheUser(t *testing.T) {
	for _, test := range []struct {
		name string
		list list
		uri  string
	}{
		{"Mine", (*github.Client).Mine, "/search/issues?order=desc&page=1&per_page=100" +
			"&q=is%3Apr+is%3Aopen+author%3A%40me+archived%3Afalse&sort=updated"},
		{"Reviews", (*github.Client).Reviews, "/search/issues?order=desc&page=1&per_page=100" +
			"&q=is%3Apr+is%3Aopen+review-requested%3A%40me+-author%3A%40me+archived%3Afalse" +
			"&sort=updated"},
		{"Issues", (*github.Client).Issues, "/search/issues?order=desc&page=1&per_page=100" +
			"&q=is%3Aissue+is%3Aopen+assignee%3A%40me+archived%3Afalse&sort=updated"},
	} {
		fake := &fakeGitHub{pages: map[string]reply{
			test.uri: okay(t, resultsOf(pullIn("web", 7), pullIn("api", 3))),
		}}

		got, err := test.list(clientOf(fake.serve(t)), t.Context())
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}

		if ids := idsOf(got); !slices.Equal(ids, []int{7, 3}) {
			t.Errorf("%s: got the items %v, want 7 and 3 in GitHub's order", test.name, ids)
		}

		want := []asked{{
			"GET " + test.uri, "Bearer " + token, "application/vnd.github+json", "2026-03-10",
			"bakhyard",
		}}
		if notes := fake.notes(); !slices.Equal(notes, want) {
			t.Errorf("%s: got the requests %q, want %q", test.name, notes, want)
		}
	}
}

func TestSearchReadsEachPageUpToAShortOne(t *testing.T) {
	fake := &fakeGitHub{pages: map[string]reply{
		searchFor(mine, 1): okay(t, resultsOf(pullsFrom(1, 100)...)),
		searchFor(mine, 2): okay(t, resultsOf(pullsFrom(101, 100)...)),
		searchFor(mine, 3): okay(t, resultsOf(pullsFrom(201, 99)...)),
		searchFor(mine, 4): okay(t, resultsOf(pullsFrom(300, 1)...)),
	}}

	got, err := clientOf(fake.serve(t)).Mine(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"GET " + searchFor(mine, 1), "GET " + searchFor(mine, 2), "GET " + searchFor(mine, 3),
	}
	if ids, requests := idsOf(got), fake.requests(); !slices.Equal(ids, upTo(299)) ||
		!slices.Equal(requests, want) {
		t.Errorf("got the items %v after the requests %q, want 1 to 299 after %q", ids,
			requests, want)
	}
}

func TestSearchStopsAtAnEmptyPage(t *testing.T) {
	fake := &fakeGitHub{pages: map[string]reply{
		searchFor(issues, 1): okay(t, resultsOf(pullsFrom(1, 100)...)),
		searchFor(issues, 2): okay(t, resultsOf()),
	}}

	got, err := clientOf(fake.serve(t)).Issues(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if ids, requests := idsOf(got), fake.requests(); !slices.Equal(ids, upTo(100)) ||
		len(requests) != 2 {
		t.Errorf("got the items %v after the requests %q, want 1 to 100 after two", ids,
			requests)
	}
}

func TestSearchReadsNoMoreThanTenPages(t *testing.T) {
	fake := &fakeGitHub{pages: map[string]reply{}}
	for page := 1; page <= 11; page++ {
		fake.pages[searchFor(reviews, page)] = okay(t, resultsOf(pullsFrom(page*100-99, 100)...))
	}

	got, err := clientOf(fake.serve(t)).Reviews(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if ids, requests := idsOf(got), fake.requests(); !slices.Equal(ids, upTo(1000)) ||
		len(requests) != 10 || requests[9] != "GET "+searchFor(reviews, 10) {
		t.Errorf("got %d items after the requests %q, want 1,000 after the first ten pages",
			len(ids), requests)
	}
}

func TestSearchListsEachItemOnce(t *testing.T) {
	// Two new pull requests pushed the last two of the first page onto the second.
	fake := &fakeGitHub{pages: map[string]reply{
		searchFor(mine, 1): okay(t, resultsOf(pullsFrom(1, 100)...)),
		searchFor(mine, 2): okay(t, resultsOf(pullsFrom(99, 5)...)),
	}}

	got, err := clientOf(fake.serve(t)).Mine(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if ids := idsOf(got); !slices.Equal(ids, upTo(103)) {
		t.Errorf("got the items %v, want 1 to 103, each once", ids)
	}
}

func TestSearchListsNothingAsAnEmptyList(t *testing.T) {
	fake := &fakeGitHub{pages: map[string]reply{
		searchFor(mine, 1): {
			http.StatusOK,
			`{"total_count":0,"incomplete_results":false,"items":[]}`,
		},
	}}

	got, err := clientOf(fake.serve(t)).Mine(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(got)
	if err != nil || string(encoded) != "[]" {
		t.Errorf("got %s, %v, want []", encoded, err)
	}
}
