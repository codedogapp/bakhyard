package bitbucket_test

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

const (
	email = "you@example.com"
	token = "secret-token"
	// myUUID is the user's uuid, which Bitbucket wraps in braces.
	myUUID = "{6b0c7a4e-1d2f-4c3b-9a8e-5f6d7c8b9a01}"
)

// reply is a fake Bitbucket's answer to a request.
type reply struct {
	status int
	body   string
}

// asked is what a fake Bitbucket noted of a request.
type asked struct {
	request, user, password, accept string
}

// fakeBitbucket is a stand-in Bitbucket, whose API is under /2.0. It answers GET /2.0/user with
// user, and any other request with the reply that pages holds for its URI, or 404.
type fakeBitbucket struct {
	user  reply
	pages map[string]reply
	mu    sync.Mutex
	asked []asked
}

// serve starts f on a test server and returns the address of its API.
func (f *fakeBitbucket) serve(t *testing.T) *url.URL {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		user, password, _ := r.BasicAuth()
		f.asked = append(f.asked, asked{
			r.Method + " " + r.URL.RequestURI(), user, password, r.Header.Get("Accept"),
		})

		answer, found := f.pages[r.URL.RequestURI()]
		if r.URL.Path == "/2.0/user" {
			answer, found = f.user, true
		}

		if !found {
			answer = reply{http.StatusNotFound, `{"type":"error"}`}
		}

		w.WriteHeader(answer.status)
		_, _ = w.Write([]byte(answer.body))
	}))
	t.Cleanup(server.Close)

	api, err := url.Parse(server.URL + "/2.0")
	if err != nil {
		t.Fatal(err)
	}

	return api
}

// notes returns what f noted of the requests it was asked.
func (f *fakeBitbucket) notes() []asked {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.asked)
}

// requests returns the requests that f was asked, as their methods and URIs.
func (f *fakeBitbucket) requests() []string {
	var requests []string
	for _, request := range f.notes() {
		requests = append(requests, request.request)
	}

	return requests
}

// clientOf returns a client of the workspaces whose API is at api.
func clientOf(api *url.URL, workspaces ...string) *bitbucket.Client {
	return bitbucket.New(bitbucket.Settings{
		API: api, Email: email, Token: token, Workspaces: workspaces,
	})
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

// meAnswer is the user as Bitbucket's GET /user answers it.
func meAnswer(t *testing.T) reply {
	t.Helper()

	return okay(t, map[string]any{"uuid": myUUID, "display_name": "You", "account_id": "557058:1"})
}

// mineIn returns the URI at which the client asks for the user's open pull requests in
// workspace.
func mineIn(workspace string) string {
	return "/2.0/workspaces/" + workspace + "/pullrequests/" + url.PathEscape(myUUID) +
		"?pagelen=50&state=OPEN"
}

// pageOf returns a page of pull requests, whose next page is at next, if next isn't empty.
func pageOf(next string, values ...map[string]any) map[string]any {
	page := map[string]any{"pagelen": 50, "values": values}
	if next != "" {
		page["next"] = next
	}

	return page
}

// pullRequest returns a pull request into acme's repo as Bitbucket's pages hold it, with fields
// the client doesn't read too. It was last updated minute minutes past noon UTC on 1 March 2026.
func pullRequest(number int, title, repo string, minute int) map[string]any {
	return map[string]any{
		"type": "pullrequest", "id": number, "title": title, "state": "OPEN", "draft": false,
		"created_on":  "2026-03-01T09:00:00.123456+00:00",
		"updated_on":  fmt.Sprintf("2026-03-01T12:%02d:00.000000+00:00", minute),
		"source":      map[string]any{"repository": map[string]any{"full_name": "fork/" + repo}},
		"destination": map[string]any{"repository": map[string]any{"full_name": "acme/" + repo}},
		"links": map[string]any{"html": map[string]any{
			"href": "https://bitbucket.org/acme/" + repo + "/pull-requests/" + strconv.Itoa(number),
		}},
		"comment_count": 3,
	}
}

func idsOf(pulls []bitbucket.Pull) []int {
	var ids []int
	for _, pull := range pulls {
		ids = append(ids, pull.ID)
	}

	return ids
}
