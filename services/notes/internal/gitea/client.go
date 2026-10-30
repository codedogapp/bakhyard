// Package gitea reads a note's bytes from the vault's repository in Gitea, and commits a note's
// new bytes in their place, or as a new note, through Gitea's contents API and the token of a bot
// user that can write the vault (docs/plan.md §3, decisions 420, 421 and 423).
package gitea

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// requestTimeout bounds each call to Gitea.
const requestTimeout = 10 * time.Second

var (
	// ErrNotFound says that the repository has no file at the path.
	ErrNotFound = errors.New("no such file in the vault's repository")
	// ErrChanged says that the file no longer has the blob that it had when it was read.
	ErrChanged = errors.New("the file has changed since it was read")
	// ErrRefused says that Gitea refused a commit: another commit took the branch first, or the
	// bot user can't write to the repository.
	ErrRefused = errors.New("refused: another commit took the branch first, or the bot can't write")
	// ErrTaken says that something in the repository takes the path: a file or a folder there, or
	// a file where one of the path's folders would be.
	ErrTaken = errors.New("the path is taken in the vault's repository")
)

// change is a commit to one file that the client asks Gitea for.
type change struct {
	// Content is the file's new bytes, which JSON carries in base64.
	Content []byte `json:"content"`
	// SHA is the ID of the blob that the new bytes replace, which a new file has none of.
	SHA string `json:"sha,omitempty"`
}

// Settings say where the vault's repository is, and as whom to reach it.
type Settings struct {
	// Repo is the address of the vault's repository in Gitea's API, such as
	// https://gitea.example/api/v1/repos/owner/vault.
	Repo *url.URL
	// Token is the access token of the bot user that writes the vault.
	Token string
}

// Client reads and writes the files of the vault's repository, on its default branch.
type Client struct {
	settings Settings
	client   *http.Client
}

// New returns the client of the repository in settings. It reaches Gitea only when it reads or
// writes.
func New(settings Settings) *Client {
	return &Client{settings: settings, client: &http.Client{
		Timeout: requestTimeout,
		// Go would follow a 301 or a 302 with a GET, which would turn an update into a read.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

// target returns the address of the file at name under route, such as "contents", when name is
// a file's path in the repository, such as "Notes/Inbox.md".
func (c *Client) target(route, name string) (*url.URL, error) {
	// A path with a "." or a ".." in it could reach outside the repository.
	if !fs.ValidPath(name) || name == "." {
		return nil, fmt.Errorf("%q isn't a file's path in the repository", name)
	}

	var segments []string
	for segment := range strings.SplitSeq(name, "/") {
		segments = append(segments, url.PathEscape(segment))
	}

	return c.settings.Repo.JoinPath(route, strings.Join(segments, "/")), nil
}

// commit asks Gitea to commit body to the file at name, a path in the repository, on its default
// branch, with method: POST makes the file and PUT updates it. It returns Gitea's status code.
func (c *Client) commit(ctx context.Context, method, name string, body change) (int, error) {
	target, err := c.target("contents", name)
	if err != nil {
		return 0, err
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	res, err := c.send(ctx, method, target, bytes.NewReader(encoded))
	if err != nil {
		return 0, err
	}

	// The status tells how the commit went, so the rest of the answer goes unread.
	_ = res.Body.Close()

	return res.StatusCode, nil
}

// send sends a request with body, if any, to target, as the bot user, and returns Gitea's
// answer, whose body the caller closes.
func (c *Client) send(
	ctx context.Context, method string, target *url.URL, body io.Reader,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "token "+c.settings.Token)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.client.Do(req)
}
