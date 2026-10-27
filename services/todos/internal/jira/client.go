// Package jira finds the user's open tickets on a Jira Cloud site, through Jira's REST API v3
// and a scoped API token (docs/plan.md §2, decisions 103 and 121).
package jira

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

// Client finds tickets on a Jira site.
type Client struct {
	settings Settings
	client   *http.Client
}

// New returns the client of the site in settings. It reaches Jira only when it searches or
// fetches.
func New(settings Settings) *Client {
	return &Client{settings: settings, client: &http.Client{Timeout: requestTimeout}}
}

// Search returns the user's open tickets, the latest updated first.
func (c *Client) Search(ctx context.Context) ([]Ticket, error) {
	tickets := []Ticket{}
	token := ""

	for range maxPages {
		found, err := c.page(ctx, token)
		if err != nil {
			return nil, err
		}

		for _, issue := range found.Issues {
			tickets = append(tickets, c.ticket(issue))
		}

		if found.IsLast || found.NextPageToken == "" {
			return tickets, nil
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

// page returns the page of the search that token names, or the first page without a token.
func (c *Client) page(ctx context.Context, token string) (page, error) {
	query := url.Values{
		"jql":        {openTickets},
		"fields":     {"summary,status"},
		"maxResults": {strconv.Itoa(pageSize)},
	}
	if token != "" {
		query.Set("nextPageToken", token)
	}

	target := c.settings.API.JoinPath("rest/api/3/search/jql")
	target.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return page{}, fmt.Errorf("search Jira: %w", err)
	}

	var found page

	err = c.do(req, "search Jira", &found)
	if err != nil {
		return page{}, err
	}

	return found, nil
}

// do sends req to Jira as the user and reads Jira's answer into answer. Its errors start with
// action, such as "search Jira", or with "read Jira's answer" when the answer is no use.
func (c *Client) do(req *http.Request, action string, answer any) error {
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
