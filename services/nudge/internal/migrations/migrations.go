// Package migrations holds nudge's database migrations, which database.Open runs when nudge
// starts (decision 102).
package migrations

import (
	"embed"
	"io/fs"
)

// files are the migrations, embedded in the binary.
//
//go:embed *.sql
var files embed.FS

// FS returns nudge's goose migrations, the .sql files at its root.
func FS() fs.FS {
	return files
}
