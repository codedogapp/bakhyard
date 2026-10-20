package spa_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/services/web/internal/spa"
)

const (
	indexBody  = "<!doctype html><title>Bakhyard</title>"
	scriptBody = "console.log('app')"
	iconBody   = "<svg></svg>"
	scriptPath = "/assets/index-D4x9f2Kq.js"
	html       = "text/html; charset=utf-8"
	immutable  = "public, max-age=31536000, immutable"
	revalidate = "no-cache"
)

func built() fstest.MapFS {
	return fstest.MapFS{
		"index.html":                {Data: []byte(indexBody)},
		"assets/index-D4x9f2Kq.js":  {Data: []byte(scriptBody)},
		"assets/fonts/grotesk.woff": {Data: []byte("font")},
		"favicon.svg":               {Data: []byte(iconBody)},
	}
}

func TestServesFiles(t *testing.T) {
	tests := []struct {
		path, body, cache string
	}{
		{"/", indexBody, revalidate},
		{scriptPath, scriptBody, immutable},
		{scriptPath + "?v=2", scriptBody, immutable},
		{"/assets/fonts/grotesk.woff", "font", immutable},
		{"/favicon.svg", iconBody, revalidate},
		{"/x/.." + scriptPath, scriptBody, immutable},
		{"/../favicon.svg", iconBody, revalidate},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			rec := request(t, built(), http.MethodGet, test.path)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			if got := rec.Body.String(); got != test.body {
				t.Errorf("body = %q, want %q", got, test.body)
			}

			if got := rec.Header().Get("Cache-Control"); got != test.cache {
				t.Errorf("Cache-Control = %q, want %q", got, test.cache)
			}
		})
	}
}

func TestServesIndexForPages(t *testing.T) {
	paths := []string{
		"/notes",
		"/notes/",
		"/notes/today",
		"/notes/today?tab=links",
		"/assets",
		"/assets/",
		"/assets/fonts",
		"/notes/../todos",
		"/../../etc/passwd",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec := request(t, built(), http.MethodGet, path)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			if got := rec.Body.String(); got != indexBody {
				t.Errorf("body = %q, want the index page", got)
			}

			if got := rec.Header().Get("Content-Type"); got != html {
				t.Errorf("Content-Type = %q, want %q", got, html)
			}

			if got := rec.Header().Get("Cache-Control"); got != revalidate {
				t.Errorf("Cache-Control = %q, want %q", got, revalidate)
			}
		})
	}
}

func TestMissingFilesAreNotFound(t *testing.T) {
	paths := []string{
		"/missing.png",
		"/assets/missing.js",
		"/assets/fonts/missing.woff2",
		"/notes/today.md",
		// A dot in the last segment reads as an extension, so page paths avoid one.
		"/notes/2026.10",
		"/../missing.png",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec := request(t, built(), http.MethodGet, path)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}

			if got := rec.Header().Get("Cache-Control"); got != "" {
				t.Errorf("Cache-Control = %q, want none", got)
			}
		})
	}
}

func TestRedirectsIndexFileToRoot(t *testing.T) {
	rec := request(t, built(), http.MethodGet, "/index.html")

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}

	if got := rec.Header().Get("Location"); got != "./" {
		t.Errorf("Location = %q, want %q", got, "./")
	}
}

func TestHead(t *testing.T) {
	rec := request(t, built(), http.MethodHead, "/notes/today")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Body.Len(); got != 0 {
		t.Errorf("body length = %d, want 0", got)
	}

	if got := rec.Header().Get("Content-Type"); got != html {
		t.Errorf("Content-Type = %q, want %q", got, html)
	}
}

func TestRejectsOtherMethods(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			rec := request(t, built(), method, "/")

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestNotBuilt(t *testing.T) {
	apps := map[string]fs.FS{
		"empty":            fstest.MapFS{},
		"assets only":      fstest.MapFS{"assets/index-D4x9f2Kq.js": {Data: []byte(scriptBody)}},
		"index is a dir":   fstest.MapFS{"index.html/page": {Data: []byte(indexBody)}},
		"other index file": fstest.MapFS{"notes/index.html": {Data: []byte(indexBody)}},
	}

	for name, app := range apps {
		for _, path := range []string{"/", "/notes/today", scriptPath, "/missing.png"} {
			t.Run(name+" "+path, func(t *testing.T) {
				rec := request(t, app, http.MethodGet, path)

				if rec.Code != http.StatusServiceUnavailable {
					t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
				}

				if got := rec.Body.String(); !strings.Contains(got, "mise run //apps/web:build") {
					t.Errorf("body = %q, want the build hint", got)
				}
			})
		}
	}
}

// request sends a request to a web service that serves app.
func request(t *testing.T, app fs.FS, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Use(middleware.GetHead)
	spa.Register(router, app)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, nil))

	return rec
}
