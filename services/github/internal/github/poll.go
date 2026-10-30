package github

import (
	"context"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

// interval is the time between two searches of a list (plan §3's polling defaults). A round of
// the three searches makes three requests, and at most 30, within search's 30 a minute (decision
// 203).
const interval = 5 * time.Minute

// Found is what a search of one of the lists finds.
type Found struct {
	// Items are the open pull requests or issues that the search lists. They're nil, and left out
	// of the JSON, until a search has succeeded; omitempty would leave out an empty list too.
	Items []Item `json:"items,omitzero"`
}

// Poller searches GitHub in the background and keeps the latest items that it finds (decision
// 204).
type Poller = poll.Poller[Found]

// NewPoller returns a poller that lists items with list, such as a Client's Mine, at once and then
// every five minutes once Run runs, and logs the searches that fail to logger, as "poll <source>".
func NewPoller(
	source string, list func(context.Context) ([]Item, error), logger *slog.Logger,
) *Poller {
	look := func(ctx context.Context) (Found, error) {
		items, err := list(ctx)

		return Found{Items: items}, err
	}

	return poll.New(source, interval, look, logger)
}
