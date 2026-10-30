package api_test

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

func TestARefusedNewNoteIsTriedOnceMore(t *testing.T) {
	for _, test := range []struct {
		name  string
		race  func(*fakeStore)
		path  string
		tries []string
	}{
		{
			"a commit of another note", func(s *fakeStore) {
				s.files["Other.md"] = gitea.File{Text: []byte("# Other"), Blob: "other"}
			}, "New.md", []string{"New.md", "New.md"},
		},
		{
			"a commit of a note at the name", func(s *fakeStore) {
				s.files["New.md"] = gitea.File{Text: []byte("# Theirs"), Blob: "theirs"}
			}, "New 1.md", []string{"New.md", "New.md", "New 1.md"},
		},
	} {
		t.Run("after "+test.name, func(t *testing.T) {
			store := storeWith()
			store.race = test.race
			router, log := routerWith(t, vaultWith(t, "Inbox.md").Name(), store)

			rec := serve(router, postNote(t, "/notes/files/New.md", "# Mine"))

			if rec.Code != http.StatusCreated || pathOf(t, rec) != test.path ||
				!slices.Equal(store.tried, test.tries) || log.Len() != 0 {
				t.Errorf("got %d %q after the tries %q and the log %q; want 201 %q after %q",
					rec.Code, rec.Body.String(), store.tried, log.String(), test.path, test.tries)
			}
		})
	}
}

func TestANewNoteThatTheStoreFailsIsABadGateway(t *testing.T) {
	down := errors.New("Gitea is down")

	for _, test := range []struct {
		name  string
		err   error
		tries []string
	}{
		{"a failure", down, []string{"New.md"}},
		{"a refusal, and its second try", gitea.ErrRefused, []string{"New.md", "New.md"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := storeWith()
			store.createFails = test.err
			router, log := routerWith(t, vaultWith(t, "Inbox.md").Name(), store)

			rec := serve(router, postNote(t, "/notes/files/New.md", "# Mine"))

			want := fmt.Sprintf("msg=%q err=%q", "make a note", test.err.Error())
			if rec.Code != http.StatusBadGateway || !strings.Contains(log.String(), want) ||
				!slices.Equal(store.tried, test.tries) {
				t.Errorf("got %d and the log %q after the tries %q, want 502 and %s after %q",
					rec.Code, log.String(), store.tried, want, test.tries)
			}
		})
	}
}

func TestOnlyANoteThatTheVaultCouldListIsMade(t *testing.T) {
	store := storeWith()
	router, _ := routerWith(t, vaultWith(t, "Inbox.md").Name(), store)

	for _, target := range []string{
		"/notes/files/", "/notes/files/New.txt", "/notes/files/New", "/notes/files/New.MD",
		"/notes/files/.md", "/notes/files/.obsidian/New.md", "/notes/files/Notes/.New.md",
		"/notes/files/%FF.md", "/notes/files/Caf%E9/New.md", "/notes/files/a/../New.md",
		"/notes/files/./New.md", "/notes/files/a//New.md", "/notes/files/%2FNew.md",
		"/notes/files/" + strings.Repeat("a", 253) + ".md",
	} {
		rec := serve(router, postNote(t, target, "# Mine"))

		if rec.Code != http.StatusNotFound {
			t.Errorf("POST %s: got %d, want 404", target, rec.Code)
		}
	}

	if len(store.tried) != 0 {
		t.Errorf("got the tries %q, want none", store.tried)
	}
}

func TestATakenNameWithNoRoomForANumberIsAConflict(t *testing.T) {
	// 255 bytes, the most that the vault's checkout takes, so "… 1.md" won't fit.
	name := strings.Repeat("a", 252) + ".md"
	store := storeWith(name)
	router, log := routerWith(t, vaultWith(t, "Inbox.md").Name(), store)

	rec := serve(router, postNote(t, "/notes/files/"+name, "# Mine"))

	if rec.Code != http.StatusConflict || !slices.Equal(store.tried, []string{name}) ||
		log.Len() != 0 {
		t.Errorf("got %d after %d tries and the log %q, want 409 after one try", rec.Code,
			len(store.tried), log.String())
	}
}

func TestANewNoteTakesUpTo10MiBOfUTF8(t *testing.T) {
	for _, test := range []struct {
		name, text    string
		code, commits int
	}{
		{"10 MiB", strings.Repeat("a", 10<<20), http.StatusCreated, 1},
		{"over 10 MiB", strings.Repeat("a", 10<<20+1), http.StatusRequestEntityTooLarge, 0},
		{"bytes that aren't UTF-8", "# Caf\xe9", http.StatusBadRequest, 0},
	} {
		store := storeWith()
		router, _ := routerWith(t, vaultWith(t, "Inbox.md").Name(), store)

		rec := serve(router, postNote(t, "/notes/files/New.md", test.text))

		if rec.Code != test.code || store.commits != test.commits {
			t.Errorf("%s: got %d and %d commits, want %d and %d", test.name, rec.Code,
				store.commits, test.code, test.commits)
		}
	}
}
