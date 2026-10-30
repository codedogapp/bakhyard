package api_test

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

func TestASaveIsRefused(t *testing.T) {
	current := versionOf(text("Note.md"))

	for _, test := range []struct {
		name, target, ifMatch, text string
		change                      func(*fakeStore)
		want                        int
	}{
		{
			"a file that isn't on the list", "/notes/files/Gone.md", versionOf(text("Gone.md")),
			"# Saved", nil, http.StatusNotFound,
		},
		{
			"a file that isn't a note", "/notes/files/img/photo.png",
			versionOf(text("img/photo.png")), "# Saved", nil, http.StatusNotFound,
		},
		{
			"a note in a dot folder", "/notes/files/.obsidian/Template.md",
			versionOf(text(".obsidian/Template.md")), "# Saved", nil, http.StatusNotFound,
		},
		{
			"no If-Match", "/notes/files/Note.md", "", "# Saved", nil,
			http.StatusPreconditionRequired,
		},
		{
			"another version", "/notes/files/Note.md", versionOf("# Old"), "# Saved", nil,
			http.StatusPreconditionFailed,
		},
		{
			"any version", "/notes/files/Note.md", "*", "# Saved", nil,
			http.StatusPreconditionFailed,
		},
		{
			"a weak version", "/notes/files/Note.md", "W/" + current, "# Saved", nil,
			http.StatusPreconditionFailed,
		},
		{
			"a list of versions", "/notes/files/Note.md", current + ", " + versionOf("# Old"),
			"# Saved", nil, http.StatusPreconditionFailed,
		},
		{
			"a note changed in the store", "/notes/files/Note.md", current, "# Saved",
			func(s *fakeStore) { s.files["Note.md"] = gitea.File{Text: []byte("# New")} },
			http.StatusPreconditionFailed,
		},
		{
			"a note gone from the store", "/notes/files/Note.md", current, "# Saved",
			func(s *fakeStore) { delete(s.files, "Note.md") }, http.StatusPreconditionFailed,
		},
		{
			"a note changed before the update", "/notes/files/Note.md", current, "# Saved",
			func(s *fakeStore) { s.updateFails = gitea.ErrChanged },
			http.StatusPreconditionFailed,
		},
		{
			"a note gone before the update", "/notes/files/Note.md", current, "# Saved",
			func(s *fakeStore) { s.updateFails = gitea.ErrNotFound },
			http.StatusPreconditionFailed,
		},
		{
			"a text over 10 MiB", "/notes/files/Note.md", current,
			strings.Repeat("a", 10<<20+1), nil, http.StatusRequestEntityTooLarge,
		},
		{
			"a text that isn't UTF-8", "/notes/files/Note.md", current, "# Caf\xe9", nil,
			http.StatusBadRequest,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			names := []string{"Note.md", "img/photo.png", ".obsidian/Template.md"}
			store := storeWith(append(names, "Gone.md")...)
			router, _ := routerWith(t, vaultWith(t, names...).Name(), store)

			if test.change != nil {
				test.change(store)
			}

			before := maps.Clone(store.files)
			rec := serve(router, putNote(t, test.target, test.ifMatch, test.text))

			if rec.Code != test.want || store.commits != 0 ||
				!maps.EqualFunc(store.files, before, sameFile) {
				t.Errorf("got %d and %d commits, want %d and none", rec.Code, store.commits,
					test.want)
			}
		})
	}
}

func TestARefusedUpdateIsTriedOnceMore(t *testing.T) {
	for _, test := range []struct {
		name string
		race func(*fakeStore)
		want int
		text string
	}{
		{
			"a commit of another note", func(s *fakeStore) {
				s.files["Other.md"] = gitea.File{Text: []byte("# Other"), Blob: "other"}
			}, http.StatusNoContent, "# Saved",
		},
		{
			"a commit of the note", func(s *fakeStore) {
				s.files["Note.md"] = gitea.File{Text: []byte("# Theirs"), Blob: "theirs"}
			}, http.StatusPreconditionFailed, "# Theirs",
		},
		{
			"a commit that removes the note", func(s *fakeStore) { delete(s.files, "Note.md") },
			http.StatusPreconditionFailed, "",
		},
	} {
		t.Run("after "+test.name, func(t *testing.T) {
			store := storeWith("Note.md")
			store.race = test.race
			router, log := routerWith(t, vaultWith(t, "Note.md").Name(), store)

			rec := serve(router,
				putNote(t, "/notes/files/Note.md", versionOf(text("Note.md")), "# Saved"))

			got := string(store.files["Note.md"].Text)
			if rec.Code != test.want || got != test.text || log.Len() != 0 {
				t.Errorf("got %d, %q in the store and the log %q; want %d, %q and no log",
					rec.Code, got, log.String(), test.want, test.text)
			}
		})
	}
}

func TestAStoreThatFailsIsABadGateway(t *testing.T) {
	down := errors.New("Gitea is down")

	for _, test := range []struct {
		name string
		fail func(*fakeStore)
		err  error
	}{
		{"the read", func(s *fakeStore) { s.readFails = down }, down},
		{"the update", func(s *fakeStore) { s.updateFails = down }, down},
		{
			"an update refused, and its second try",
			func(s *fakeStore) { s.updateFails = gitea.ErrRefused }, gitea.ErrRefused,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := storeWith("Note.md")
			test.fail(store)
			router, log := routerWith(t, vaultWith(t, "Note.md").Name(), store)

			rec := serve(router,
				putNote(t, "/notes/files/Note.md", versionOf(text("Note.md")), "# Saved"))

			want := fmt.Sprintf("msg=%q err=%q", "save a note", test.err.Error())
			if rec.Code != http.StatusBadGateway || !strings.Contains(log.String(), want) {
				t.Errorf("got %d and the log %q, want 502 and %s", rec.Code, log.String(), want)
			}
		})
	}
}

// sameFile reports whether a and b hold the same text and blob.
func sameFile(a, b gitea.File) bool { return string(a.Text) == string(b.Text) && a.Blob == b.Blob }
