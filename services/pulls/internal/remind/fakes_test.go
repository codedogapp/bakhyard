package remind_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/notifier"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
	"github.com/codedogapp/bakhyard/services/pulls/internal/remind"
)

// monday is the first Monday in a synctest bubble, whose clock starts at 2000-01-01 00:00 UTC, a
// Saturday.
var monday = time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)

// opened is when the pull requests that have been open for long were opened.
var opened = monday.AddDate(0, 0, -7)

// at returns the time hour:minute on monday, days later.
func at(days, hour, minute int) time.Time {
	return monday.AddDate(0, 0, days).Add(time.Duration(hour)*time.Hour +
		time.Duration(minute)*time.Minute)
}

// pull returns the pull request id of acme/web, with title, opened at created.
func pull(id int, title string, created time.Time) bitbucket.Pull {
	return bitbucket.Pull{ID: id, Title: title, Repo: "acme/web", Created: created}
}

// listing returns a lister of my pull requests that always finds pulls.
func listing(pulls ...bitbucket.Pull) func(context.Context) ([]bitbucket.Pull, error) {
	return func(context.Context) ([]bitbucket.Pull, error) { return pulls, nil }
}

// approvals is a stand-in for Bitbucket's merge checks, which answers the approvals of each pull
// request by its ID and records the IDs it was asked. It fails for any other ID, with approvals
// that would be reminded of, so a test sees whether the error is heeded.
type approvals struct {
	lock    sync.Mutex
	answers map[int]bitbucket.Approvals
	asked   []int
}

// read returns pull's approvals.
func (a *approvals) read(_ context.Context, pull bitbucket.Pull) (bitbucket.Approvals, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.asked = append(a.asked, pull.ID)

	answer, found := a.answers[pull.ID]
	if !found {
		return bitbucket.Approvals{Required: 1}, fmt.Errorf(
			"read the merge checks of %s #%d: 500 Internal Server Error", pull.Repo, pull.ID)
	}

	return answer, nil
}

// needing returns a stand-in that answers given of required approvals for each of ids.
func needing(given, required int, ids ...int) *approvals {
	answers := map[int]bitbucket.Approvals{}
	for _, id := range ids {
		answers[id] = bitbucket.Approvals{Given: given, Required: required}
	}

	return &approvals{answers: answers}
}

// push is a notification that a pusher was given, and when.
type push struct {
	at      time.Time
	message notifier.Message
}

// pusher is a stand-in for notify, which records each notification and answers it with err, or,
// if hold is set, holds it until the push's ctx ends.
type pusher struct {
	lock   sync.Mutex
	pushes []push
	err    error
	hold   bool
}

// Send records message.
func (p *pusher) Send(ctx context.Context, message notifier.Message) error {
	p.lock.Lock()
	p.pushes = append(p.pushes, push{at: time.Now(), message: message})
	p.lock.Unlock()

	if p.hold {
		return held(ctx)
	}

	return p.err
}

// held waits for ctx to end, as a request does when pulls stops, and fails with its error.
func held(ctx context.Context) error {
	<-ctx.Done()

	return fmt.Errorf("held: %w", ctx.Err())
}

// times returns when the notifications were pushed.
func (p *pusher) times() []time.Time {
	p.lock.Lock()
	defer p.lock.Unlock()

	times := make([]time.Time, 0, len(p.pushes))
	for _, push := range p.pushes {
		times = append(times, push.at)
	}

	return times
}

// messages returns the notifications pushed.
func (p *pusher) messages() []notifier.Message {
	p.lock.Lock()
	defer p.lock.Unlock()

	messages := make([]notifier.Message, 0, len(p.pushes))
	for _, push := range p.pushes {
		messages = append(messages, push.message)
	}

	return messages
}

// start runs, until ctx ends, a reminder in UTC of the pull requests that mine finds, with the
// approvals that read reads, which pushes to pusher. It returns the reminder's logs, and a channel
// that closes once its Run returns.
func start(
	ctx context.Context,
	mine func(context.Context) ([]bitbucket.Pull, error),
	read func(context.Context, bitbucket.Pull) (bitbucket.Approvals, error),
	pusher remind.Pusher,
) (*bytes.Buffer, <-chan struct{}) {
	var logs bytes.Buffer

	poller := bitbucket.NewPoller(mine, slog.New(slog.DiscardHandler))
	go poller.Run(ctx)

	reminder := remind.Reminder{
		Mine: poller, Approvals: read, Pusher: pusher,
		Logger: slog.New(slog.NewTextHandler(&logs, nil)), Location: time.UTC,
	}
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		reminder.Run(ctx)
	}()

	return &logs, stopped
}

// wantTimes checks that pusher was given a notification at each of want, and at no other time.
func wantTimes(t *testing.T, pusher *pusher, want ...time.Time) {
	t.Helper()

	if got := pusher.times(); !slices.EqualFunc(got, want, time.Time.Equal) {
		t.Errorf("pushed at %v, want %v", got, want)
	}
}
