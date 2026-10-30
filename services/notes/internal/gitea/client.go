// Package gitea reads a note's bytes from the vault's repository in Gitea, and commits a note's
// new bytes in their place, through Gitea's contents API and the token of a bot user that can
// write the vault (docs/plan.md §3, decisions 420 and 421).
package gitea

import (
	"context"
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
)

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
