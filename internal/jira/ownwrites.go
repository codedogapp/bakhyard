package jira

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"
)

// ownWrites remembers the tickets to which the client added worklogs, so that a worklog read
// finds them before Jira's search does, which shows a change only seconds or minutes after it
// (decision 191). It remembers a ticket until the span read starts after the latest start it added
// there. Its zero value remembers nothing.
type ownWrites struct {
	mu sync.Mutex
	// latest maps a ticket's ID to the latest start of the worklogs that the client added to it.
	latest map[string]time.Time
}

// remember remembers the ticket whose ID is issueID, to which the client added a worklog that
// starts at started.
func (o *ownWrites) remember(issueID string, started time.Time) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.latest == nil {
		o.latest = map[string]time.Time{}
	}

	if latest, ok := o.latest[issueID]; !ok || started.After(latest) {
		o.latest[issueID] = started
	}
}

// unfound forgets the tickets whose latest start is before start, and returns, in order, the IDs
// of the rest that the search didn't find, which found lists. It keeps those that the search
// found: the search may have found one through an older worklog, and lose it when that worklog
// leaves its days, before it finds the new one.
func (o *ownWrites) unfound(found []issue, start time.Time) []string {
	o.mu.Lock()
	defer o.mu.Unlock()

	maps.DeleteFunc(o.latest, func(_ string, latest time.Time) bool {
		return latest.Before(start)
	})

	return slices.DeleteFunc(slices.Sorted(maps.Keys(o.latest)), func(issueID string) bool {
		return slices.ContainsFunc(found, func(searched issue) bool {
			return searched.ID == issueID
		})
	})
}

// forget forgets the ticket whose ID is issueID.
func (o *ownWrites) forget(issueID string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.latest, issueID)
}

// ownWorklogs returns the user's worklogs in the span on the remembered tickets that the search
// didn't find, which found lists, in the order of their IDs. It skips and forgets a ticket whose
// worklogs Jira answers with a 404, as when the ticket is deleted.
func (c *Client) ownWorklogs(ctx context.Context, found []issue, mine span) ([]Worklog, error) {
	var worklogs []Worklog

	for _, issueID := range c.writes.unfound(found, mine.start) {
		read, err := c.worklogs(ctx, issueID, mine)
		if isNotFound(err) {
			c.writes.forget(issueID)

			continue
		}

		if err != nil {
			return nil, err
		}

		worklogs = append(worklogs, read...)
	}

	return worklogs, nil
}
