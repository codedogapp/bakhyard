package gitea_test

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

func TestReadAnswersTheFilesBytesAndBlob(t *testing.T) {
	client, fake := clientOf(t, answer(http.StatusOK, fileAnswer("b1", "# Café\n\nSoon.\n")))

	got, err := client.Read(t.Context(), "Notes/Plans 2026/A b#c?d%e+f é.md")
	if err != nil {
		t.Fatal(err)
	}

	if string(got.Text) != "# Café\n\nSoon.\n" || got.Blob != "b1" {
		t.Errorf("got %q in the blob %q, want the note in b1", got.Text, got.Blob)
	}

	want := []asked{{
		method: http.MethodGet,
		path:   repoPath + "/contents-ext/Notes/Plans%202026/A%20b%23c%3Fd%25e+f%20%C3%A9.md",
		query:  "includes=file_content", auth: "token " + token, accept: "application/json",
	}}
	if asks := fake.asks(); !slices.Equal(asks, want) {
		t.Errorf("got the requests %q, want %q", asks, want)
	}
}

func TestReadAnswersAnEmptyFile(t *testing.T) {
	client, _ := clientOf(t, answer(http.StatusOK, fileAnswer("e69de29", "")))

	got, err := client.Read(t.Context(), "Empty.md")
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Text) != 0 || got.Blob != "e69de29" {
		t.Errorf("got %q in the blob %q, want nothing in e69de29", got.Text, got.Blob)
	}
}

func TestReadOfAFileThatIsntThereIsErrNotFound(t *testing.T) {
	client, _ := clientOf(t, answer(http.StatusNotFound,
		`{"errors":["object does not exist"],"message":"GetContentsOrList","url":"…"}`))

	_, err := client.Read(t.Context(), "Gone.md")
	if !errors.Is(err, gitea.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestReadFailsOnWhatIsntAFilesBytes(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
	}{
		{
			"a folder", `{"dir_contents":[{"name":"A.md","type":"file"}]}`, "it's a folder",
			http.StatusOK,
		},
		{"a link", `{"file_contents":{"type":"symlink","sha":"b2","encoding":null,` +
			`"content":null,"target":"Other.md"}}`, "it's a symlink, not a file", http.StatusOK},
		{
			"a file over Gitea's limit", `{"file_contents":{"type":"file","sha":"b3",` +
				`"size":10485761,"encoding":null,"content":null}}`, "Gitea sent none of its bytes",
			http.StatusOK,
		},
		{"bytes that aren't base64", `{"file_contents":{"type":"file","sha":"b4",` +
			`"encoding":"base64","content":"not base64!"}}`, "read Gitea's answer", http.StatusOK},
		{"an answer that isn't JSON", `<html>`, "read Gitea's answer", http.StatusOK},
		{
			"a token that Gitea refuses", `{"message":"token is required"}`,
			"401 Unauthorized", http.StatusUnauthorized,
		},
		{
			"a failure in Gitea", `{"message":"internal"}`, "500 Internal Server Error",
			http.StatusInternalServerError,
		},
	} {
		client, _ := clientOf(t, answer(test.status, test.body))

		_, err := client.Read(t.Context(), "Notes/A.md")
		if err == nil || !strings.Contains(err.Error(), test.want) ||
			errors.Is(err, gitea.ErrNotFound) {
			t.Errorf("%s: got %v, want an error that says %q", test.name, err, test.want)
		}
	}
}
