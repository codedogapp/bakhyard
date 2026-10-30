// Package vault reads the Obsidian vault (docs/plan.md §3, Vault reads).
package vault

import (
	"fmt"
	"io/fs"
	"strings"
	"unicode/utf8"
)

// Scan walks fsys and returns the paths of its regular files, slash-separated and relative to
// fsys's root, in walk order. It skips every folder and file whose name starts with a dot, such
// as .obsidian/ and .DS_Store, or isn't valid UTF-8, which JSON can't carry. It lists no symlink
// or other special file.
func Scan(fsys fs.FS) ([]string, error) {
	var files []string

	err := fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path != "." && skipped(entry.Name()) {
			return skip(entry)
		}

		if entry.Type().IsRegular() {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan the vault: %w", err)
	}

	return files, nil
}

// maxName is the longest folder or file name, in bytes, that the file systems of Linux, ext4's
// among them, take, so the longest that a vault's checkout on the laptop can hold.
const maxName = 255

// Listable reports whether Scan could list a regular file at name, a slash-separated path such as
// "Notes/Inbox.md": one that io/fs calls valid, with no folder or file on it that Scan skips,
// "." included, or that's longer than a checkout can hold.
func Listable(name string) bool {
	if !fs.ValidPath(name) {
		return false
	}

	for segment := range strings.SplitSeq(name, "/") {
		if skipped(segment) || len(segment) > maxName {
			return false
		}
	}

	return true
}

// skipped reports whether Scan skips a folder or file named name.
func skipped(name string) bool {
	return strings.HasPrefix(name, ".") || !utf8.ValidString(name)
}

// skip leaves out entry: the whole folder if it is one, else the file alone.
func skip(entry fs.DirEntry) error {
	if entry.IsDir() {
		return fs.SkipDir
	}

	return nil
}
