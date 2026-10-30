package gitea

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// maxAnswer bounds the size of each answer about a file: up to 10 MiB of bytes, the most that
// Gitea's API sends by default, in base64, and the rest of the answer.
const maxAnswer = 16 << 20

// File is a file's bytes in the repository, with the ID of the git blob that holds them.
type File struct {
	Text []byte
	Blob string
}

// kind is an entry's type in Gitea's answer: file, dir, symlink or submodule.
type kind string

// encoding is how Gitea's answer carries a file's bytes: base64, or null when it leaves them out.
type encoding string

const (
	// fileKind is the type of a file that's neither a link nor a submodule.
	fileKind kind = "file"
	// base64Encoding is the encoding of the bytes that Gitea sends.
	base64Encoding encoding = "base64"
)

// entry is the part of Gitea's answer about a path that the client reads.
type entry struct {
	Type     kind     `json:"type"`
	SHA      string   `json:"sha"`
	Encoding encoding `json:"encoding"`
	// Content is the file's bytes, which JSON carries in base64.
	Content []byte `json:"content"`
}

// Read returns the file at name, a path in the repository such as "Notes/Inbox.md", or
// ErrNotFound when the repository has none.
func (c *Client) Read(ctx context.Context, name string) (File, error) {
	action := "read " + name + " from Gitea"

	target, err := c.target("contents-ext", name)
	if err != nil {
		return File{}, fmt.Errorf("%s: %w", action, err)
	}

	// Gitea sends a file's bytes only when it's asked to.
	target.RawQuery = url.Values{"includes": {"file_content"}}.Encode()

	res, err := c.send(ctx, http.MethodGet, target, nil)
	if err != nil {
		return File{}, fmt.Errorf("%s: %w", action, err)
	}

	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return File{}, fmt.Errorf("%s: %w", action, ErrNotFound)
	default:
		return File{}, fmt.Errorf("%s: %s", action, res.Status)
	}

	var answer struct {
		File *entry `json:"file_contents"`
	}

	err = json.UnmarshalRead(io.LimitReader(res.Body, maxAnswer), &answer)
	if err != nil {
		return File{}, fmt.Errorf("%s: read Gitea's answer: %w", action, err)
	}

	return answer.File.file(action)
}

// file returns the file that e describes, which Gitea leaves out for a folder. Its errors start
// with action.
func (e *entry) file(action string) (File, error) {
	switch {
	case e == nil:
		return File{}, fmt.Errorf("%s: it's a folder", action)
	case e.Type != fileKind:
		return File{}, fmt.Errorf("%s: it's a %s, not a file", action, e.Type)
	// Gitea leaves out the bytes of a file over its API's limit.
	case e.Encoding != base64Encoding:
		return File{}, fmt.Errorf("%s: Gitea sent none of its bytes", action)
	}

	return File{Text: e.Content, Blob: e.SHA}, nil
}
