package jira

import (
	"context"
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

// AddWorklog logs worklog on the ticket key as the user, in whole seconds (decision 189). Jira
// keeps its defaults: it emails the ticket's watchers, and takes the time off the ticket's
// remaining estimate. A key that isn't a Jira key fails before anything is sent.
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

	return c.do(req, http.StatusCreated, action, nil)
}
