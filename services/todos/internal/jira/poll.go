package jira

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

// pollInterval is the time between two polls (plan §3's polling defaults).
const pollInterval = 5 * time.Minute

// Finder finds tickets on Jira, as a Client does.
type Finder interface {
	// Search returns the user's open tickets.
	Search(ctx context.Context) ([]Ticket, error)
	// Fetch returns the tickets that keys name, whatever their status.
	Fetch(ctx context.Context, keys []string) ([]Ticket, error)
}

// Found is what a poll of Jira finds.
type Found struct {
	// Tickets are the user's open tickets. They're nil, and left out of the JSON, until a poll has
	// succeeded; omitempty would leave out an empty list too.
	Tickets []Ticket `json:"tickets,omitzero"`
	// Done are the tickets that the todos left to do name and that are done (decision 133), left
	// out of the JSON when there are none.
	Done []Ticket `json:"done,omitempty"`
}

// Poller polls Jira in the background and keeps the latest of what it finds (decisions 122 and
// 133).
type Poller = poll.Poller[Found]

// NewPoller returns a poller that finds the tickets with finder, such as a Client, and learns
// from linked the keys of the tickets that the todos left to do name. It polls every five minutes
// once Run runs, and logs the polls that fail to logger.
func NewPoller(
	finder Finder, linked func(context.Context) ([]string, error), logger *slog.Logger,
) *Poller {
	look := func(ctx context.Context) (Found, error) { return lookUp(ctx, finder, linked) }

	return poll.New("Jira", pollInterval, look, logger)
}

// lookUp searches the user's open tickets with finder, then fetches the other tickets that the
// todos left to do name, as linked lists them, and returns the open tickets and the fetched ones
// that are done. It fails whole when any of them fails.
func lookUp(
	ctx context.Context, finder Finder, linked func(context.Context) ([]string, error),
) (Found, error) {
	open, err := finder.Search(ctx)
	if err != nil {
		return Found{}, err
	}

	keys, err := linked(ctx)
	if err != nil {
		return Found{}, err
	}

	keys = slices.DeleteFunc(keys, func(key string) bool {
		return slices.ContainsFunc(open, func(ticket Ticket) bool { return ticket.Key == key })
	})

	fetched, err := finder.Fetch(ctx, keys)
	if err != nil {
		return Found{}, err
	}

	done := slices.DeleteFunc(fetched, func(ticket Ticket) bool { return ticket.Category != Done })

	return Found{Tickets: open, Done: done}, nil
}
