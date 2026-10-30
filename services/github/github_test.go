package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

func TestGitHubClientIsOffWithoutTheSetting(t *testing.T) {
	// No token: it isn't read without the setting.
	client, err := githubClient(envOf(nil), secretsWith(nil))
	if client != nil || err != nil {
		t.Errorf("got %v, %v, want no client and no error", client, err)
	}
}

func TestGitHubClientNeedsAnHTTPSAPI(t *testing.T) {
	for _, address := range []string{
		"http://api.github.com", "ftp://api.github.com", "api.github.com", "https://", "://",
	} {
		// No token: the setting is checked before the token is read.
		client, err := githubClient(envOf(map[string]string{githubAPIVar: address}),
			secretsWith(nil))

		want := `BAKHYARD_GITHUB_API: "` + address + `" is not an https URL`
		if client != nil || err == nil || err.Error() != want {
			t.Errorf("%s: got %v, %v, want the error %s", address, client, err, want)
		}
	}
}

func TestGitHubClientNeedsTheToken(t *testing.T) {
	client, err := githubClient(envOf(map[string]string{githubAPIVar: "https://api.github.com"}),
		secretsWith(nil))

	want := "read the GitHub token: no secret github-token"
	if client != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want the error %s", client, err, want)
	}
}

func TestGitHubStopsOnABadSetting(t *testing.T) {
	_, err := routes(t.Context(), envOf(map[string]string{githubAPIVar: "http://api.github.com"}),
		secretsWith(nil))

	want := `BAKHYARD_GITHUB_API: "http://api.github.com" is not an https URL`
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want the error %s", err, want)
	}
}

func TestEachListLogsItsFailedSearches(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)

	client, err := githubClient(envOf(map[string]string{githubAPIVar: server.URL}),
		secretsWith(map[string]string{"github-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer

	pollers := []string{"GitHub", "GitHub reviews", "GitHub issues"}
	searches := []string{"my open PRs", "my review requests", "my issues"}

	mine, reviews, issues := poll(t.Context(), client, slog.New(slog.NewTextHandler(&logs, nil)))
	for _, poller := range []*github.Poller{mine, reviews, issues} {
		// Latest waits for the first search, which logs its failure before it ends.
		if _, err := poller.Latest(t.Context()); err != nil {
			t.Fatal(err)
		}
	}

	var want []string
	for i, poller := range pollers {
		want = append(want, `msg="poll `+poller+`" err="search GitHub for `+searches[i]+
			`: 404 Not Found"`)
	}

	var got []string

	for line := range strings.Lines(logs.String()) {
		_, logged, _ := strings.Cut(strings.TrimSuffix(line, "\n"), "level=ERROR ")
		got = append(got, logged)
	}

	if slices.Sort(got); !slices.Equal(got, slices.Sorted(slices.Values(want))) {
		t.Errorf("got the logs %q, want %q", got, want)
	}
}

// githubServer returns the address of a stand-in for GitHub's API, for the token secret. Its
// search finds one item: my open PR in acme/web, a PR in acme/api that waits for my review, or my
// issue in acme/docs.
func githubServer(t *testing.T) string {
	t.Helper()

	repos := map[string]string{
		"is:pr is:open author:@me archived:false":                       "acme/web",
		"is:pr is:open review-requested:@me -author:@me archived:false": "acme/api",
		"is:issue is:open assignee:@me archived:false":                  "acme/docs",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repo, found := repos[r.URL.Query().Get("q")]
		if r.Header.Get("Authorization") != "Bearer secret" || r.URL.Path != "/search/issues" ||
			!found {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = w.Write([]byte(`{"incomplete_results":false,"items":[{"id":1,"number":1,` +
			`"repository_url":"https://api.github.com/repos/` + repo + `"}]}`))
	}))
	t.Cleanup(server.Close)

	return server.URL
}
