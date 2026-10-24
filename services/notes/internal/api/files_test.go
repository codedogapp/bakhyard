package api_test

import (
	"bytes"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/notes/internal/api"
)

func TestAListedFileComesWithItsHeaders(t *testing.T) {
	for name, wantType := range map[string]string{
		"Daily/Today.md": "text/markdown; charset=utf-8",
		"img/photo.png":  "image/png",
	} {
		t.Run(name, func(t *testing.T) {
			router, _ := newRouter(t, vaultWith(t, name), []string{name})
			rec := get(t, router, "/notes/files/"+name, nil)

			if rec.Code != http.StatusOK || rec.Body.String() != text(name) {
				t.Fatalf("got %d %q, want 200 %q", rec.Code, rec.Body.String(), text(name))
			}

			for header, want := range map[string]string{
				"Content-Type":            wantType,
				"Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'; sandbox",
				"X-Content-Type-Options":  "nosniff",
				"Cache-Control":           "private, no-cache",
			} {
				if got := rec.Header().Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}

			if rec.Header().Get("Last-Modified") == "" {
				t.Error("no Last-Modified")
			}
		})
	}
}

func TestEscapedNamesAreServed(t *testing.T) {
	names := []string{"With space.md", "Why?.md", "C#.md", "100%.md", "I'm, here.md", "Café/日本.md"}
	router, _ := newRouter(t, vaultWith(t, names...), names)

	for target, name := range map[string]string{
		"/notes/files/With%20space.md": "With space.md",
		"/notes/files/Why%3F.md":       "Why?.md",
		"/notes/files/C%23.md":         "C#.md",
		"/notes/files/100%25.md":       "100%.md",
		// As browsers send it, and escaped in full.
		"/notes/files/I'm,%20here.md":                  "I'm, here.md",
		"/notes/files/I%27m%2C%20here.md":              "I'm, here.md",
		"/notes/files/Caf%C3%A9/%E6%97%A5%E6%9C%AC.md": "Café/日本.md",
	} {
		rec := get(t, router, target, nil)
		if rec.Code != http.StatusOK || rec.Body.String() != text(name) {
			t.Errorf("GET %s: got %d %q, want 200 %q", target, rec.Code, rec.Body.String(),
				text(name))
		}
	}
}

func TestOnlyListedRegularFilesAreServed(t *testing.T) {
	root := vaultWith(t, "Note.md", "Gone.md", "Moved.md", "Unlisted.md", ".obsidian/app.json",
		"Daily/Today.md")
	if err := root.Symlink("Note.md", "Link.md"); err != nil {
		t.Fatal(err)
	}

	router, _ := newRouter(t, root, []string{"Note.md", "Gone.md", "Moved.md", "Daily/Today.md"})

	// After the scan, one listed file is deleted and another becomes a folder.
	if err := root.Remove("Gone.md"); err != nil {
		t.Fatal(err)
	}

	if err := root.Remove("Moved.md"); err != nil {
		t.Fatal(err)
	}

	if err := root.Mkdir("Moved.md", 0o700); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		".obsidian/app.json", "Unlisted.md", "Link.md", "Daily", "Daily/", "",
		"Daily/../Note.md", "../Note.md", "/Note.md", "Gone.md", "Moved.md",
	} {
		if rec := get(t, router, "/notes/files/"+name, nil); rec.Code != http.StatusNotFound {
			t.Errorf("GET %q: got %d, want 404", name, rec.Code)
		}
	}
}

func TestAFileThatCantBeReadIsAServerError(t *testing.T) {
	root := vaultWith(t, "Locked.md")
	router, log := newRouter(t, root, []string{"Locked.md"})

	if err := root.Chmod("Locked.md", 0); err != nil {
		t.Fatal(err)
	}

	rec := get(t, router, "/notes/files/Locked.md", nil)
	if rec.Code != http.StatusInternalServerError ||
		strings.Contains(rec.Body.String(), "Locked") ||
		!strings.Contains(log.String(), "Locked.md: permission denied") {
		t.Errorf("got %d %q and the log %q; want 500, the cause logged and not sent", rec.Code,
			rec.Body.String(), log.String())
	}
}

func TestAFileAnswersConditionalAndRangeRequests(t *testing.T) {
	router, _ := newRouter(t, vaultWith(t, "Note.md"), []string{"Note.md"})
	modified := get(t, router, "/notes/files/Note.md", nil).Header().Get("Last-Modified")

	for _, test := range []struct {
		name   string
		header http.Header
		code   int
		body   string
	}{
		{"unchanged", http.Header{"If-Modified-Since": {modified}}, http.StatusNotModified, ""},
		{"range", http.Header{"Range": {"bytes=0-2"}}, http.StatusPartialContent, "# N"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := get(t, router, "/notes/files/Note.md", test.header)
			if rec.Code != test.code || rec.Body.String() != test.body {
				t.Errorf("got %d %q, want %d %q", rec.Code, rec.Body.String(), test.code, test.body)
			}
		})
	}
}

// vaultWith returns a vault, opened as a root, that holds the files names, each with text(name).
func vaultWith(t *testing.T, names ...string) *os.Root {
	t.Helper()

	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})

	for _, name := range names {
		if err := root.MkdirAll(path.Dir(name), 0o700); err != nil {
			t.Fatal(err)
		}

		if err := root.WriteFile(name, []byte(text(name)), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

// text is what the file name holds in a vault from vaultWith.
func text(name string) string { return "# " + name }

// newRouter returns a router that serves root with the list files, and the log it writes.
func newRouter(t *testing.T, root *os.Root, files []string) (http.Handler, *bytes.Buffer) {
	t.Helper()

	var log bytes.Buffer

	router := chi.NewRouter()
	api.Register(router, root, files, slog.New(slog.NewTextHandler(&log, nil)))

	return router, &log
}

// get sends GET target with header to router, and returns the answer.
func get(
	t *testing.T,
	router http.Handler,
	target string,
	header http.Header,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	maps.Copy(req.Header, header)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
