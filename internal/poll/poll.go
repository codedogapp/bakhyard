// Package poll looks something up in the background, at once and then at a fixed interval, keeps
// the latest of what it found, and answers it over HTTP (decisions 122, 135 and 145).
package poll

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// Snapshot is what the polls have found: what the latest poll to succeed found, when that poll
// began, and why the latest poll failed, if it did.
type Snapshot[T any] struct {
	// Found is what the latest poll to succeed found, T's zero value until one has. T must be a
	// struct, whose fields sit at the top level of the snapshot's JSON.
	Found T `json:",embed"`
	// Searched is when that poll began, in UTC to the second, zero before one has succeeded.
	Searched time.Time `json:"searched,omitzero"`
	// Failure says why the latest poll failed, and is empty when it didn't.
	Failure string `json:"failure,omitempty"`
}

// Poller looks something up in the background and keeps the latest of what it finds.
type Poller[T any] struct {
	source string
	every  time.Duration
	look   func(context.Context) (T, error)
	logger *slog.Logger
	latest atomic.Pointer[Snapshot[T]]
	// polled is closed once the first poll has ended, however it ended.
	polled chan struct{}
}

// New returns a poller that looks source up with look, at once and then every interval, once
// Run runs; with an interval that isn't positive, it polls only once. It logs each poll that
// fails to logger, as "poll <source>".
func New[T any](
	source string, every time.Duration, look func(context.Context) (T, error), logger *slog.Logger,
) *Poller[T] {
	poller := &Poller[T]{
		source: source, every: every, look: look, logger: logger, polled: make(chan struct{}),
	}
	poller.latest.Store(&Snapshot[T]{})

	return poller
}

// Run polls at once and then every interval, until ctx ends, which also cuts short the poll that
// runs then. It must run only once.
func (p *Poller[T]) Run(ctx context.Context) {
	tick := time.Tick(p.every)

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
func (p *Poller[T]) Latest(ctx context.Context) (Snapshot[T], error) {
	select {
	case <-p.polled:
		return *p.latest.Load(), nil
	case <-ctx.Done():
		return Snapshot[T]{}, ctx.Err()
	}
}

// poll polls once and keeps what it finds, or why it failed.
func (p *Poller[T]) poll(ctx context.Context) {
	began := time.Now().UTC().Truncate(time.Second)

	found, err := p.look(ctx)

	switch {
	case ctx.Err() != nil:
		// The service is stopping, which cut the poll short: that's no news.
	case err != nil:
		p.logger.ErrorContext(ctx, "poll "+p.source, "err", err)

		failed := *p.latest.Load()
		// An HTTP/1.1 status line may carry any byte, but JSON needs valid UTF-8.
		failed.Failure = strings.ToValidUTF8(err.Error(), "\uFFFD")
		p.latest.Store(&failed)
	default:
		p.latest.Store(&Snapshot[T]{Found: found, Searched: began})
	}
}
