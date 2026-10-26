// Package migrations holds todos' database migrations, which database.Open runs when todos starts
// (decision 102).
package migrations

import (
	"embed"
	"io/fs"
)

// files are the migrations, embedded in the binary.
//
//go:embed *.sql
var files embed.FS

// FS returns todos' goose migrations, the .sql files at its root.
func FS() fs.FS {
	return files
}
