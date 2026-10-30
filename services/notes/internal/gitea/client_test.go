package gitea_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

// call is one of the client's calls on the file at name: a Read or an Update.
type call func(ctx context.Context, client *gitea.Client, name string) error

// calls returns the client's calls, by name.
func calls() map[string]call {
	return map[string]call{
		"Read": func(ctx context.Context, client *gitea.Client, name string) error {
			_, err := client.Read(ctx, name)

			return err
		},
		"Update": func(ctx context.Context, client *gitea.Client, name string) error {
			return client.Update(ctx, name, "b1", []byte("text"))
		},
	}
}

func TestNamesOutsideTheRepositorySendNothing(t *testing.T) {
	for _, name := range []string{
		"", ".", "..", "../Other.md", "Notes/../../Other.md", "Notes/./A.md", "/A.md", "Notes/",
		"Notes//A.md",
	} {
		for kind, call := range calls() {
			client, fake := clientOf(t, answer(http.StatusOK, fileAnswer("b1", "text")))

			err := call(t.Context(), client, name)
			if err == nil || !strings.Contains(err.Error(), "isn't a file's path") {
				t.Errorf("%s of %q: got %v, want an error that says it isn't a path", kind, name,
					err)
			}

			if asks := fake.asks(); len(asks) != 0 {
				t.Errorf("%s of %q: got the requests %q, want none", kind, name, asks)
			}
		}
	}
}

func TestARepoWithATrailingSlashReachesTheSameFile(t *testing.T) {
	fake := &fakeGitea{reply: answer(http.StatusOK, fileAnswer("b1", "text"))}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	repo, err := url.Parse(server.URL + repoPath + "/")
	if err != nil {
		t.Fatal(err)
	}

	_, err = gitea.New(gitea.Settings{Repo: repo, Token: token}).Read(t.Context(), "A.md")
	if err != nil {
		t.Fatal(err)
	}

	if asks := fake.asks(); len(asks) != 1 || asks[0].path != repoPath+"/contents-ext/A.md" {
		t.Errorf("got the requests %q, want one for A.md", asks)
	}
}

func TestEachCallFailsWithoutGitea(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	repo, err := url.Parse(server.URL + repoPath)
	if err != nil {
		t.Fatal(err)
	}

	client := gitea.New(gitea.Settings{Repo: repo, Token: token})

	for kind, call := range calls() {
		err := call(t.Context(), client, "A.md")
		if err == nil || errors.Is(err, gitea.ErrNotFound) || errors.Is(err, gitea.ErrChanged) {
			t.Errorf("%s: got %v, want an error", kind, err)
		}
	}
}

func TestEachCallStopsWhenItsContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	for kind, call := range calls() {
		client, fake := clientOf(t, answer(http.StatusOK, fileAnswer("b1", "text")))

		err := call(ctx, client, "A.md")
		if !errors.Is(err, context.Canceled) {
			t.Errorf("%s: got %v, want context.Canceled", kind, err)
		}

		if asks := fake.asks(); len(asks) != 0 {
			t.Errorf("%s: got the requests %q, want none", kind, asks)
		}
	}
}
