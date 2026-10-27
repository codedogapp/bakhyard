package bitbucket_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// failed reports whether err is an error that starts with want and doesn't hold the token, with
// no pull requests in got.
func failed(got []bitbucket.Pull, err error, want string) bool {
	return got == nil && err != nil && strings.HasPrefix(err.Error(), want) &&
		!strings.Contains(err.Error(), token)
}

// answers are the requests to which the client gets an answer, and the actions that their
// errors name.
var answers = []struct{ uri, action string }{
	{"/2.0/user", "find me on Bitbucket"},
	{mineIn("acme"), "list my Bitbucket pull requests"},
}

// mineWith returns what Mine gives when Bitbucket answers uri with answer, and the rest well.
func mineWith(t *testing.T, uri string, answer reply) ([]bitbucket.Pull, error) {
	t.Helper()

	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
		mineIn("acme"): {http.StatusOK, `{"values":[]}`},
	}}
	if uri == "/2.0/user" {
		fake.user = answer
	} else {
		fake.pages[uri] = answer
	}

	return clientOf(fake.serve(t), "acme").Mine(t.Context())
}

// padded returns an answer that is valid JSON, as a user and as a page, and size bytes long.
func padded(size int) string {
	start, end := `{"uuid":"`+myUUID+`","values":[],"pad":"`, `"}`

	return start + strings.Repeat("x", size-len(start)-len(end)) + end
}

func TestMineFailsOnABadAnswer(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		// want starts the error, after the failed action when it starts with a colon.
		want string
	}{
		{"401", "", http.StatusUnauthorized, ": 401 Unauthorized"},
		{"204", "", http.StatusNoContent, ": 204 No Content"},
		{
			"404", `{"type":"error","error":{"message":"No such workspace"}}`, http.StatusNotFound,
			": 404 Not Found",
		},
		{"not JSON", "<html>Log in</html>", http.StatusOK, "read Bitbucket's answer: "},
		{"cut short", `{"uuid":"` + myUUID + `","values":[`, http.StatusOK, "read Bitbucket's answer: "},
		{"over 4 MiB", padded(4<<20 + 1), http.StatusOK, "read Bitbucket's answer: "},
	} {
		for _, step := range answers {
			got, err := mineWith(t, step.uri, reply{test.status, test.body})

			want := test.want
			if strings.HasPrefix(want, ":") {
				want = step.action + want
			}

			if !failed(got, err, want) {
				t.Errorf("%s at %s: got %v, %v, want an error that starts with %q", test.name,
					step.uri, got, err, want)
			}
		}
	}
}

func TestMineReadsAnAnswerOfFourMiB(t *testing.T) {
	for _, step := range answers {
		got, err := mineWith(t, step.uri, reply{http.StatusOK, padded(4 << 20)})
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("at %s: got %v, %v, want an empty list", step.uri, got, err)
		}
	}
}

func TestMineFailsOnABadPull(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
		mineIn("acme"): {http.StatusOK, `{"values":[{"id":1,"updated_on":"yesterday"}]}`},
	}}

	got, err := clientOf(fake.serve(t), "acme").Mine(t.Context())
	if !failed(got, err, "read Bitbucket's answer: ") {
		t.Errorf("got %v, %v, want an error that names the answer", got, err)
	}
}

func TestMineNeedsMyUUID(t *testing.T) {
	for _, body := range []string{`{}`, `{"uuid":""}`, `{"uuid":null}`} {
		fake := &fakeBitbucket{user: reply{http.StatusOK, body}}

		got, err := clientOf(fake.serve(t), "acme").Mine(t.Context())
		if !failed(got, err, "find me on Bitbucket: no uuid in the answer") ||
			len(fake.requests()) != 1 {
			t.Errorf("answered %s: got %v, %v after %q, want an error after one request", body,
				got, err, fake.requests())
		}
	}
}

func TestMineGivesUpAfterTenPages(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{}}
	api := fake.serve(t)
	again := api.Scheme + "://" + api.Host + "/2.0/again"
	fake.pages[mineIn("acme")] = okay(t, pageOf(again, pullRequest(1, "One", "web", 1)))
	fake.pages["/2.0/again"] = fake.pages[mineIn("acme")]

	got, err := clientOf(api, "acme").Mine(t.Context())
	if !failed(got, err, "list my Bitbucket pull requests: more than 10 pages") ||
		len(fake.requests()) != 11 {
		t.Errorf("got %v, %v after %d requests, want an error after 11", got, err,
			len(fake.requests()))
	}
}

func TestMineFollowsNoNextLinkOffTheAPI(t *testing.T) {
	var visited atomic.Bool

	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		visited.Store(true)
	}))
	t.Cleanup(elsewhere.Close)

	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{}}
	api := fake.serve(t)

	for _, link := range []string{
		elsewhere.URL + "/2.0/next",
		"https://" + api.Host + "/2.0/next",
		"http://bitbucket.example/2.0/next",
		"/2.0/next",
		"http://[::1",
	} {
		fake.pages[mineIn("acme")] = okay(t, pageOf(link, pullRequest(1, "One", "web", 1)))

		got, err := clientOf(api, "acme").Mine(t.Context())
		if !failed(got, err, "list my Bitbucket pull requests: the next page is off "+
			api.Host) || visited.Load() {
			t.Errorf("%s: got %v, %v, visited elsewhere %t, want an error", link, got, err,
				visited.Load())
		}
	}
}

func TestMineFailsWithoutBitbucket(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	api, err := url.Parse(server.URL + "/2.0")
	if err != nil {
		t.Fatal(err)
	}

	got, err := clientOf(api, "acme").Mine(t.Context())
	if !failed(got, err, "find me on Bitbucket: ") {
		t.Errorf("got %v, %v, want an error that names the request", got, err)
	}
}

func TestMineStopsWhenItsContextEnds(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t)}
	api := fake.serve(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got, err := clientOf(api, "acme").Mine(ctx)
	if got != nil || !errors.Is(err, context.Canceled) || len(fake.requests()) != 0 {
		t.Errorf("got %v, %v after %q, want context.Canceled before any request", got, err,
			fake.requests())
	}
}
