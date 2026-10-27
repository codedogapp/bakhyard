package bitbucket

import (
	"context"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

// pollInterval is the time between two polls (plan §3's polling defaults).
const pollInterval = 5 * time.Minute

// Found is what a poll of the user's pull requests finds.
type Found struct {
	// Pulls are the user's open pull requests. They're nil, and left out of the JSON, until a poll
	// has succeeded; omitempty would leave out an empty list too.
	Pulls []Pull `json:"pulls,omitzero"`
}

// Poller polls Bitbucket in the background and keeps the latest of the user's open pull requests
// that it finds (decisions 135 and 146).
type Poller = poll.Poller[Found]

// NewPoller returns a poller that lists the user's open pull requests with mine, such as a
// Client's Mine. It polls every five minutes once Run runs, and logs the polls that fail to
// logger.
func NewPoller(mine func(context.Context) ([]Pull, error), logger *slog.Logger) *Poller {
	look := func(ctx context.Context) (Found, error) {
		pulls, err := mine(ctx)

		return Found{Pulls: pulls}, err
	}

	return poll.New("Bitbucket", pollInterval, look, logger)
}
