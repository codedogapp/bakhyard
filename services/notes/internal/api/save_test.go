package api_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestASaveCommitsTheNoteInPlaceOfTheBlobItRead(t *testing.T) {
	names := []string{"Daily/Today.md", "Why?.md", "C#.md", "100%.md", "Café/日本.md"}
	store := storeWith(names...)
	router, _ := routerWith(t, vaultWith(t, names...).Name(), store)

	for target, name := range map[string]string{
		"/notes/files/Daily/Today.md":                  "Daily/Today.md",
		"/notes/files/Why%3F.md":                       "Why?.md",
		"/notes/files/C%23.md":                         "C#.md",
		"/notes/files/100%25.md":                       "100%.md",
		"/notes/files/Caf%C3%A9/%E6%97%A5%E6%9C%AC.md": "Café/日本.md",
	} {
		saved := "# Saved\n\n" + name
		rec := serve(router, putNote(t, target, versionOf(text(name)), saved))

		if rec.Code != http.StatusNoContent || rec.Header().Get("ETag") != versionOf(saved) {
			t.Errorf("PUT %s: got %d, ETag %s; want 204, ETag %s", target, rec.Code,
				rec.Header().Get("ETag"), versionOf(saved))
		}

		if got := string(store.files[name].Text); got != saved {
			t.Errorf("%s in the store: got %q, want %q", name, got, saved)
		}
	}

	if store.commits != len(names) {
		t.Errorf("got %d commits, want %d", store.commits, len(names))
	}
}

func TestANoteOfUpTo10MiBIsSaved(t *testing.T) {
	store := storeWith("Note.md")
	router, _ := routerWith(t, vaultWith(t, "Note.md").Name(), store)

	saved := strings.Repeat("a", 10<<20)
	rec := serve(router, putNote(t, "/notes/files/Note.md", versionOf(text("Note.md")), saved))

	if rec.Code != http.StatusNoContent || len(store.files["Note.md"].Text) != len(saved) {
		t.Errorf("got %d and %d bytes in the store, want 204 and %d", rec.Code,
			len(store.files["Note.md"].Text), len(saved))
	}
}

func TestATextTheStoreHoldsAlreadyIsNotCommitted(t *testing.T) {
	store := storeWith("Note.md")
	router, _ := routerWith(t, vaultWith(t, "Note.md").Name(), store)
	current := versionOf(text("Note.md"))

	// Whatever the version, as RFC 9110 allows for a change made already.
	for _, version := range []string{current, versionOf("# Old"), "*"} {
		rec := serve(router, putNote(t, "/notes/files/Note.md", version, text("Note.md")))

		if rec.Code != http.StatusNoContent || rec.Header().Get("ETag") != current ||
			store.commits != 0 {
			t.Errorf("If-Match %s: got %d, ETag %s, %d commits; want 204, ETag %s, none",
				version, rec.Code, rec.Header().Get("ETag"), store.commits, current)
		}
	}
}

func TestTheListSaysWhetherNotesAreSaved(t *testing.T) {
	vault := vaultWith(t, "Inbox.md").Name()
	saving, _ := routerWith(t, vault, storeWith("Inbox.md"))
	reading, _ := newRouter(t, vault)

	for router, want := range map[http.Handler]string{
		saving:  `{"files":["Inbox.md"],"editable":true}`,
		reading: `{"files":["Inbox.md"]}`,
	} {
		for _, rec := range []*httptest.ResponseRecorder{
			get(t, router, "/notes", nil), postSync(t, router),
		} {
			if rec.Code != http.StatusOK || rec.Body.String() != want {
				t.Errorf("got %d %q, want 200 %q", rec.Code, rec.Body.String(), want)
			}
		}
	}
}

func TestWithAStoreAFileTakesGetPutAndPost(t *testing.T) {
	router, _ := routerWith(t, vaultWith(t, "Inbox.md").Name(), storeWith("Inbox.md"))

	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		rec := serve(router,
			httptest.NewRequestWithContext(t.Context(), method, "/notes/files/Inbox.md", nil))

		allow := slices.Sorted(slices.Values(rec.Header().Values("Allow")))
		if rec.Code != http.StatusMethodNotAllowed ||
			!slices.Equal(allow, []string{"GET", "POST", "PUT"}) {
			t.Errorf("%s: got %d, Allow %q; want 405, Allow GET, POST and PUT", method, rec.Code,
				allow)
		}
	}
}
