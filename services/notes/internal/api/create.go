package api

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
	"github.com/codedogapp/bakhyard/services/notes/internal/vault"
)

// maxTries is the most names that a new note tries in the store before it gives up: the list has
// all but the names that git-sync hasn't pulled yet, so a name that the list leaves free is
// rarely taken (decision 423).
const maxTries = 10

// errNoName says that a new note found no free name: maxTries names were taken, or the next one
// would be too long for the vault.
var errNoName = errors.New("no free name for the new note")

// made is the body of a new note's 201: the path that the note got.
type made struct {
	Path string `json:"path"`
}

// creator serves POST /notes/files/<path>, which makes a new note in store, at <path> or at the
// first free name after it.
type creator struct {
	idx    *index
	store  Store
	logger *slog.Logger
}

// ServeHTTP commits the body as a new note, and answers 201 with the path it got (decision 423).
// A path that isn't a note that the vault could list gets a 404, a body over maxNote a 413 and one
// that isn't UTF-8 a 400. A note that finds no free name gets a 409. The store's other failures
// get a 502, and the logger gets their cause.
func (c creator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The decoded path, so that a name escaped in the URL, such as %3F for ?, is the note's name.
	name := strings.TrimPrefix(r.URL.Path, filesPath)
	if !isNote(name) || !vault.Listable(name) {
		http.NotFound(w, r)

		return
	}

	text, ok := readText(w, r)
	if !ok {
		return
	}

	got, err := c.create(r.Context(), name, text)
	switch {
	case errors.Is(err, errNoName):
		answer(w, http.StatusConflict)
	case err != nil:
		c.logger.ErrorContext(r.Context(), "make a note", "err", err)
		answer(w, http.StatusBadGateway)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// A listable name is valid UTF-8, so it always encodes: an error means the client has gone.
		_ = json.MarshalWrite(w, made{Path: got})
	}
}

// create commits text as a new note in the store, at the first of name's numbered names that
// neither the list nor the store takes, and returns that name. It returns errNoName once maxTries
// names are taken, or when the next name would be too long for the vault.
func (c creator) create(ctx context.Context, name string, text []byte) (string, error) {
	tries := 0

	for n := 0; ; n++ {
		candidate := numbered(name, n)
		if c.idx.takes(candidate) {
			continue
		}

		if tries == maxTries || !vault.Listable(candidate) {
			return "", errNoName
		}

		tries++

		err := c.try(ctx, candidate, text)
		switch {
		case err == nil:
			return candidate, nil
		case !errors.Is(err, gitea.ErrTaken):
			return "", err
		}
	}
}

// try commits text as the new note name in the store. Gitea refuses a commit that another one beat
// to the branch, so a refused try gets one more, which Gitea's own check of the name makes safe: it
// makes the note, or finds the name taken (decision 423).
func (c creator) try(ctx context.Context, name string, text []byte) error {
	err := c.store.Create(ctx, name, text)
	if errors.Is(err, gitea.ErrRefused) {
		err = c.store.Create(ctx, name, text)
	}

	return err
}

// numbered returns name for n = 0, and else name with " n" before its ".md", as Obsidian names its
// untitled notes: "Untitled.md", then "Untitled 1.md", "Untitled 2.md" and so on.
func numbered(name string, n int) string {
	if n == 0 {
		return name
	}

	return strings.TrimSuffix(name, ".md") + " " + strconv.Itoa(n) + ".md"
}
