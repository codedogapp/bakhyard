// Package attention serves time's side of the attention contract: the weekdays of the time report
// with nothing logged, today's included, as items that need the user on the Home (docs/plan.md
// decision 188).
package attention

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/time/internal/report"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// dueHour is the hour of a day by which its hours are due: 8 hours after the 09:00 at which the
// worklogs that Bakhyard adds start (decision 180).
const dueHour = 17

// Lister lists the weekdays with nothing logged in the time reports that a poller keeps, as the
// attention contract's items.
type Lister struct {
	reports *worklogs.Poller
	loc     *time.Location
}

// NewLister returns a lister of the weekdays with nothing logged in the reports that reports
// keeps, whose days follow loc, as the reports' do. It lists none when Jira isn't set up and
// reports is nil.
func NewLister(reports *worklogs.Poller, loc *time.Location) *Lister {
	return &Lister{reports: reports, loc: loc}
}

// ListItems lists the weekdays with nothing logged in the report that the latest poll to succeed
// worked out, today's included, the earliest first, once the first poll has ended, or returns
// ctx's error if ctx ends first. Until a poll succeeds, the items are unavailable.
func (l *Lister) ListItems(
	ctx context.Context,
	_ *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	if l.reports == nil {
		return attentionv1.ListItemsResponse_builder{}.Build(), nil
	}

	latest, err := l.reports.Latest(ctx)
	if err != nil {
		return nil, err
	}

	if latest.Searched.IsZero() {
		return nil, connect.NewError(connect.CodeUnavailable,
			"no read of the worklogs on Jira has succeeded yet")
	}

	days := latest.Found.Days
	items := make([]*attentionv1.Item, 0, len(days))

	for i, day := range days {
		if !day.Unlogged {
			continue
		}

		item, err := l.item(day, i == len(days)-1)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return attentionv1.ListItemsResponse_builder{Items: items}.Build(), nil
}

// item returns the attention item of day, a weekday with nothing logged, which is the report's
// today when today is true. It's due at 17:00 on the day.
func (l *Lister) item(day report.Day, today bool) (*attentionv1.Item, error) {
	date, err := time.Parse(time.DateOnly, day.Date)
	if err != nil {
		return nil, fmt.Errorf("read the day's date: %w", err)
	}

	title, detail := "Log "+date.Weekday().String()+"'s hours", "Nothing logged"

	if today {
		title, detail = "Log today's hours", "Nothing logged yet"
	}

	year, month, dayOfMonth := date.Date()

	return attentionv1.Item_builder{
		Id:     new(day.Date),
		Title:  new(title),
		Detail: new(detail),
		Source: new(attentionv1.Source_SOURCE_JIRA),
		Due:    timestamppb.New(time.Date(year, month, dayOfMonth, dueHour, 0, 0, 0, l.loc)),
	}.Build(), nil
}
