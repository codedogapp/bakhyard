package githubapi_test

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/githubapi"
)

// asked is what a stand-in GitHub noted of a request.
type asked struct {
	request, authorization, accept, version, agent string
}

// answer is a stand-in GitHub's answer to every request.
type answer struct {
	status int
	header http.Header
}

// standIn starts a stand-in GitHub that answers each request with reply, and returns a client of
// its API, at path on the stand-in, with the token "secret", and what the stand-in notes of each
// request.
func standIn(t *testing.T, path string, reply answer) (*githubapi.Client, <-chan asked) {
	t.Helper()

	notes := make(chan asked, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notes <- asked{
			r.Method + " " + r.URL.RequestURI(), r.Header.Get("Authorization"),
			r.Header.Get("Accept"), r.Header.Get("X-GitHub-Api-Version"), r.UserAgent(),
		}

		maps.Copy(w.Header(), reply.header)
		w.WriteHeader(reply.status)
		_, _ = w.Write([]byte(`{"login":"you"}`))
	}))
	t.Cleanup(server.Close)

	api, err := url.Parse(server.URL + path)
	if err != nil {
		t.Fatal(err)
	}

	return githubapi.New(githubapi.Settings{API: api, Token: "secret"}), notes
}

func TestGetAsksAsTheUser(t *testing.T) {
	// GitHub Enterprise Server's API is below /api/v3, where api.github.com's is at the root.
	for _, prefix := range []string{"", "/api/v3"} {
		client, notes := standIn(t, prefix, answer{status: http.StatusOK})

		res, err := client.Get(t.Context(), "search/issues", url.Values{"q": {"is:pr is:open"}},
			"search GitHub")
		if err != nil {
			t.Fatalf("%q: %v", prefix, err)
		}

		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()

		if err != nil || string(body) != `{"login":"you"}` {
			t.Errorf("%q: got the body %q and %v, want GitHub's answer", prefix, body, err)
		}

		want := asked{
			"GET " + prefix + "/search/issues?q=is%3Apr+is%3Aopen", "Bearer secret",
			"application/vnd.github+json", "2026-03-10", "bakhyard",
		}
		if got := <-notes; got != want {
			t.Errorf("%q: got the request %q, want %q", prefix, got, want)
		}
	}
}

func TestGetFailsWhenGitHubSaysNo(t *testing.T) {
	for _, status := range []int{
		http.StatusNoContent, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
	} {
		client, _ := standIn(t, "", answer{status: status})

		res, err := client.Get(t.Context(), "user", nil, "ask GitHub")

		want := "ask GitHub: " + strconv.Itoa(status) + " " + http.StatusText(status)
		if res != nil || err == nil || err.Error() != want {
			t.Errorf("%d: got %v and %v, want the error %s", status, res, err, want)
		}
	}
}

func TestGetFailsWithoutGitHub(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	api, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := githubapi.New(githubapi.Settings{API: api, Token: "secret"})

	res, err := client.Get(t.Context(), "user", nil, "ask GitHub")
	if res != nil || err == nil || !strings.HasPrefix(err.Error(), "ask GitHub: ") {
		t.Errorf("got %v and %v, want an error that starts with ask GitHub", res, err)
	}
}

func TestGetStopsWhenItsContextEnds(t *testing.T) {
	client, notes := standIn(t, "", answer{status: http.StatusOK})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := client.Get(ctx, "user", nil, "ask GitHub")
	if res != nil || !errors.Is(err, context.Canceled) || len(notes) != 0 {
		t.Errorf("got %v and %v after %d requests, want context.Canceled before any request",
			res, err, len(notes))
	}
}
