// Package migrations holds the gateway's database migrations, which database.Open runs when the
// gateway starts (decision 81).
package migrations

import (
	"embed"
	"io/fs"
)

// files are the migrations, embedded in the binary.
//
//go:embed *.sql
var files embed.FS

// FS returns the gateway's goose migrations, the .sql files at its root.
func FS() fs.FS {
	return files
}
