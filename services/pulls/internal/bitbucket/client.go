// Package bitbucket lists the user's open pull requests on Bitbucket Cloud, and those that wait for
// the user's review, through its REST API 2.0 and a scoped API token (docs/plan.md §2, decisions
// 135 and 171).
package bitbucket

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"
)

const (
	// pageSize is the number of pull requests that the client asks for in each page.
	pageSize = 50
	// maxPages bounds the pages of each workspace, so that a server that never sends its last
	// page can't keep the client going.
	maxPages = 10
	// maxAnswer bounds the size of each answer from Bitbucket.
	maxAnswer = 4 << 20
	// requestTimeout bounds each call to Bitbucket.
	requestTimeout = 10 * time.Second
	// listing starts the errors of a listing of the user's pull requests.
	listing = "list my Bitbucket pull requests"
)

// Settings say whose pull requests to list, and in which workspaces.
type Settings struct {
	// API is the address of Bitbucket's REST API, https://api.bitbucket.org/2.0.
	API *url.URL
	// Email and Token are the user's Atlassian account email and API token.
	Email string
	Token string
	// Workspaces are the slugs of the workspaces whose pull requests the client lists.
	Workspaces []string
}

// Client lists pull requests on Bitbucket.
type Client struct {
	settings Settings
	client   *http.Client
}

// New returns the client of the workspaces in settings. It reaches Bitbucket only when it lists.
func New(settings Settings) *Client {
	return &Client{settings: settings, client: &http.Client{Timeout: requestTimeout}}
}

// Mine returns the user's open pull requests in the workspaces, the latest updated first.
func (c *Client) Mine(ctx context.Context) ([]Pull, error) {
	uuid, err := c.me(ctx)
	if err != nil {
		return nil, err
	}

	pulls := []Pull{}

	for _, workspace := range c.settings.Workspaces {
		target := c.settings.API.JoinPath(
			"workspaces", url.PathEscape(workspace), "pullrequests", url.PathEscape(uuid),
		)
		target.RawQuery = url.Values{
			"state": {"OPEN"}, "pagelen": {strconv.Itoa(pageSize)},
		}.Encode()

		found, err := c.pages(ctx, target, listing)
		if err != nil {
			return nil, err
		}

		pulls = append(pulls, found...)
	}

	latestFirst(pulls)

	return pulls, nil
}

// latestFirst sorts pulls the latest updated first. Pull requests updated at the same time keep
// their order.
func latestFirst(pulls []Pull) {
	slices.SortStableFunc(pulls, func(a, b Pull) int { return b.Updated.Compare(a.Updated) })
}

// me returns the user's uuid, which Bitbucket wraps in braces.
func (c *Client) me(ctx context.Context) (string, error) {
	var user struct {
		UUID string `json:"uuid"`
	}

	err := c.get(ctx, c.settings.API.JoinPath("user"), "find me on Bitbucket", &user)
	if err != nil {
		return "", err
	}

	if user.UUID == "" {
		return "", errors.New("find me on Bitbucket: no uuid in the answer")
	}

	return user.UUID, nil
}

// page is the part of a page of values, such as pull requests, that the client reads.
type page[T any] struct {
	Values []T    `json:"values"`
	Next   string `json:"next"`
}

// pages returns the pull requests on the page at target and on the pages that follow it. Its
// errors start with action, such as "list my Bitbucket pull requests".
func (c *Client) pages(ctx context.Context, target *url.URL, action string) ([]Pull, error) {
	var pulls []Pull

	for range maxPages {
		var found page[pullRequest]

		err := c.get(ctx, target, action, &found)
		if err != nil {
			return nil, err
		}

		for _, request := range found.Values {
			pulls = append(pulls, request.pull())
		}

		if found.Next == "" {
			return pulls, nil
		}

		target, err = c.next(found.Next, action)
		if err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("%s: more than %d pages", action, maxPages)
}

// next returns the next page's link as Bitbucket gave it, which clients mustn't build, when it
// stays on the API's scheme and host, so that the token goes nowhere else. Its error starts with
// action.
func (c *Client) next(link, action string) (*url.URL, error) {
	target, err := url.Parse(link)
	if err != nil || target.Scheme != c.settings.API.Scheme ||
		target.Host != c.settings.API.Host {
		return nil, fmt.Errorf("%s: the next page is off %s", action, c.settings.API.Host)
	}

	return target, nil
}

// get reads the answer at target into answer, as the user. Its errors start with action, such
// as "find me on Bitbucket", or with "read Bitbucket's answer" when the answer is no use.
func (c *Client) get(ctx context.Context, target *url.URL, action string, answer any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	req.SetBasicAuth(c.settings.Email, c.settings.Token)
	req.Header.Set("Accept", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", action, res.Status)
	}

	err = json.UnmarshalRead(io.LimitReader(res.Body, maxAnswer), answer)
	if err != nil {
		return fmt.Errorf("read Bitbucket's answer: %w", err)
	}

	return nil
}
