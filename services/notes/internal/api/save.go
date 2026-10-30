package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

// maxNote is the most that a save takes: Gitea's API leaves out the bytes of a file over 10 MiB, so
// notes couldn't read a bigger note back to save it again (decision 422).
const maxNote = 10 << 20

// Store is the vault's repository, where a save commits a note, and a new note goes: Gitea's,
// through its client (decision 420).
type Store interface {
	// Read returns the text of the file name and its blob ID, or gitea.ErrNotFound.
	Read(ctx context.Context, name string) (gitea.File, error)
	// Update commits text as the file name in place of blob, or fails with gitea.ErrChanged when
	// the file has changed since, gitea.ErrNotFound when it's gone, or gitea.ErrRefused when Gitea
	// refuses the commit.
	Update(ctx context.Context, name, blob string, text []byte) error
	// Create commits text as a new file name, or fails with gitea.ErrTaken when something takes
	// the name, or gitea.ErrRefused when Gitea refuses the commit.
	Create(ctx context.Context, name string, text []byte) error
}

// errChanged says that the note isn't at the version that a save names, or is gone.
var errChanged = errors.New("the note has changed")

// saver serves PUT /notes/files/<path>, which saves a note on idx's latest list in store.
type saver struct {
	idx    *index
	store  Store
	logger *slog.Logger
}

// ServeHTTP commits the body as the note at <path>, once the note in the store is at the version
// that If-Match names, and answers 204 with the new version as its ETag (decisions 400 and 422); a
// body that the store holds already gets the 204 with no commit. A file that isn't a note on the
// list gets a 404, a save without If-Match a 428, and a note that has changed or is gone a 412.
// The store's other failures get a 502, and the logger gets their cause.
func (s saver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The decoded path, so that a name escaped in the URL, such as %3F for ?, matches the list.
	name := strings.TrimPrefix(r.URL.Path, filesPath)
	if !isNote(name) || !s.idx.lists(name) {
		http.NotFound(w, r)

		return
	}

	version := r.Header.Get("If-Match")
	if version == "" {
		answer(w, http.StatusPreconditionRequired)

		return
	}

	text, ok := readText(w, r)
	if !ok {
		return
	}

	saved, err := s.save(r.Context(), name, version, text)
	switch {
	case errors.Is(err, errChanged):
		answer(w, http.StatusPreconditionFailed)
	case err != nil:
		s.logger.ErrorContext(r.Context(), "save a note", "err", err)
		answer(w, http.StatusBadGateway)
	default:
		w.Header().Set("ETag", saved)
		w.WriteHeader(http.StatusNoContent)
	}
}

// save commits text as the note name in the store, unless the note there isn't at version, and
// returns the note's new version.
func (s saver) save(ctx context.Context, name, version string, text []byte) (string, error) {
	file, err := s.store.Read(ctx, name)
	if errors.Is(err, gitea.ErrNotFound) {
		return "", errChanged
	}

	if err != nil {
		return "", err
	}

	// A text that the store holds already is saved, whatever the version, as RFC 9110 allows for a
	// change made already, such as one whose answer was lost. Gitea would commit it again, empty.
	if bytes.Equal(file.Text, text) {
		return versionOf(text), nil
	}

	// Only the very version read: a *, a list or a weak tag doesn't skip the check (decision 400).
	if versionOf(file.Text) != version {
		return "", errChanged
	}

	err = s.update(ctx, name, file.Blob, text)
	if err != nil {
		return "", err
	}

	return versionOf(text), nil
}

// update commits text as the note name in the store in place of blob. Gitea refuses a commit that
// another one beat to the branch, so a refused update gets one more try, which Gitea's own check of
// the blob makes safe: it saves, or finds the note changed (decision 422).
func (s saver) update(ctx context.Context, name, blob string, text []byte) error {
	err := s.store.Update(ctx, name, blob, text)
	if errors.Is(err, gitea.ErrRefused) {
		err = s.store.Update(ctx, name, blob, text)
	}

	if errors.Is(err, gitea.ErrChanged) || errors.Is(err, gitea.ErrNotFound) {
		return errChanged
	}

	return err
}

// readText returns the body of a save, a note's text in UTF-8. When it can't, the client gets a
// 413 for a body over maxNote, or a 400, and readText returns false.
func readText(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	text, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxNote))
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		answer(w, http.StatusRequestEntityTooLarge)

		return nil, false
	}

	if err != nil || !utf8.Valid(text) {
		answer(w, http.StatusBadRequest)

		return nil, false
	}

	return text, true
}

// answer answers the status code, with its text as the body.
func answer(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
