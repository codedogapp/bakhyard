package api_test

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestASyncReadsTheCheckoutThatTheVaultsLinkNowNames(t *testing.T) {
	link := linkTo(t, vaultWith(t, "Note.md", "Old.md").Name())
	router, _ := newRouter(t, link)

	newer := vaultWith(t, "New.md", "Note.md")
	if err := newer.WriteFile("Note.md", []byte("# Newer"), 0o600); err != nil {
		t.Fatal(err)
	}

	swap(t, link, newer.Name())

	// Until a sync, the list and the files it names come from the checkout it was made from.
	expectFiles(t, router, "before the sync", map[string]string{
		"Note.md": text("Note.md"), "Old.md": text("Old.md"), "New.md": "",
	})

	want := `{"files":["New.md","Note.md"]}`
	if rec := postSync(t, router); rec.Code != http.StatusOK ||
		rec.Body.String() != want {
		t.Fatalf("the sync: got %d %q, want 200 %q", rec.Code, rec.Body.String(), want)
	}

	expectFiles(t, router, "after the sync", map[string]string{
		"Note.md": "# Newer", "New.md": text("New.md"), "Old.md": "",
	})
}

func TestASyncThatCantOpenTheVaultKeepsTheList(t *testing.T) {
	link := linkTo(t, vaultWith(t, "Note.md").Name())
	router, log := newRouter(t, link)

	swap(t, link, filepath.Join(t.TempDir(), "Missing"))

	rec := postSync(t, router)
	if cause := "open the vault: open " + link + ": no such file or directory"; rec.Code !=
		http.StatusInternalServerError || !strings.Contains(log.String(), cause) {
		t.Errorf("got %d and the log %q; want 500 and %q logged", rec.Code, log.String(), cause)
	}

	if rec := get(t, router, "/notes", nil); rec.Body.String() != `{"files":["Note.md"]}` {
		t.Errorf("the list after a failed sync: got %q", rec.Body.String())
	}

	expectFiles(t, router, "after a failed sync", map[string]string{"Note.md": text("Note.md")})
}

func TestSyncsLeaveNoFolderOpen(t *testing.T) {
	for _, test := range []struct {
		name string
		lock bool
		code int
	}{
		{"succeeded", false, http.StatusOK},
		{"failed", true, http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := vaultWith(t, "Note.md", "Locked/Inside.md")
			router, _ := newRouter(t, root.Name())

			if test.lock {
				lock(t, root, "Locked")
			}

			// Without a collection, the finalizer of a root that a sync left open can't close it.
			defer debug.SetGCPercent(debug.SetGCPercent(-1))

			const syncs = 20

			before := openFiles(t)
			for range syncs {
				if rec := postSync(t, router); rec.Code != test.code {
					t.Fatalf("sync: got %d, want %d", rec.Code, test.code)
				}
			}

			// A margin, in case the runtime opens a file of its own meanwhile.
			if more := openFiles(t) - before; more >= syncs/2 {
				t.Errorf("%d syncs left %d more files open, want none", syncs, more)
			}
		})
	}
}

// expectFiles checks that router answers each file in want with its text, or with 404 when its
// text is "".
func expectFiles(t *testing.T, router http.Handler, when string, want map[string]string) {
	t.Helper()

	for name, body := range want {
		rec := get(t, router, "/notes/files/"+name, nil)
		if body == "" && rec.Code != http.StatusNotFound {
			t.Errorf("%s %s: got %d, want 404", name, when, rec.Code)
		}

		if body != "" && (rec.Code != http.StatusOK || rec.Body.String() != body) {
			t.Errorf("%s %s: got %d %q, want 200 %q", name, when, rec.Code, rec.Body.String(), body)
		}
	}
}

// linkTo returns a link to the folder target, in a folder of its own, as git-sync links its
// checkout.
func linkTo(t *testing.T, target string) string {
	t.Helper()

	link := filepath.Join(t.TempDir(), "vault")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	return link
}

// swap points link at target as git-sync publishes a checkout: it renames a new link over it.
func swap(t *testing.T, link, target string) {
	t.Helper()

	next := filepath.Join(filepath.Dir(link), "tmp-link")
	if err := os.Symlink(target, next); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(next, link); err != nil {
		t.Fatal(err)
	}
}

// openFiles returns how many files the test's process has open, as /dev/fd lists them.
func openFiles(t *testing.T) int {
	t.Helper()

	entries, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Fatal(err)
	}

	return len(entries)
}
