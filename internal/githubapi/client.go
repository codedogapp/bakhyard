// Package githubapi asks GitHub's REST API as the user, with a personal access token, for each
// service that talks to GitHub: github searches it, and github and planner ask it when their
// tokens expire (docs/plan.md decisions 202, 503 and 504).
package githubapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	// requestTimeout bounds each call to GitHub.
	requestTimeout = 10 * time.Second
	// apiVersion is the version of GitHub's REST API that the client asks for.
	apiVersion = "2026-03-10"
	// userAgent names the app to GitHub, as GitHub asks.
	userAgent = "bakhyard"
)

// Settings say where GitHub's API is, and as whom to ask it.
type Settings struct {
	// API is the address of GitHub's REST API, such as https://api.github.com.
	API *url.URL
	// Token is the user's personal access token.
	Token string
}

// Client asks GitHub's API as the user.
type Client struct {
	settings Settings
	client   *http.Client
}

// New returns the client of the user whose token settings hold. It reaches GitHub only when it
// asks.
func New(settings Settings) *Client {
	return &Client{settings: settings, client: &http.Client{Timeout: requestTimeout}}
}

// Get asks GitHub's API as the user for the resource at path, below the API's address, with
// query, and returns GitHub's answer once GitHub has said OK; the caller closes its body. Its
// errors start with action, such as "search GitHub for my issues".
func (c *Client) Get(
	ctx context.Context, path string, query url.Values, action string,
) (*http.Response, error) {
	target := c.settings.API.JoinPath(path)
	target.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.settings.Token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}

	if res.StatusCode != http.StatusOK {
		_ = res.Body.Close()

		return nil, fmt.Errorf("%s: %s", action, res.Status)
	}

	return res, nil
}
