package jira

import (
	"context"
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
	req, err := newPost(ctx, c.settings.API.JoinPath("rest/api/3/issue/bulkfetch"),
		bulkFetch{Keys: keys, Fields: []string{"summary", "status"}})
	if err != nil {
		return fetched{}, fmt.Errorf("fetch Jira tickets: %w", err)
	}

	var found fetched

	err = c.do(req, http.StatusOK, "fetch Jira tickets", &found)
	if err != nil {
		return fetched{}, err
	}

	return found, nil
}
