// Package spa serves a built single-page app.
package spa

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	indexFile = "index.html"
	// Vite fingerprints the file names in assets/, so a file there never changes.
	assetsDir    = "assets/"
	cacheForever = "public, max-age=31536000, immutable"
	revalidate   = "no-cache"
	notBuilt     = "The web app is not built: run `mise run //apps/web:build`, " +
		"then restart the web service."
)

// Register serves the built app in files on router for GET. A path that names a file gets
// that file; any other path without an extension gets index.html, so the app routes its own
// pages. Paths are cleaned first, as a browser cleans them. Without index.html, every
// request gets a 503.
func Register(router chi.Router, files fs.FS) {
	index, err := fs.ReadFile(files, indexFile)
	if err != nil {
		router.Get("/*", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, notBuilt, http.StatusServiceUnavailable)
		})

		return
	}

	fileServer := http.FileServerFS(files)

	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

		switch {
		case isFile(files, name):
			w.Header().Set("Cache-Control", cacheControl(name))
			fileServer.ServeHTTP(w, r)
		case path.Ext(name) != "":
			http.NotFound(w, r)
		default:
			w.Header().Set("Cache-Control", revalidate)
			http.ServeContent(w, r, indexFile, time.Time{}, bytes.NewReader(index))
		}
	})
}

func cacheControl(name string) string {
	if strings.HasPrefix(name, assetsDir) {
		return cacheForever
	}

	return revalidate
}

// isFile reports whether name is a regular file in files, so directories are never listed.
func isFile(files fs.FS, name string) bool {
	info, err := fs.Stat(files, name)

	return err == nil && info.Mode().IsRegular()
}
