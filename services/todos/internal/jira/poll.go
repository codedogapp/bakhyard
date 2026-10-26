package jira

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// pollInterval is the time between two searches (plan §3's polling defaults).
const pollInterval = 5 * time.Minute

// Snapshot is what the searches have found: the tickets of the latest search that succeeded, and
// why the latest search failed, if it did.
type Snapshot struct {
	// Tickets are what the latest search to succeed found. They're nil, and left out of the JSON,
	// until one has; omitempty would leave out an empty list too.
	Tickets []Ticket `json:"tickets,omitzero"`
	// Searched is when that search began, in UTC to the second, zero before one has succeeded.
	Searched time.Time `json:"searched,omitzero"`
	// Failure says why the latest search failed, and is empty when it didn't.
	Failure string `json:"failure,omitempty"`
}

// Poller searches Jira in the background and keeps the latest of what it finds (decision 122).
type Poller struct {
	search func(context.Context) ([]Ticket, error)
	logger *slog.Logger
	latest atomic.Pointer[Snapshot]
	// searched is closed once the first search has ended, however it ended.
	searched chan struct{}
}

// NewPoller returns a poller that searches with search, such as a Client's Search, and logs the
// searches that fail to logger. It searches only once Run runs.
func NewPoller(search func(context.Context) ([]Ticket, error), logger *slog.Logger) *Poller {
	poller := &Poller{search: search, logger: logger, searched: make(chan struct{})}
	poller.latest.Store(&Snapshot{})

	return poller
}

// Run searches at once and then every five minutes, until ctx ends, which also cuts short the
// search that runs then. It must run only once.
func (p *Poller) Run(ctx context.Context) {
	tick := time.Tick(pollInterval)

	p.poll(ctx)
	close(p.searched)

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			p.poll(ctx)
		}
	}
}

// Latest returns what the searches have found, once the first search has ended, or ctx's error
// if ctx ends first.
func (p *Poller) Latest(ctx context.Context) (Snapshot, error) {
	select {
	case <-p.searched:
		return *p.latest.Load(), nil
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	}
}

// poll searches once and keeps the tickets that the search finds, or why it failed.
func (p *Poller) poll(ctx context.Context) {
	began := time.Now().UTC().Truncate(time.Second)

	found, err := p.search(ctx)

	switch {
	case ctx.Err() != nil:
		// The service is stopping, which cut the search short: that's no news.
	case err != nil:
		p.logger.ErrorContext(ctx, "poll Jira", "err", err)

		failed := *p.latest.Load()
		// An HTTP/1.1 status line may carry any byte, but JSON needs valid UTF-8.
		failed.Failure = strings.ToValidUTF8(err.Error(), "\uFFFD")
		p.latest.Store(&failed)
	default:
		p.latest.Store(&Snapshot{Tickets: found, Searched: began})
	}
}
