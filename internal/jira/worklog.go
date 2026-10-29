package jira

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"
)

const (
	// worklogPageSize is the number of worklogs that the client asks for in each page. A worklog
	// takes about 2.4 KB with its two users' avatars, so a page stays under maxAnswer.
	worklogPageSize = 1000
	// startedLayout is the layout of a worklog's start in Jira's answers and in a new worklog,
	// which isn't RFC 3339.
	startedLayout = "2006-01-02T15:04:05.000-0700"
)

// Worklog is time that the user logged on a ticket.
type Worklog struct {
	// Started is when the work started.
	Started time.Time
	// Spent is how long the work took.
	Spent time.Duration
}

// user is the part of an account in Jira's answers that the client reads.
type user struct {
	AccountID string `json:"accountId"`
}

// worklogPage is the part of a page of a ticket's worklogs that the client reads.
type worklogPage struct {
	Worklogs []worklog `json:"worklogs"`
	// Total is the number of worklogs in all the pages.
	Total int `json:"total"`
}

// worklog is the part of a worklog in Jira's answer that the client reads.
type worklog struct {
	Author           user   `json:"author"`
	Started          string `json:"started"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
}

// span is the user's account and the span of time whose worklogs the client reads.
type span struct {
	account    string
	start, end time.Time
}

// Worklogs returns the user's worklogs that start from start up to, but not including, end, the
// earliest first; those that start together keep the order in which the client reads their
// tickets. It reads the tickets that Jira's search finds, in Jira's order, then those that the
// search didn't find but to which the client added worklogs (decision 191).
func (c *Client) Worklogs(ctx context.Context, start, end time.Time) ([]Worklog, error) {
	account, err := c.account(ctx)
	if err != nil {
		return nil, err
	}

	issues, err := c.search(ctx, loggedOn(start, end), "id")
	if err != nil {
		return nil, err
	}

	mine := span{account: account, start: start, end: end}
	worklogs := []Worklog{}

	for _, issue := range issues {
		found, err := c.worklogs(ctx, issue.ID, mine)
		if err != nil {
			return nil, err
		}

		worklogs = append(worklogs, found...)
	}

	own, err := c.ownWorklogs(ctx, issues, mine)
	if err != nil {
		return nil, err
	}

	worklogs = append(worklogs, own...)
	slices.SortStableFunc(worklogs, func(a, b Worklog) int { return a.Started.Compare(b.Started) })

	return worklogs, nil
}

// loggedOn returns the JQL that finds the tickets on which the user logged work from start up
// to end. JQL reads dates in the zone of the user's Jira profile, and any zone's date is within a
// day of the UTC date, so the search takes in a day more on each side; Worklogs keeps the span.
func loggedOn(start, end time.Time) string {
	return fmt.Sprintf(
		`worklogAuthor = currentUser() AND worklogDate >= "%s" AND worklogDate < "%s" ORDER BY key`,
		start.UTC().AddDate(0, 0, -1).Format(time.DateOnly),
		end.UTC().AddDate(0, 0, 2).Format(time.DateOnly),
	)
}

// account returns the ID of the user's Atlassian account.
func (c *Client) account(ctx context.Context) (string, error) {
	var myself user

	err := c.get(
		ctx,
		c.settings.API.JoinPath("rest/api/3/myself"),
		"read the Jira account",
		&myself,
	)
	if err != nil {
		return "", err
	}

	if myself.AccountID == "" {
		return "", errors.New("read Jira's answer: no account ID")
	}

	return myself.AccountID, nil
}

// worklogs returns the user's worklogs in the span on the ticket whose ID is issueID, page after
// page.
func (c *Client) worklogs(ctx context.Context, issueID string, mine span) ([]Worklog, error) {
	var worklogs []Worklog

	startAt := 0

	for range maxPages {
		found, err := c.worklogPage(ctx, issueID, mine, startAt)
		if err != nil {
			return nil, err
		}

		kept, err := mine.pick(found.Worklogs)
		if err != nil {
			return nil, err
		}

		worklogs = append(worklogs, kept...)
		startAt += len(found.Worklogs)

		if len(found.Worklogs) == 0 || startAt >= found.Total {
			return worklogs, nil
		}
	}

	return nil, fmt.Errorf("read Jira worklogs: more than %d pages", maxPages)
}

// worklogPage returns the page of the ticket's worklogs in the span that starts at startAt.
func (c *Client) worklogPage(ctx context.Context, issueID string, mine span, startAt int) (
	worklogPage, error,
) {
	query := url.Values{
		// Jira doesn't say whether it keeps a worklog that starts at a bound, so the answer
		// starts a millisecond early, and pick keeps the span.
		"startedAfter":  {strconv.FormatInt(mine.start.UnixMilli()-1, 10)},
		"startedBefore": {strconv.FormatInt(mine.end.UnixMilli(), 10)},
		"startAt":       {strconv.Itoa(startAt)},
		"maxResults":    {strconv.Itoa(worklogPageSize)},
	}

	// The ID is one path segment, whatever Jira sends.
	target := c.settings.API.JoinPath("rest/api/3/issue", url.PathEscape(issueID), "worklog")
	target.RawQuery = query.Encode()

	var found worklogPage

	err := c.get(ctx, target, "read Jira worklogs", &found)
	if err != nil {
		return worklogPage{}, err
	}

	return found, nil
}

// pick returns the worklogs in found that are the user's and start in the span.
func (s span) pick(found []worklog) ([]Worklog, error) {
	var kept []Worklog

	for _, entry := range found {
		if entry.Author.AccountID != s.account {
			continue
		}

		started, err := time.Parse(startedLayout, entry.Started)
		if err != nil {
			return nil, fmt.Errorf("read Jira's answer: %w", err)
		}

		if started.Before(s.start) || !started.Before(s.end) {
			continue
		}

		spent := time.Duration(entry.TimeSpentSeconds) * time.Second
		kept = append(kept, Worklog{Started: started, Spent: spent})
	}

	return kept, nil
}
