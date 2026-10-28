package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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
		"the sync":          postSync(t, router),
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
	router, log := newRouter(t, root.Name())
	lock(t, root, "Locked")

	rec := postSync(t, router)
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
	router, _ := newRouter(t, vaultWith(t, "Note.md").Name())
	syncing, stop := context.WithCancel(t.Context())

	// The reads go on until the syncs end, so that every swap and close lands amid them.
	var reads sync.WaitGroup
	for range 4 {
		reads.Go(func() { readUntil(syncing, t, router) })
	}

	// Two at a time, so that the syncs overlap too.
	var syncs sync.WaitGroup
	for range 2 {
		syncs.Go(func() {
			for range 200 {
				if rec := postSync(t, router); rec.Code != http.StatusOK {
					t.Errorf("sync: got %d, want 200", rec.Code)
				}
			}
		})
	}

	syncs.Wait()
	stop()
	reads.Wait()
}

// readUntil reads the list and a file from router until ctx ends, or until a read fails.
func readUntil(ctx context.Context, t *testing.T, router http.Handler) {
	t.Helper()

	for ctx.Err() == nil {
		for _, target := range []string{"/notes", "/notes/files/Note.md"} {
			if rec := get(t, router, target, nil); rec.Code != http.StatusOK {
				t.Errorf("GET %s: got %d, want 200", target, rec.Code)

				return
			}
		}
	}
}

// changedVault returns a router whose list names Old.md, over a vault that has since swapped it
// for New.md.
func changedVault(t *testing.T) http.Handler {
	t.Helper()

	root := vaultWith(t, "Old.md")
	router, _ := newRouter(t, root.Name())

	if err := root.WriteFile("New.md", []byte(text("New.md")), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := root.Remove("Old.md"); err != nil {
		t.Fatal(err)
	}

	return router
}

// lock makes the folder name in root unreadable until the test ends.
func lock(t *testing.T, root *os.Root, name string) {
	t.Helper()

	if err := root.Chmod(name, 0); err != nil {
		t.Fatal(err)
	}
	// Runs before the vault's own cleanup, which must read the folder to remove it.
	t.Cleanup(func() {
		if err := root.Chmod(name, 0o700); err != nil {
			t.Error(err)
		}
	})
}

// postSync sends POST /notes/sync to router, and returns the answer.
func postSync(t *testing.T, router http.Handler) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec,
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/notes/sync", nil))

	return rec
}
