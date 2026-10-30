// Package github lists the user's open pull requests, the pull requests that wait for the user's
// review and the open issues assigned to the user, across every owner, through the search of
// GitHub's REST API and a personal access token (docs/plan.md §2, decisions 201 and 202).
package github

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// pageSize is the number of results that the client asks for in each page, GitHub's most.
	pageSize = 100
	// maxPages bounds the pages of each search, since GitHub gives only its first 1,000 results.
	maxPages = 10
	// maxAnswer bounds the size of each answer from GitHub.
	maxAnswer = 4 << 20
	// requestTimeout bounds each call to GitHub.
	requestTimeout = 10 * time.Second
	// apiVersion is the version of GitHub's REST API that the client asks for.
	apiVersion = "2026-03-10"
	// userAgent names the app to GitHub, as GitHub asks.
	userAgent = "bakhyard"
)

// Settings say where GitHub's API is, and as whom to search it.
type Settings struct {
	// API is the address of GitHub's REST API, https://api.github.com.
	API *url.URL
	// Token is the user's personal access token.
	Token string
}

// Client searches GitHub as the user.
type Client struct {
	settings Settings
	client   *http.Client
}

// New returns the client of the user whose token settings hold. It reaches GitHub only when it
// searches.
func New(settings Settings) *Client {
	return &Client{settings: settings, client: &http.Client{Timeout: requestTimeout}}
}

// Mine returns the user's open pull requests, the latest updated first.
func (c *Client) Mine(ctx context.Context) ([]Item, error) {
	return c.search(ctx, "is:pr is:open author:@me archived:false",
		"search GitHub for my open PRs")
}

// Reviews returns the open pull requests of others that wait for the review of the user or of
// one of the user's teams, the latest updated first.
func (c *Client) Reviews(ctx context.Context) ([]Item, error) {
	return c.search(ctx, "is:pr is:open review-requested:@me -author:@me archived:false",
		"search GitHub for my review requests")
}

// Issues returns the open issues assigned to the user, the latest updated first.
func (c *Client) Issues(ctx context.Context) ([]Item, error) {
	return c.search(ctx, "is:issue is:open assignee:@me archived:false",
		"search GitHub for my issues")
}

// search returns the items that query finds, in GitHub's order, the latest updated first, each
// once. Its errors start with action, such as "search GitHub for my issues", or with "read
// GitHub's answer" when the answer is no use.
func (c *Client) search(ctx context.Context, query, action string) ([]Item, error) {
	var found []issue

	for number := range maxPages {
		page, err := c.page(ctx, query, number+1, action)
		if err != nil {
			return nil, err
		}

		found = append(found, page...)

		if len(page) < pageSize {
			break
		}
	}

	return itemsOf(found)
}

// results are the part of a page of search results in GitHub's answer that the client reads.
type results struct {
	Incomplete bool    `json:"incomplete_results"`
	Items      []issue `json:"items"`
}

// page returns the issues and pull requests on page number of the search for query. Its errors
// are those of search.
func (c *Client) page(
	ctx context.Context, query string, number int, action string,
) ([]issue, error) {
	target := c.settings.API.JoinPath("search", "issues")
	target.RawQuery = url.Values{
		"q": {query}, "sort": {"updated"}, "order": {"desc"},
		"per_page": {strconv.Itoa(pageSize)}, "page": {strconv.Itoa(number)},
	}.Encode()

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

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", action, res.Status)
	}

	var answer results

	err = json.UnmarshalRead(io.LimitReader(res.Body, maxAnswer), &answer)
	if err != nil {
		return nil, fmt.Errorf("read GitHub's answer: %w", err)
	}

	// A search that ran out of time may have missed some results.
	if answer.Incomplete {
		return nil, fmt.Errorf("%s: GitHub's results are incomplete", action)
	}

	return answer.Items, nil
}
