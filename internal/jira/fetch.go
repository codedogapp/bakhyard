package jira

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"slices"
)

// fetchSize is the number of tickets that the client fetches in each call: the most that
// Jira's bulk fetch takes from any request.
const fetchSize = 100

// bulkFetch is the body of a bulk fetch: the tickets' keys and the fields that the client
// reads.
type bulkFetch struct {
	Keys   []string `json:"issueIdsOrKeys"`
	Fields []string `json:"fields"`
}

// fetched is the part of a bulk fetch's answer that the client reads.
type fetched struct {
	Issues []issue `json:"issues"`
}

// Fetch returns the tickets that keys name, whatever their status, in Jira's order. Jira leaves
// out a key that names no ticket the user can see, and answers for a ticket that moved under
// its new key.
func (c *Client) Fetch(ctx context.Context, keys []string) ([]Ticket, error) {
	tickets := []Ticket{}

	for batch := range slices.Chunk(keys, fetchSize) {
		found, err := c.fetch(ctx, batch)
		if err != nil {
			return nil, err
		}

		for _, issue := range found.Issues {
			tickets = append(tickets, c.ticket(issue))
		}
	}

	return tickets, nil
}

// fetch returns Jira's answer for a batch of at most fetchSize keys.
func (c *Client) fetch(ctx context.Context, keys []string) (fetched, error) {
	body, err := json.Marshal(bulkFetch{Keys: keys, Fields: []string{"summary", "status"}})
	if err != nil {
		return fetched{}, fmt.Errorf("fetch Jira tickets: %w", err)
	}

	target := c.settings.API.JoinPath("rest/api/3/issue/bulkfetch")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(),
		bytes.NewReader(body))
	if err != nil {
		return fetched{}, fmt.Errorf("fetch Jira tickets: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	var found fetched

	err = c.do(req, "fetch Jira tickets", &found)
	if err != nil {
		return fetched{}, err
	}

	return found, nil
}
