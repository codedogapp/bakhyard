package jira

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"
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

// Snapshot is what the polls have found: the tickets of the latest poll that succeeded, and why
// the latest poll failed, if it did.
type Snapshot struct {
	// Tickets are the user's open tickets that the latest poll to succeed found. They're nil, and
	// left out of the JSON, until one has; omitempty would leave out an empty list too.
	Tickets []Ticket `json:"tickets,omitzero"`
	// Done are the tickets that the todos left to do name and that poll found done (decision
	// 133), left out of the JSON when there are none.
	Done []Ticket `json:"done,omitempty"`
	// Searched is when that poll began, in UTC to the second, zero before one has succeeded.
	Searched time.Time `json:"searched,omitzero"`
	// Failure says why the latest poll failed, and is empty when it didn't.
	Failure string `json:"failure,omitempty"`
}

// Poller polls Jira in the background and keeps the latest of what it finds (decisions 122 and
// 133).
type Poller struct {
	finder Finder
	linked func(context.Context) ([]string, error)
	logger *slog.Logger
	latest atomic.Pointer[Snapshot]
	// polled is closed once the first poll has ended, however it ended.
	polled chan struct{}
}

// NewPoller returns a poller that finds the tickets with finder, such as a Client, and learns
// from linked the keys of the tickets that the todos left to do name. It logs the polls that
// fail to logger, and polls only once Run runs.
func NewPoller(
	finder Finder, linked func(context.Context) ([]string, error), logger *slog.Logger,
) *Poller {
	poller := &Poller{finder: finder, linked: linked, logger: logger, polled: make(chan struct{})}
	poller.latest.Store(&Snapshot{})

	return poller
}

// Run polls at once and then every five minutes, until ctx ends, which also cuts short the poll
// that runs then. It must run only once.
func (p *Poller) Run(ctx context.Context) {
	tick := time.Tick(pollInterval)

	p.poll(ctx)
	close(p.polled)

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			p.poll(ctx)
		}
	}
}

// Latest returns what the polls have found, once the first poll has ended, or ctx's error if
// ctx ends first.
func (p *Poller) Latest(ctx context.Context) (Snapshot, error) {
	select {
	case <-p.polled:
		return *p.latest.Load(), nil
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	}
}

// poll polls once and keeps what it finds, or why it failed.
func (p *Poller) poll(ctx context.Context) {
	began := time.Now().UTC().Truncate(time.Second)

	open, done, err := p.look(ctx)

	switch {
	case ctx.Err() != nil:
		// The service is stopping, which cut the poll short: that's no news.
	case err != nil:
		p.logger.ErrorContext(ctx, "poll Jira", "err", err)

		failed := *p.latest.Load()
		// An HTTP/1.1 status line may carry any byte, but JSON needs valid UTF-8.
		failed.Failure = strings.ToValidUTF8(err.Error(), "\uFFFD")
		p.latest.Store(&failed)
	default:
		p.latest.Store(&Snapshot{Tickets: open, Done: done, Searched: began})
	}
}

// look searches the user's open tickets, then fetches the other tickets that the todos left to
// do name, and returns the open tickets and the fetched ones that are done.
func (p *Poller) look(ctx context.Context) (open, done []Ticket, err error) {
	open, err = p.finder.Search(ctx)
	if err != nil {
		return nil, nil, err
	}

	keys, err := p.linked(ctx)
	if err != nil {
		return nil, nil, err
	}

	keys = slices.DeleteFunc(keys, func(key string) bool {
		return slices.ContainsFunc(open, func(ticket Ticket) bool { return ticket.Key == key })
	})

	found, err := p.finder.Fetch(ctx, keys)
	if err != nil {
		return nil, nil, err
	}

	done = slices.DeleteFunc(found, func(ticket Ticket) bool { return ticket.Category != Done })

	return open, done, nil
}
