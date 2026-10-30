package gitea_test

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

func TestCreateCommitsTheTextAsANewFile(t *testing.T) {
	client, fake := clientOf(t, answer(http.StatusCreated, `{"content":{"sha":"b1"},"commit":{}}`))

	err := client.Create(t.Context(), "Notes/A b#c.md", []byte("# Café\n"))
	if err != nil {
		t.Fatal(err)
	}

	asks := fake.asks()
	if len(asks) != 1 {
		t.Fatalf("got the requests %q, want one", asks)
	}

	got, body := asks[0], asks[0].body
	got.body = ""

	want := asked{
		method: http.MethodPost, path: repoPath + "/contents/Notes/A%20b%23c.md",
		auth: "token " + token, accept: "application/json", contentType: "application/json",
	}
	if got != want {
		t.Errorf("got the request %q, want %q", got, want)
	}

	var sent struct {
		Content []byte `json:"content"`
	}

	err = json.Unmarshal([]byte(body), &sent, json.RejectUnknownMembers(true))
	if err != nil || string(sent.Content) != "# Café\n" {
		t.Errorf("got the body %s (%v), want only the text in base64", body, err)
	}
}

func TestCreateTellsATakenPathAndARefusal(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{
			"a file at the path", http.StatusUnprocessableEntity,
			`{"message":"repository file already exists [path: A.md]"}`, gitea.ErrTaken,
		},
		{
			"a file where a folder would be", http.StatusUnprocessableEntity, `{"message":"a ` +
				`file exists where you’re trying to create a subdirectory [path: A.md]"}`,
			gitea.ErrTaken,
		},
		{
			"a push that lost a race to the branch", http.StatusForbidden, `{"message":"cannot ` +
				`lock ref 'refs/heads/main': is at b3 but expected b2"}`, gitea.ErrRefused,
		},
		{
			"a bot that can't write", http.StatusForbidden, `{"message":"user should have a ` +
				`permission to write to the target branch"}`, gitea.ErrRefused,
		},
	} {
		client, _ := clientOf(t, answer(test.status, test.body))

		err := client.Create(t.Context(), "A.md/B.md", []byte("text"))
		if !errors.Is(err, test.want) {
			t.Errorf("%s: got %v, want %v", test.name, err, test.want)
		}
	}
}

func TestCreateFailsOnAnyOtherAnswer(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
	}{
		{
			"a failure in Gitea", `{"message":"internal"}`, "500 Internal Server Error",
			http.StatusInternalServerError,
		},
		{"an update", `{"content":{"sha":"b2"}}`, "200 OK", http.StatusOK},
		{"a repository that's gone", `{"message":"Not Found"}`, "404 Not Found", http.StatusNotFound},
	} {
		client, _ := clientOf(t, answer(test.status, test.body))

		err := client.Create(t.Context(), "A.md", []byte("text"))
		if err == nil || !strings.Contains(err.Error(), test.want) ||
			errors.Is(err, gitea.ErrTaken) || errors.Is(err, gitea.ErrRefused) ||
			errors.Is(err, gitea.ErrNotFound) {
			t.Errorf("%s: got %v, want an error that says %q", test.name, err, test.want)
		}
	}
}

func TestCreateAtAPathOutsideTheRepositorySendsNothing(t *testing.T) {
	client, fake := clientOf(t, answer(http.StatusCreated, `{}`))

	for _, name := range []string{"../A.md", "", ".", "/A.md", "Notes/./A.md"} {
		if err := client.Create(t.Context(), name, []byte("text")); err == nil {
			t.Errorf("%q: got no error, want one", name)
		}
	}

	if asks := fake.asks(); len(asks) != 0 {
		t.Errorf("got the requests %q, want none", asks)
	}
}
