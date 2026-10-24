package api

import (
	"os"
	"sync"
	"sync/atomic"

	"github.com/codedogapp/bakhyard/services/notes/internal/vault"
)

// snapshot is one scan of the vault: the paths of its files in walk order, and the same paths as
// a set.
type snapshot struct {
	files  []string
	listed map[string]bool
}

func newSnapshot(files []string) *snapshot {
	listed := make(map[string]bool, len(files))
	for _, file := range files {
		listed[file] = true
	}

	return &snapshot{files: files, listed: listed}
}

// index keeps the latest scan of the vault in root, which a sync replaces whole (decision 93).
type index struct {
	root     *os.Root
	latest   atomic.Pointer[snapshot]
	scanning sync.Mutex
}

// newIndex returns an index of root that starts with files, the paths of its first scan.
func newIndex(root *os.Root, files []string) *index {
	idx := &index{root: root}
	idx.latest.Store(newSnapshot(files))

	return idx
}

// current returns the latest scan.
func (idx *index) current() *snapshot {
	return idx.latest.Load()
}

// rescan scans the vault again and keeps the result, unless the scan fails. Scans run one at a
// time, so the one kept is always the newest.
func (idx *index) rescan() (*snapshot, error) {
	idx.scanning.Lock()
	defer idx.scanning.Unlock()

	files, err := vault.Scan(idx.root.FS())
	if err != nil {
		return nil, err
	}

	next := newSnapshot(files)
	idx.latest.Store(next)

	return next, nil
}
