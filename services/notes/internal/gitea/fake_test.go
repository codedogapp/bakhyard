package gitea_test

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

const (
	// repoPath is the path of the fake's repository in its API.
	repoPath = "/api/v1/repos/owner/vault"
	// token is the bot user's token that the client sends.
	token = "bot-token"
)

// asked is what a request to the fake asked for.
type asked struct {
	method, path, query, auth, accept, contentType, body string
}

// fakeGitea is a Gitea that answers each request with reply, and keeps what each asked for.
type fakeGitea struct {
	reply http.HandlerFunc

	mu       sync.Mutex
	requests []asked
}

func (f *fakeGitea) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	f.mu.Lock()
	f.requests = append(f.requests, asked{
		r.Method, r.URL.EscapedPath(), r.URL.RawQuery, r.Header.Get("Authorization"),
		r.Header.Get("Accept"), r.Header.Get("Content-Type"), string(body),
	})
	f.mu.Unlock()

	f.reply(w, r)
}

// asks returns the requests that the fake has had.
func (f *fakeGitea) asks() []asked {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]asked(nil), f.requests...)
}

// clientOf returns a client of the repository owner/vault on a fake Gitea that answers each
// request with reply, and the fake.
func clientOf(t *testing.T, reply http.HandlerFunc) (*gitea.Client, *fakeGitea) {
	t.Helper()

	fake := &fakeGitea{reply: reply}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	repo, err := url.Parse(server.URL + repoPath)
	if err != nil {
		t.Fatal(err)
	}

	return gitea.New(gitea.Settings{Repo: repo, Token: token}), fake
}

// answer returns a reply with status and body.
func answer(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json;charset=utf-8")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

// fileAnswer returns Gitea's answer about a file that holds text in the blob blob.
func fileAnswer(blob, text string) string {
	return `{"file_contents":{"name":"Inbox.md","path":"Notes/Inbox.md","sha":"` + blob +
		`","type":"file","mode":"100644","size":` + strconv.Itoa(len(text)) +
		`,"encoding":"base64","content":"` + base64.StdEncoding.EncodeToString([]byte(text)) +
		`","target":null,"submodule_git_url":null}}`
}
