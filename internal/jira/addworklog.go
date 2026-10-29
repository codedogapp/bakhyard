package jira

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// newWorklog is the body of a new worklog: when the work started, in Jira's layout, and how long
// it took.
type newWorklog struct {
	Started          string `json:"started"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
}

// addedWorklog is the part of Jira's answer to a new worklog that the client reads.
type addedWorklog struct {
	// IssueID is the ID of the worklog's ticket.
	IssueID string `json:"issueId"`
}

// AddWorklog logs worklog on the ticket key as the user, in whole seconds (decision 189). Jira
// keeps its defaults: it emails the ticket's watchers, and takes the time off the ticket's
// remaining estimate. A key that isn't a Jira key fails before anything is sent. The client
// remembers the ticket, so that Worklogs reads the new worklog before Jira's search finds it; once
// Jira answers 201, the worklog counts as added even if the client can't read the answer, since
// another try would add it twice (decision 191).
func (c *Client) AddWorklog(ctx context.Context, key string, worklog Worklog) error {
	const action = "add a Jira worklog"

	// The key is a path segment, which "." or ".." would move.
	if !IsKey(key) {
		return fmt.Errorf("%s: %q isn't a Jira key", action, key)
	}

	req, err := newPost(ctx, c.settings.API.JoinPath("rest/api/3/issue", key, "worklog"),
		newWorklog{
			Started:          worklog.Started.Format(startedLayout),
			TimeSpentSeconds: int64(worklog.Spent / time.Second),
		})
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	var added addedWorklog

	err = c.do(req, http.StatusCreated, action, &added)
	if errors.Is(err, errAnswer) {
		return nil
	}

	if err != nil {
		return err
	}

	if added.IssueID != "" {
		c.writes.remember(added.IssueID, worklog.Started)
	}

	return nil
}
