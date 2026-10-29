// Package worklogs polls Jira for the user's worklogs of the last 7 days and keeps the time report
// that they make (docs/plan.md decisions 180 to 182).
package worklogs

import (
	"context"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/internal/poll"
	"github.com/codedogapp/bakhyard/services/time/internal/report"
)

// pollInterval is the time between two polls (plan §3's polling defaults).
const pollInterval = 5 * time.Minute

// Poller polls Jira in the background and keeps the latest time report.
type Poller = poll.Poller[report.Report]

// NewPoller returns a poller that reads the user's worklogs with read, such as a jira.Client's
// Worklogs, and works the time report out from them. Each poll takes the clock's time in loc,
// whose days and weeks the report follows, reads the worklogs of the report's span at that time
// and works the report out at that same time. It polls every five minutes once Run runs, and logs
// the polls that fail to logger.
func NewPoller(
	read func(ctx context.Context, start, end time.Time) ([]jira.Worklog, error),
	loc *time.Location,
	logger *slog.Logger,
) *Poller {
	look := func(ctx context.Context) (report.Report, error) {
		now := time.Now().In(loc)
		start, end := report.Span(now)

		worklogs, err := read(ctx, start, end)
		if err != nil {
			return report.Report{}, err
		}

		return report.New(worklogs, now), nil
	}

	return poll.New("Jira worklogs", pollInterval, look, logger)
}
