package api_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/notes/internal/api"
	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

// fakeStore is a vault's repository in memory, which takes an update only in place of the blob it
// holds, and a new file only at a name it doesn't hold, in that very case, as Gitea does, and fails
// as told.
type fakeStore struct {
	files map[string]gitea.File
	// commits counts the updates and new files it took.
	commits int
	// readFails, updateFails and createFails, when set, are the errors that Read, Update and
	// Create return.
	readFails, updateFails, createFails error
	// race, when set, is another commit, which beats the next update or new file to the branch, so
	// that Gitea refuses it.
	race func(*fakeStore)
	// tried lists the names that Create was asked for, in order.
	tried []string
}

// storeWith returns a store that holds the files names, each with text(name), as vaultWith's
// vault does.
func storeWith(names ...string) *fakeStore {
	files := make(map[string]gitea.File, len(names))
	for _, name := range names {
		files[name] = gitea.File{Text: []byte(text(name)), Blob: "blob of " + name}
	}

	return &fakeStore{files: files}
}

func (s *fakeStore) Read(_ context.Context, name string) (gitea.File, error) {
	if s.readFails != nil {
		return gitea.File{}, s.readFails
	}

	file, ok := s.files[name]
	if !ok {
		return gitea.File{}, gitea.ErrNotFound
	}

	return file, nil
}

func (s *fakeStore) Update(_ context.Context, name, blob string, content []byte) error {
	if s.race != nil {
		race := s.race
		s.race = nil
		race(s)

		return gitea.ErrRefused
	}

	if s.updateFails != nil {
		return s.updateFails
	}

	file, ok := s.files[name]
	if !ok {
		return gitea.ErrNotFound
	}

	if file.Blob != blob {
		return gitea.ErrChanged
	}

	s.commits++
	s.files[name] = gitea.File{Text: content, Blob: fmt.Sprintf("blob %d", s.commits)}

	return nil
}

func (s *fakeStore) Create(_ context.Context, name string, content []byte) error {
	s.tried = append(s.tried, name)

	if s.race != nil {
		race := s.race
		s.race = nil
		race(s)

		return gitea.ErrRefused
	}

	if s.createFails != nil {
		return s.createFails
	}

	if _, ok := s.files[name]; ok {
		return gitea.ErrTaken
	}

	s.commits++
	s.files[name] = gitea.File{Text: content, Blob: fmt.Sprintf("blob %d", s.commits)}

	return nil
}

// routerWith returns a router that serves the vault in folder and saves its notes in store, and the
// log it writes.
func routerWith(t *testing.T, folder string, store api.Store) (http.Handler, *bytes.Buffer) {
	t.Helper()

	var log bytes.Buffer

	router := chi.NewRouter()

	err := api.Register(router, api.Vault{Folder: folder, Store: store},
		slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatal(err)
	}

	return router, &log
}

// versionOf returns the version of a note that holds content: the SHA-256 of its bytes, in hex
// and in quotes (decision 400).
func versionOf(content string) string {
	sum := sha256.Sum256([]byte(content))

	return `"` + hex.EncodeToString(sum[:]) + `"`
}

// putNote returns PUT target with content as its body, and ifMatch as its If-Match unless it's
// blank.
func putNote(t *testing.T, target, ifMatch, content string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, target,
		strings.NewReader(content))
	req.Header.Set("Content-Type", "text/markdown; charset=utf-8")

	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}

	return req
}

// serve returns router's answer to req.
func serve(router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
