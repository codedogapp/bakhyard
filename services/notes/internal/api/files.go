package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// filesPath starts the path of every file that GET /notes/files/<path> returns.
const filesPath = "/notes/files/"

// fileCSP keeps a file opened in the browser from running a script or loading anything, since
// vault files share the app's origin (decision 92).
const fileCSP = "default-src 'none'; style-src 'unsafe-inline'; sandbox"

// serveFile returns the handler of GET /notes/files/<path>: it answers the file at <path>, read
// through the root of idx's latest scan, if that scan lists it, and 404 otherwise. A note comes
// with its version as its ETag (decision 400).
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
			failRead(w, r, logger, err)

			return
		}
		// The file is only read, so closing it can't lose anything.
		defer func() { _ = file.Close() }()

		body, version, err := bodyOf(name, file)
		if err != nil {
			failRead(w, r, logger, err)

			return
		}

		setFileHeaders(w.Header(), name, version)
		http.ServeContent(w, r, name, info.ModTime(), body)
	}
}

// bodyOf returns what GET answers for the vault file name, opened as file: a note read whole, so
// that its version names the very bytes sent, with that version; any other file as it is, with no
// version.
func bodyOf(name string, file *os.File) (io.ReadSeeker, string, error) {
	if !isNote(name) {
		return file, "", nil
	}

	text, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", name, err)
	}

	return bytes.NewReader(text), versionOf(text), nil
}

// failRead answers a 500 for a vault file that couldn't be read, and logs the cause, which the
// client doesn't get.
func failRead(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	logger.ErrorContext(r.Context(), "read a vault file", "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

// setFileHeaders sets the headers that every file gets (decision 92), and a note's type and its
// version, which bodyOf gives (decision 400).
func setFileHeaders(header http.Header, name, version string) {
	header.Set("Content-Security-Policy", fileCSP)
	header.Set("X-Content-Type-Options", "nosniff")
	// The browser asks every time, so a file is fresh after a Sync; Last-Modified makes that a 304,
	// and so does a note's ETag, though each sync's new checkout changes the note's mtime.
	header.Set("Cache-Control", "private, no-cache")

	if !isNote(name) {
		return
	}

	// Go's type for .md depends on the host's MIME files, so notes get theirs here.
	header.Set("Content-Type", "text/markdown; charset=utf-8")
	header.Set("ETag", version)
}
