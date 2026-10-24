package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestASyncPicksUpTheVaultAsItIsNow(t *testing.T) {
	router := changedVault(t)

	if rec := get(t, router, "/notes/files/New.md", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("New.md before the sync: got %d, want 404", rec.Code)
	}

	want := `{"files":["New.md"]}`
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"the sync":          post(t, router, "/notes/sync"),
		"the list after it": get(t, router, "/notes", nil),
	} {
		if rec.Code != http.StatusOK || rec.Body.String() != want ||
			rec.Header().Get("Content-Type") != "application/json" {
			t.Errorf("%s: got %d %q %q, want 200 application/json %q", name, rec.Code,
				rec.Header().Get("Content-Type"), rec.Body.String(), want)
		}
	}

	if rec := get(t, router, "/notes/files/New.md", nil); rec.Code != http.StatusOK ||
		rec.Body.String() != text("New.md") {
		t.Errorf("New.md after the sync: got %d %q, want 200", rec.Code, rec.Body.String())
	}

	if rec := get(t, router, "/notes/files/Old.md", nil); rec.Code != http.StatusNotFound {
		t.Errorf("Old.md after the sync: got %d, want 404", rec.Code)
	}
}

func TestAFailedSyncKeepsTheList(t *testing.T) {
	root := vaultWith(t, "Note.md", "Locked/Inside.md")
	router, log := newRouter(t, root, []string{"Locked/Inside.md", "Note.md"})

	if err := root.Chmod("Locked", 0); err != nil {
		t.Fatal(err)
	}
	// Runs before the vault's own cleanup, which must read the folder to remove it.
	t.Cleanup(func() {
		if err := root.Chmod("Locked", 0o700); err != nil {
			t.Error(err)
		}
	})

	rec := post(t, router, "/notes/sync")
	if rec.Code != http.StatusInternalServerError ||
		strings.Contains(rec.Body.String(), "Locked") ||
		!strings.Contains(log.String(), "Locked: permission denied") {
		t.Errorf("got %d %q and the log %q; want 500, the cause logged and not sent", rec.Code,
			rec.Body.String(), log.String())
	}

	want := `{"files":["Locked/Inside.md","Note.md"]}`
	if rec := get(t, router, "/notes", nil); rec.Body.String() != want {
		t.Errorf("the list after a failed sync: got %q, want %q", rec.Body.String(), want)
	}

	if rec := get(t, router, "/notes/files/Note.md", nil); rec.Code != http.StatusOK {
		t.Errorf("Note.md after a failed sync: got %d, want 200", rec.Code)
	}
}

func TestSyncsAndReadsCanOverlap(t *testing.T) {
	router, _ := newRouter(t, vaultWith(t, "Note.md"), []string{"Note.md"})

	var requests sync.WaitGroup
	for range 10 {
		requests.Go(func() {
			if rec := post(t, router, "/notes/sync"); rec.Code != http.StatusOK {
				t.Errorf("sync: got %d, want 200", rec.Code)
			}
		})
		requests.Go(func() {
			if rec := get(t, router, "/notes", nil); rec.Code != http.StatusOK {
				t.Errorf("list: got %d, want 200", rec.Code)
			}
		})
		requests.Go(func() {
			if rec := get(t, router, "/notes/files/Note.md", nil); rec.Code != http.StatusOK {
				t.Errorf("file: got %d, want 200", rec.Code)
			}
		})
	}

	requests.Wait()
}

// changedVault returns a router whose list names Old.md, over a vault that has since swapped it
// for New.md.
func changedVault(t *testing.T) http.Handler {
	t.Helper()

	root := vaultWith(t, "Old.md")
	router, _ := newRouter(t, root, []string{"Old.md"})

	if err := root.WriteFile("New.md", []byte(text("New.md")), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := root.Remove("Old.md"); err != nil {
		t.Fatal(err)
	}

	return router
}

// post sends POST target to router, and returns the answer.
func post(t *testing.T, router http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, target, nil))

	return rec
}
