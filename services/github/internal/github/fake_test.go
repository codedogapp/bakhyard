package github_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/internal/githubapi"
	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

const token = "secret-token"

// The searches of Mine, Reviews and Issues.
const (
	mine    = "is:pr is:open author:@me archived:false"
	reviews = "is:pr is:open review-requested:@me -author:@me archived:false"
	issues  = "is:issue is:open assignee:@me archived:false"
)

// reply is a fake GitHub's answer to a request.
type reply struct {
	status int
	body   string
}

// asked is what a fake GitHub noted of a request.
type asked struct {
	request, authorization, accept, version, agent string
}

// fakeGitHub is a stand-in GitHub, whose API is at its root, as on api.github.com. It answers
// each request with the reply that pages holds for its URI, or 404.
type fakeGitHub struct {
	pages map[string]reply
	mu    sync.Mutex
	asked []asked
}

// serve starts f on a test server and returns the address of its API.
func (f *fakeGitHub) serve(t *testing.T) *url.URL {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		f.asked = append(f.asked, asked{
			r.Method + " " + r.URL.RequestURI(), r.Header.Get("Authorization"),
			r.Header.Get("Accept"), r.Header.Get("X-GitHub-Api-Version"), r.UserAgent(),
		})

		answer, found := f.pages[r.URL.RequestURI()]
		if !found {
			answer = reply{http.StatusNotFound, `{"message":"Not Found","status":"404"}`}
		}

		w.WriteHeader(answer.status)
		_, _ = w.Write([]byte(answer.body))
	}))
	t.Cleanup(server.Close)

	api, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return api
}

// notes returns what f noted of the requests it was asked.
func (f *fakeGitHub) notes() []asked {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.asked)
}

// requests returns the requests that f was asked, as their methods and URIs.
func (f *fakeGitHub) requests() []string {
	var requests []string
	for _, request := range f.notes() {
		requests = append(requests, request.request)
	}

	return requests
}

// clientOf returns a client of the GitHub whose API is at api.
func clientOf(api *url.URL) *github.Client {
	return github.New(githubapi.Settings{API: api, Token: token})
}

// okay returns a 200 reply with body as JSON.
func okay(t *testing.T, body any) reply {
	t.Helper()

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	return reply{http.StatusOK, string(encoded)}
}

// searchFor returns the URI at which the client asks for page of the search for query.
func searchFor(query string, page int) string {
	return "/search/issues?order=desc&page=" + strconv.Itoa(page) + "&per_page=100&q=" +
		url.QueryEscape(query) + "&sort=updated"
}

// resultsOf returns a page of search results that holds found.
func resultsOf(found ...map[string]any) map[string]any {
	return map[string]any{"total_count": len(found), "incomplete_results": false, "items": found}
}

// pullIn returns the pull request number in acme's repo as GitHub's search results hold it, with
// fields that the client doesn't read too. Its id across GitHub is 1000 plus number.
func pullIn(repo string, number int) map[string]any {
	api := "https://api.github.com/repos/acme/" + repo
	page := "https://github.com/acme/" + repo + "/pull/" + strconv.Itoa(number)

	return map[string]any{
		"url": api + "/issues/" + strconv.Itoa(number), "repository_url": api,
		"html_url": page, "id": 1000 + number, "node_id": "PR_kwDOAbc", "number": number,
		"title": "Pull " + strconv.Itoa(number), "user": map[string]any{"login": "you", "id": 1},
		"labels": []any{}, "state": "open", "locked": false, "assignees": []any{}, "comments": 2,
		"created_at": "2026-03-01T09:00:00Z", "updated_at": "2026-03-01T12:00:00Z",
		"closed_at": nil, "author_association": "OWNER", "draft": false,
		"pull_request": map[string]any{"html_url": page, "merged_at": nil}, "score": 1.0,
	}
}

// pullsFrom returns count pull requests in acme's web, numbered from first.
func pullsFrom(first, count int) []map[string]any {
	var pulls []map[string]any
	for number := range count {
		pulls = append(pulls, pullIn("web", first+number))
	}

	return pulls
}

func idsOf(items []github.Item) []int {
	var ids []int
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

// upTo returns the numbers from 1 to last.
func upTo(last int) []int {
	var all []int
	for number := 1; number <= last; number++ {
		all = append(all, number)
	}

	return all
}
