package api_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestANewNoteGetsTheNameItAsksFor(t *testing.T) {
	for _, test := range []struct{ target, name, text string }{
		{"/notes/files/Plans/2026-10-30-trip.md", "Plans/2026-10-30-trip.md", "# Trip\n"},
		{"/notes/files/Why%3F.md", "Why?.md", "# Why?"},
		{"/notes/files/C%23%20100%25.md", "C# 100%.md", "# C#"},
		{"/notes/files/Caf%C3%A9/%E6%97%A5%E6%9C%AC.md", "Café/日本.md", "# 日本"},
		{"/notes/files/Empty.md", "Empty.md", ""},
	} {
		store := storeWith("Inbox.md")
		router, log := routerWith(t, vaultWith(t, "Inbox.md").Name(), store)

		rec := serve(router, postNote(t, test.target, test.text))

		if rec.Code != http.StatusCreated || pathOf(t, rec) != test.name ||
			rec.Header().Get("Content-Type") != "application/json" {
			t.Errorf("POST %s: got %d %q (%s), want 201 with the path %q", test.target, rec.Code,
				rec.Body.String(), rec.Header().Get("Content-Type"), test.name)
		}

		file, ok := store.files[test.name]
		if !ok || string(file.Text) != test.text ||
			!slices.Equal(store.tried, []string{test.name}) {
			t.Errorf("POST %s: got %q in the store after the tries %q, want %q after one",
				test.target, file.Text, store.tried, test.text)
		}

		if log.Len() != 0 {
			t.Errorf("POST %s: got the log %q, want none", test.target, log.String())
		}
	}
}

func TestANameThatTheStoreTakesGetsTheNextNumber(t *testing.T) {
	// The list doesn't have them yet: git-sync hasn't pulled them.
	store := storeWith("Notes/Idea.md", "Notes/Idea 1.md")
	router, _ := routerWith(t, vaultWith(t, "Inbox.md").Name(), store)

	rec := serve(router, postNote(t, "/notes/files/Notes/Idea.md", "# Mine"))

	want := []string{"Notes/Idea.md", "Notes/Idea 1.md", "Notes/Idea 2.md"}
	if rec.Code != http.StatusCreated || pathOf(t, rec) != want[2] ||
		!slices.Equal(store.tried, want) {
		t.Errorf("got %d %q after the tries %q, want 201 %q after %q", rec.Code,
			rec.Body.String(), store.tried, want[2], want)
	}

	if got := string(store.files["Notes/Idea.md"].Text); got != text("Notes/Idea.md") {
		t.Errorf("got %q in the note that was there, want it as it was", got)
	}
}

func TestANameOnTheListInAnyCaseIsntTried(t *testing.T) {
	store := storeWith()
	router, _ := routerWith(t, vaultWith(t, "Inbox.md", "INBOX 1.md").Name(), store)

	rec := serve(router, postNote(t, "/notes/files/inbox.md", "# Mine"))

	if rec.Code != http.StatusCreated || pathOf(t, rec) != "inbox 2.md" ||
		!slices.Equal(store.tried, []string{"inbox 2.md"}) {
		t.Errorf("got %d %q after the tries %q, want 201 inbox 2.md after it alone", rec.Code,
			rec.Body.String(), store.tried)
	}
}

func TestANewNoteTriesTenNamesThatTheListLeavesFree(t *testing.T) {
	// The list's names, A.md to A 11.md, don't count as tries, so A 12.md to A 21.md are the ten.
	listed := numberedNames("A", 12)
	tries := numberedNames("A", 22)[12:]

	for _, test := range []struct {
		taken, code int
		path        string
	}{
		{9, http.StatusCreated, "A 21.md"},
		{10, http.StatusConflict, ""},
	} {
		store := storeWith(numberedNames("A", 12+test.taken)...)
		router, log := routerWith(t, vaultWith(t, listed...).Name(), store)

		rec := serve(router, postNote(t, "/notes/files/A.md", "# Mine"))

		if rec.Code != test.code || pathOf(t, rec) != test.path ||
			!slices.Equal(store.tried, tries) || log.Len() != 0 {
			t.Errorf("%d more taken: got %d %q after the tries %q, want %d %q after %q",
				test.taken, rec.Code, rec.Body.String(), store.tried, test.code, test.path, tries)
		}
	}
}

// postNote returns POST target with content as its body.
func postNote(t *testing.T, target, content string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, target,
		strings.NewReader(content))
	req.Header.Set("Content-Type", "text/markdown; charset=utf-8")

	return req
}

// pathOf returns the path in a new note's answer, or "" when it has none.
func pathOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var made struct {
		Path string `json:"path"`
	}

	if json.Unmarshal(rec.Body.Bytes(), &made, json.RejectUnknownMembers(true)) != nil {
		return ""
	}

	return made.Path
}

// numberedNames returns the first n names that a new note at base.md tries: base.md, base 1.md
// and so on.
func numberedNames(base string, n int) []string {
	names := []string{base + ".md"}
	for i := 1; i < n; i++ {
		names = append(names, base+" "+strconv.Itoa(i)+".md")
	}

	return names
}
