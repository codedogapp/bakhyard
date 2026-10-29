package bitbucket

import (
	"context"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

// The times between two polls (plan §3's polling defaults). Looking for the pull requests that
// wait for the user's review takes many calls, and Bitbucket allows 1,000 an hour (decision 171).
const (
	mineInterval    = 5 * time.Minute
	reviewsInterval = 15 * time.Minute
)

// Found is what a poll of pull requests finds.
type Found struct {
	// Pulls are the open pull requests that the poll lists. They're nil, and left out of the
	// JSON, until a poll has succeeded; omitempty would leave out an empty list too.
	Pulls []Pull `json:"pulls,omitzero"`
}

// Poller polls Bitbucket in the background and keeps the latest pull requests that it finds
// (decisions 135, 146 and 171).
type Poller = poll.Poller[Found]

// NewPoller returns a poller that lists the user's open pull requests with mine, such as a
// Client's Mine. It polls every five minutes once Run runs, and logs the polls that fail to
// logger.
func NewPoller(mine func(context.Context) ([]Pull, error), logger *slog.Logger) *Poller {
	return newPoller("Bitbucket", mineInterval, mine, logger)
}

// NewReviewsPoller returns a poller that lists the open pull requests that wait for the user's
// review with reviews, such as a Client's Reviews. It polls every 15 minutes once Run runs, and
// logs the polls that fail to logger.
func NewReviewsPoller(reviews func(context.Context) ([]Pull, error), logger *slog.Logger) *Poller {
	return newPoller("Bitbucket reviews", reviewsInterval, reviews, logger)
}

// newPoller returns a poller that lists pull requests with list, at once and then every interval
// once Run runs, and logs the polls that fail to logger, as "poll <source>".
func newPoller(
	source string,
	interval time.Duration,
	list func(context.Context) ([]Pull, error),
	logger *slog.Logger,
) *Poller {
	look := func(ctx context.Context) (Found, error) {
		pulls, err := list(ctx)

		return Found{Pulls: pulls}, err
	}

	return poll.New(source, interval, look, logger)
}
