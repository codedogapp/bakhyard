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

		name := entry.Name()
		if path != "." && (strings.HasPrefix(name, ".") || !utf8.ValidString(name)) {
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

// skip leaves out entry: the whole folder if it is one, else the file alone.
func skip(entry fs.DirEntry) error {
	if entry.IsDir() {
		return fs.SkipDir
	}

	return nil
}
