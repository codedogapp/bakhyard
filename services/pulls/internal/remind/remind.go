// Package remind pushes pulls' reminders (decision 175): at each slot, one notification that lists
// the user's pull requests that still need approvals.
package remind

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/codedogapp/bakhyard/internal/notifier"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

const (
	// minAge is how long a pull request has to have been open, at a slot, to be reminded of.
	minAge = 24 * time.Hour
	// title is every reminder's title.
	title = "Remind your reviewers"
	// pullsPath is the page of the app that a tap on a reminder opens.
	pullsPath = "/pulls"
)

// Pusher pushes a notification to the user's devices.
type Pusher interface {
	Send(ctx context.Context, message notifier.Message) error
}

// Reminder reminds the user, at each slot, of the pull requests that Mine has found and that still
// need approvals.
type Reminder struct {
	Mine *bitbucket.Poller
	// Approvals reads the approvals that a pull request has and those it needs.
	Approvals func(context.Context, bitbucket.Pull) (bitbucket.Approvals, error)
	Pusher    Pusher
	Logger    *slog.Logger
	// Location is the time zone of the slots.
	Location *time.Location
}

// Run reminds at each slot until ctx ends. A slot that goes by while pulls is down, or while the
// reminder before it runs late, isn't made up.
func (r Reminder) Run(ctx context.Context) {
	from := time.Now()

	for {
		slot := nextSlot(from, r.Location)

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(slot)):
		}

		r.remind(ctx, slot)
		// Even if the clock has stepped back, the next slot comes after this one.
		from = later(time.Now(), slot)
	}
}

// remind pushes one notification that lists the pull requests that still need approvals at slot,
// in the order of the latest poll that succeeded, unless there are none.
func (r Reminder) remind(ctx context.Context, slot time.Time) {
	latest, err := r.Mine.Latest(ctx)
	if err != nil {
		return
	}

	var lines []string

	for _, pull := range latest.Found.Pulls {
		if pull.Draft || slot.Sub(pull.Created) <= minAge {
			continue
		}

		approvals, err := r.Approvals(ctx, pull)
		if err != nil {
			r.warn(ctx, "check a PR for the reminder", err)

			continue
		}

		if approvals.Given < approvals.Required {
			lines = append(lines, line(pull, approvals))
		}
	}

	r.push(ctx, lines)
}

// push pushes one reminder of lines, one per pull request, unless there are none or ctx has ended.
func (r Reminder) push(ctx context.Context, lines []string) {
	if len(lines) == 0 || ctx.Err() != nil {
		return
	}

	message := notifier.Message{Title: title, Body: strings.Join(lines, "\n"), URL: pullsPath}
	if err := r.Pusher.Send(ctx, message); err != nil {
		r.warn(ctx, "push the PR reminder", err)
	}
}

// warn logs a failure as a warning, unless it comes from ctx's end.
func (r Reminder) warn(ctx context.Context, failure string, err error) {
	if ctx.Err() == nil {
		r.Logger.WarnContext(ctx, failure, "err", err)
	}
}

// line returns the reminder's line for pull, which has approvals.
func line(pull bitbucket.Pull, approvals bitbucket.Approvals) string {
	noun := "approvals"
	if approvals.Required == 1 {
		noun = "approval"
	}

	return fmt.Sprintf("%s (%s #%d): %d of %d %s", pull.Title, pull.Repo, pull.ID,
		approvals.Given, approvals.Required, noun)
}
