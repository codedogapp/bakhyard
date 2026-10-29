package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/internal/jsonbody"
	"github.com/codedogapp/bakhyard/services/time/internal/report"
)

const (
	// startHour is the hour of the day at which a worklog that time adds starts.
	startHour = 9
	// maxMinutes is the most time a worklog may take: a day.
	maxMinutes = 24 * 60
)

var (
	errTicket  = errors.New("invalid ticket: not a Jira key such as ABC-123")
	errDate    = errors.New("invalid date: not a date such as 2026-10-27")
	errDay     = errors.New("invalid date: not one of the last 7 days")
	errMinutes = errors.New("invalid minutes: not from 1 to 1440")
)

// addition is the body of POST /time/worklogs.
type addition struct {
	// Ticket is the key of the ticket to log time on, such as ABC-123.
	Ticket string `json:"ticket"`
	// Date is the day to log time on, such as 2026-10-27.
	Date string `json:"date"`
	// Minutes is the time to log.
	Minutes int `json:"minutes"`
}

// addWorklog returns the handler of POST /time/worklogs, which adds a worklog with backends.Add,
// starting at 09:00 in loc on the date, then polls backends.Reports, so that the report shows the
// worklog, and answers 201. A body that isn't a JSON object with just a ticket, a date and
// minutes, or a ticket, a date or minutes that break their rule, gets a 400 with the reason, the
// ticket's first; a body over 64 KiB gets a 413, and a failed add a 502, whose cause logger gets.
// The add and the poll go on if the client leaves (docs/plan.md decision 192).
func addWorklog(backends Backends, loc *time.Location, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if backends.Reports == nil {
			http.Error(w, notSetUp, http.StatusNotFound)

			return
		}

		var body addition
		if !jsonbody.Read(w, r, &body,
			"the body isn't a JSON object with just a ticket, a date and minutes") {
			return
		}

		worklog, err := body.worklog(time.Now().In(loc))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		ctx := context.WithoutCancel(r.Context())

		err = backends.Add(ctx, body.Ticket, worklog)
		if err != nil {
			logger.ErrorContext(ctx, "add the worklog", "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)

			return
		}

		// ctx doesn't end, so PollNow returns once it has polled; a failed poll goes to the
		// report's failure, as a timed poll's does.
		_ = backends.Reports.PollNow(ctx)

		w.WriteHeader(http.StatusCreated)
	}
}

// worklog returns the worklog that a asks for at now, or why a breaks a rule: it starts at 09:00
// on a's date, which must be one of the report's days at now, in now's location.
func (a addition) worklog(now time.Time) (jira.Worklog, error) {
	if !jira.IsKey(a.Ticket) {
		return jira.Worklog{}, errTicket
	}

	date, err := time.Parse(time.DateOnly, a.Date)
	if err != nil {
		return jira.Worklog{}, errDate
	}

	started := time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0,
		now.Location())

	start, end := report.Span(now)
	if started.Before(start) || !started.Before(end) {
		return jira.Worklog{}, errDay
	}

	if a.Minutes < 1 || a.Minutes > maxMinutes {
		return jira.Worklog{}, errMinutes
	}

	return jira.Worklog{Started: started, Spent: time.Duration(a.Minutes) * time.Minute}, nil
}
