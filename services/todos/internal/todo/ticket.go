package todo

import (
	"errors"

	"github.com/codedogapp/bakhyard/internal/jira"
)

// ErrTicket is the error of a ticket that is neither a Jira key nor empty.
var ErrTicket = errors.New("invalid ticket: not a Jira key such as ABC-123")

// validTicket reports whether ticket is a Jira key, or empty, which unlinks the todo's ticket.
func validTicket(ticket string) bool {
	return ticket == "" || jira.IsKey(ticket)
}
