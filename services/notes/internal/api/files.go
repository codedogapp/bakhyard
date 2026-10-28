package api

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
)

// filesPath starts the path of every file that GET /notes/files/<path> returns.
const filesPath = "/notes/files/"

// fileCSP keeps a file opened in the browser from running a script or loading anything, since
// vault files share the app's origin (decision 92).
const fileCSP = "default-src 'none'; style-src 'unsafe-inline'; sandbox"

// serveFile returns the handler of GET /notes/files/<path>: it answers the file at <path>, read
// through the root of idx's latest scan, if that scan lists it, and 404 otherwise.
func serveFile(idx *index, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The decoded path, so that a name escaped in the URL, such as %3F for ?, matches the list.
		name := strings.TrimPrefix(r.URL.Path, filesPath)

		file, info, err := idx.open(name)
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)

			return
		}

		if err != nil {
			logger.ErrorContext(r.Context(), "read a vault file", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}
		// The file is only read, so closing it can't lose anything.
		defer func() { _ = file.Close() }()

		setFileHeaders(w.Header(), name)
		http.ServeContent(w, r, name, info.ModTime(), file)
	}
}

// open opens the file name in root, and fails with fs.ErrNotExist unless it's a regular file. The
// file is only read, so closing it on an error path can't lose anything.
func open(root *os.Root, name string) (*os.File, fs.FileInfo, error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, nil, err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, nil, err
	}

	if !info.Mode().IsRegular() {
		_ = file.Close()

		return nil, nil, fmt.Errorf("open %s: not a regular file: %w", name, fs.ErrNotExist)
	}

	return file, info, nil
}

// setFileHeaders sets the headers that every file gets (decision 92).
func setFileHeaders(header http.Header, name string) {
	header.Set("Content-Security-Policy", fileCSP)
	header.Set("X-Content-Type-Options", "nosniff")
	// The browser asks every time, so a file is fresh after a Sync; Last-Modified makes that a 304.
	header.Set("Cache-Control", "private, no-cache")

	// Go's type for .md depends on the host's MIME files, so notes get theirs here.
	if path.Ext(name) == ".md" {
		header.Set("Content-Type", "text/markdown; charset=utf-8")
	}
}
