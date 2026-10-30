// Package refresh polls on request rather than on a timer: a request that finds what was found
// more than an hour old has it found again in the background, and polls that fail in a row wait
// longer and longer (docs/plan.md §3, decisions 67, 300 and 302).
package refresh

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

const (
	// fresh is how long what a poll found stays fresh.
	fresh = time.Hour
	// firstWait is how long the next poll waits after a poll that fails; each poll that fails in
	// a row doubles it, up to longestWait.
	firstWait   = 5 * time.Minute
	longestWait = 6 * time.Hour
)

// OnRequest serves what its poller found, and has the poller poll again when a request finds it
// stale. The first request waits for the first poll; the others get what was found at once.
type OnRequest[T any] struct {
	poller *poll.Poller[T]
	// asked carries a request's ask for a poll to Run.
	asked chan struct{}

	mu sync.Mutex
	// polling says whether a poll was asked for and hasn't ended.
	polling bool
	// due is when the next poll is due, or zero before the first.
	due time.Time
	// wait is how long the next poll waits after the last one, which failed, or 0 when the last
	// poll succeeded.
	wait time.Duration
}

// New returns an OnRequest whose poller looks source up with look and logs each poll that fails
// to logger, as poll.New's does, but polls only when a request asks.
func New[T any](
	source string, look func(context.Context) (T, error), logger *slog.Logger,
) *OnRequest[T] {
	return &OnRequest[T]{poller: poll.New(source, 0, look, logger), asked: make(chan struct{}, 1)}
}

// Run polls each time a request asks, one poll at a time, until ctx ends. At the first ask it runs
// the poller, whose Run polls once and lets the poller answer; later asks poll with PollNow.
func (o *OnRequest[T]) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-o.asked:
	}

	go o.poller.Run(ctx)

	for {
		latest, err := o.poller.Latest(ctx)
		if err != nil {
			return
		}

		o.polled(latest)

		select {
		case <-ctx.Done():
			return
		case <-o.asked:
			if o.poller.PollNow(ctx) != nil {
				return
			}
		}
	}
}

// ServeHTTP answers what the poller found, as poll.Poller does, after asking for a poll when one
// is due.
func (o *OnRequest[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.ask()
	o.poller.ServeHTTP(w, r)
}

// ask asks Run for a poll when one is due and none is under way.
func (o *OnRequest[T]) ask() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.polling || time.Now().Before(o.due) {
		return
	}

	o.polling = true
	// Run has taken every ask before the last poll ended, so this never blocks.
	o.asked <- struct{}{}
}

// polled notes the poll that ended with latest: the next poll is due when what it found turns
// stale, or after a wait when it failed.
func (o *OnRequest[T]) polled(latest poll.Snapshot[T]) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.polling = false

	if latest.Failure == "" {
		o.wait = 0
		o.due = latest.Searched.Add(fresh)

		return
	}

	o.wait = min(max(2*o.wait, firstWait), longestWait)
	o.due = time.Now().Add(o.wait)
}
