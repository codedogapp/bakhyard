package api

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
)

// isNote reports whether the vault file name is a note, a .md file, which gets its own type and a
// version.
func isNote(name string) bool { return path.Ext(name) == ".md" }

// versionOf returns the version of a note that holds text: the SHA-256 of its bytes, in hex and in
// quotes, so that it's a strong ETag, which a save sends back in If-Match (decision 400).
func versionOf(text []byte) string {
	sum := sha256.Sum256(text)

	return `"` + hex.EncodeToString(sum[:]) + `"`
}
