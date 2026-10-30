package api

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/codedogapp/bakhyard/services/notes/internal/vault"
)

// snapshot is one scan of the vault: the root it was scanned through, the paths of its files in
// walk order, and the same paths as a set.
type snapshot struct {
	root   *os.Root
	files  []string
	listed map[string]bool
}

// scan opens the vault in folder as a new root, so that a link that has moved since leads to its
// new target (decision 151), and lists the files in it.
func scan(folder string) (*snapshot, error) {
	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil, fmt.Errorf("open the vault: %w", err)
	}

	files, err := vault.Scan(root.FS())
	if err != nil {
		// Nothing was written through root, so closing it can't lose anything.
		_ = root.Close()

		return nil, err
	}

	listed := make(map[string]bool, len(files))
	for _, file := range files {
		listed[file] = true
	}

	return &snapshot{root: root, files: files, listed: listed}, nil
}

// index keeps the latest scan of the vault in folder, which a sync replaces whole (decisions 93
// and 155).
type index struct {
	folder string
	// scanning makes the scans take turns, so the one kept is always the newest.
	scanning sync.Mutex
	// mu guards latest. A request holds it for reading while it opens a file through latest's
	// root, so once a sync has swapped latest, no request opens anything through the old root.
	mu     sync.RWMutex
	latest *snapshot
}

// newIndex returns the index of the vault in folder, which it scans for the first time.
func newIndex(folder string) (*index, error) {
	first, err := scan(folder)
	if err != nil {
		return nil, err
	}

	return &index{folder: folder, latest: first}, nil
}

// files returns the paths of the files that the latest scan lists.
func (idx *index) files() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.latest.files
}

// lists reports whether the latest scan lists the file name.
func (idx *index) lists(name string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.latest.listed[name]
}

// open opens the file name through the root of the latest scan, and fails with fs.ErrNotExist
// unless that scan lists it and it's still a regular file.
func (idx *index) open(name string) (*os.File, fs.FileInfo, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if !idx.latest.listed[name] {
		return nil, nil, fs.ErrNotExist
	}

	return open(idx.latest.root, name)
}

// rescan scans the vault again and keeps the result in place of the latest scan, whose root it
// then closes, and returns the new list. When the scan fails, the latest scan stays.
func (idx *index) rescan() ([]string, error) {
	idx.scanning.Lock()
	defer idx.scanning.Unlock()

	next, err := scan(idx.folder)
	if err != nil {
		return nil, err
	}

	idx.mu.Lock()
	previous := idx.latest
	idx.latest = next
	idx.mu.Unlock()

	// The files already opened through the old root stay open, and nothing was written through
	// it, so closing it can't lose anything.
	_ = previous.root.Close()

	return next.files, nil
}
