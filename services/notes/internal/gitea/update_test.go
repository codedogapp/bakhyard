package gitea_test

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

func TestUpdateCommitsTheTextInPlaceOfTheBlob(t *testing.T) {
	client, fake := clientOf(t, answer(http.StatusOK, `{"content":{"sha":"b2"},"commit":{}}`))

	err := client.Update(t.Context(), "Notes/A b#c.md", "b1", []byte("# Café\n"))
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
		method: http.MethodPut, path: repoPath + "/contents/Notes/A%20b%23c.md",
		auth: "token " + token, accept: "application/json", contentType: "application/json",
	}
	if got != want {
		t.Errorf("got the request %q, want %q", got, want)
	}

	var sent struct {
		Content []byte `json:"content"`
		SHA     string `json:"sha"`
	}

	err = json.Unmarshal([]byte(body), &sent, json.RejectUnknownMembers(true))
	if err != nil || string(sent.Content) != "# Café\n" || sent.SHA != "b1" {
		t.Errorf("got the body %s (%v), want the text in base64 in place of b1", body, err)
	}
}

func TestUpdateTellsAChangedFileAGoneOneAndARefusal(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{
			"a stale blob", http.StatusUnprocessableEntity,
			`{"message":"sha does not match [given: b1, expected: b9]"}`, gitea.ErrChanged,
		},
		{
			"a file that's gone", http.StatusNotFound,
			`{"message":"repository file does not exist [path: A.md]"}`, gitea.ErrNotFound,
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

		err := client.Update(t.Context(), "A.md", "b1", []byte("text"))
		if !errors.Is(err, test.want) {
			t.Errorf("%s: got %v, want %v", test.name, err, test.want)
		}
	}
}

func TestUpdateFailsOnAnyOtherAnswer(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
	}{
		{
			"a failure in Gitea", `{"message":"internal"}`, "500 Internal Server Error",
			http.StatusInternalServerError,
		},
		{"a file that Gitea made", `{"content":{"sha":"b2"}}`, "201 Created", http.StatusCreated},
	} {
		client, _ := clientOf(t, answer(test.status, test.body))

		err := client.Update(t.Context(), "A.md", "b1", []byte("text"))
		if err == nil || !strings.Contains(err.Error(), test.want) ||
			errors.Is(err, gitea.ErrChanged) || errors.Is(err, gitea.ErrNotFound) ||
			errors.Is(err, gitea.ErrRefused) {
			t.Errorf("%s: got %v, want an error that says %q", test.name, err, test.want)
		}
	}
}

func TestUpdateDoesntFollowARedirect(t *testing.T) {
	client, fake := clientOf(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			http.Redirect(w, r, "/elsewhere", http.StatusFound)

			return
		}

		w.WriteHeader(http.StatusOK)
	})

	err := client.Update(t.Context(), "A.md", "b1", []byte("text"))
	if err == nil || !strings.Contains(err.Error(), "302 Found") {
		t.Errorf("got %v, want an error that says 302 Found", err)
	}

	if asks := fake.asks(); len(asks) != 1 {
		t.Errorf("got the requests %q, want only the PUT", asks)
	}
}

func TestUpdateWithoutABlobSendsNothing(t *testing.T) {
	client, fake := clientOf(t, answer(http.StatusCreated, `{}`))

	err := client.Update(t.Context(), "A.md", "", []byte("text"))
	if err == nil || !strings.Contains(err.Error(), "no blob to replace") {
		t.Errorf("got %v, want an error that says there's no blob", err)
	}

	if asks := fake.asks(); len(asks) != 0 {
		t.Errorf("got the requests %q, want none", asks)
	}
}
