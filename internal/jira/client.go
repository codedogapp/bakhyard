// Package jira reads the user's tickets and worklogs on a Jira Cloud site, through Jira's REST
// API v3 and a scoped API token: it finds the open tickets, fetches tickets by key, and reads and
// adds the user's worklogs, for each service that the Jira settings set up (docs/plan.md §2,
// decisions 103, 121, 133, 180 and 189).
package jira

import (
	"bytes"
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
	// openTickets finds the tickets assigned to the user that aren't done, the latest updated
	// first.
	openTickets = "assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC"
	// pageSize is the number of tickets that the client asks for in each page.
	pageSize = 100
	// maxPages bounds a search, so that a server that never sends its last page can't keep it
	// going.
	maxPages = 10
	// maxAnswer bounds the size of each answer from Jira.
	maxAnswer = 4 << 20
	// requestTimeout bounds each call to Jira.
	requestTimeout = 10 * time.Second
)

// Settings say which Jira site to search, and as whom.
type Settings struct {
	// Site is the site's own address, https://<site>.atlassian.net, where the tickets' pages are.
	Site *url.URL
	// API is the address of the site's REST API: https://api.atlassian.com/ex/jira/<cloudId>
	// for a scoped token.
	API *url.URL
	// Email and Token are the user's account email and API token.
	Email string
	Token string
}

// Client reads tickets and worklogs on a Jira site, and adds worklogs.
type Client struct {
	settings Settings
	client   *http.Client
}

// New returns the client of the site in settings. It reaches Jira only when it's asked for
// something.
func New(settings Settings) *Client {
	return &Client{settings: settings, client: &http.Client{Timeout: requestTimeout}}
}

// Search returns the user's open tickets, the latest updated first.
func (c *Client) Search(ctx context.Context) ([]Ticket, error) {
	issues, err := c.search(ctx, openTickets, "summary,status")
	if err != nil {
		return nil, err
	}

	tickets := make([]Ticket, 0, len(issues))
	for _, issue := range issues {
		tickets = append(tickets, c.ticket(issue))
	}

	return tickets, nil
}

// search returns the tickets that jql finds, with the fields that fields lists, page after page.
func (c *Client) search(ctx context.Context, jql, fields string) ([]issue, error) {
	var issues []issue

	token := ""

	for range maxPages {
		found, err := c.page(ctx, jql, fields, token)
		if err != nil {
			return nil, err
		}

		issues = append(issues, found.Issues...)

		if found.IsLast || found.NextPageToken == "" {
			return issues, nil
		}

		token = found.NextPageToken
	}

	return nil, fmt.Errorf("search Jira: more than %d pages", maxPages)
}

// page is the part of a page of search results that the client reads.
type page struct {
	Issues        []issue `json:"issues"`
	NextPageToken string  `json:"nextPageToken"`
	IsLast        bool    `json:"isLast"`
}

// issue is the part of a ticket in Jira's answer that the client reads.
type issue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields fields `json:"fields"`
}

// fields are the ticket's fields that the client asks for.
type fields struct {
	Summary string `json:"summary"`
	Status  status `json:"status"`
}

// status is a ticket's status, as its workflow names it, and the category it falls into.
type status struct {
	Name     string         `json:"name"`
	Category statusCategory `json:"statusCategory"`
}

// statusCategory is the part of a status category that the client reads: its key, which Jira
// fixes whatever the workflow.
type statusCategory struct {
	Key string `json:"key"`
}

// page returns the page of jql's search that token names, or the first page without a token.
func (c *Client) page(ctx context.Context, jql, fields, token string) (page, error) {
	query := url.Values{
		"jql":        {jql},
		"fields":     {fields},
		"maxResults": {strconv.Itoa(pageSize)},
	}
	if token != "" {
		query.Set("nextPageToken", token)
	}

	target := c.settings.API.JoinPath("rest/api/3/search/jql")
	target.RawQuery = query.Encode()

	var found page

	err := c.get(ctx, target, "search Jira", &found)
	if err != nil {
		return page{}, err
	}

	return found, nil
}

// get asks Jira for target and reads Jira's answer, a 200, into answer, as do does.
func (c *Client) get(ctx context.Context, target *url.URL, action string, answer any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return c.do(req, http.StatusOK, action, answer)
}

// newPost returns a request that posts body to target as JSON.
func newPost(ctx context.Context, target *url.URL, body any) (*http.Request, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(),
		bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// do sends req to Jira as the user and, unless answer is nil, reads Jira's answer into answer.
// Jira must answer with the status want. Its errors start with action, such as "search Jira",
// or with "read Jira's answer" when the answer is no use.
func (c *Client) do(req *http.Request, want int, action string, answer any) error {
	req.SetBasicAuth(c.settings.Email, c.settings.Token)
	req.Header.Set("Accept", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != want {
		return fmt.Errorf("%s: %s", action, res.Status)
	}

	if answer == nil {
		return nil
	}

	err = json.UnmarshalRead(io.LimitReader(res.Body, maxAnswer), answer)
	if err != nil {
		return fmt.Errorf("read Jira's answer: %w", err)
	}

	return nil
}

// ticket returns the Ticket of an issue in Jira's answer.
func (c *Client) ticket(found issue) Ticket {
	return Ticket{
		Key:      found.Key,
		Summary:  found.Fields.Summary,
		Status:   found.Fields.Status.Name,
		Category: categoryOf(found.Fields.Status.Category.Key),
		// The key is one path segment, whatever Jira sends.
		URL: c.settings.Site.JoinPath("browse", url.PathEscape(found.Key)).String(),
	}
}
