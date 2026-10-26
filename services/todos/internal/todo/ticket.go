package todo

import (
	"errors"
	"regexp"
)

// jiraKey matches a Jira ticket's key, as Jira makes them: its project's key, an upper-case letter
// and at least one more upper-case letter, digit or underscore, then a hyphen and the ticket's
// number (decision 125).
var jiraKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]+-[1-9][0-9]*$`)

// ErrTicket is the error of a ticket that is neither a Jira key nor empty.
var ErrTicket = errors.New("invalid ticket: not a Jira key such as ABC-123")

// validTicket reports whether ticket is a Jira key, or empty, which unlinks the todo's ticket.
func validTicket(ticket string) bool {
	return ticket == "" || jiraKey.MatchString(ticket)
}
