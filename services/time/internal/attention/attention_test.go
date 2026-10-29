package attention_test

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/internal/poll"
	"github.com/codedogapp/bakhyard/services/time/internal/attention"
	"github.com/codedogapp/bakhyard/services/time/internal/report"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// A bubble's clock starts at midnight UTC on Saturday, 1 January 2000. In Hawaii, ten hours behind
// UTC, that's 14:00 on Friday, 31 December 1999, so the report's days run from Saturday the 25th
// to today, Friday the 31st.
var hawaii = time.FixedZone("HST", -10*60*60)

// onTuesday is the worklog that the reads find, unless they fail: two hours on Tuesday the 28th.
var onTuesday = []jira.Worklog{{Started: hawaiiAt(28, 9), Spent: 2 * time.Hour}}

// withTuesday are the items when only Tuesday has hours.
var withTuesday = []string{
	"1999-12-27 Log Monday's hours · Nothing logged · SOURCE_JIRA · 1999-12-27T17:00:00-10:00",
	"1999-12-29 Log Wednesday's hours · Nothing logged · SOURCE_JIRA · 1999-12-29T17:00:00-10:00",
	"1999-12-30 Log Thursday's hours · Nothing logged · SOURCE_JIRA · 1999-12-30T17:00:00-10:00",
	"1999-12-31 Log today's hours · Nothing logged yet · SOURCE_JIRA · 1999-12-31T17:00:00-10:00",
}

func TestListItemsListsTheWeekdaysWithNothingLogged(t *testing.T) {
	tests := []struct {
		name   string
		loc    *time.Location
		logged []jira.Worklog
		want   []string
	}{
		{name: "days before today and today", loc: hawaii, logged: onTuesday, want: withTuesday},
		{
			name: "today with something logged",
			loc:  hawaii,
			logged: append(slices.Clone(onTuesday),
				jira.Worklog{Started: hawaiiAt(31, 9), Spent: time.Minute}),
			want: withTuesday[:3],
		},
		{
			// In UTC, today is Saturday, 1 January 2000, and Friday is a day before it.
			name: "a weekend today",
			loc:  time.UTC,
			want: []string{
				"1999-12-27 Log Monday's hours · Nothing logged · SOURCE_JIRA · 1999-12-27T17:00:00Z",
				"1999-12-28 Log Tuesday's hours · Nothing logged · SOURCE_JIRA · 1999-12-28T17:00:00Z",
				"1999-12-29 Log Wednesday's hours · Nothing logged · SOURCE_JIRA · 1999-12-29T17:00:00Z",
				"1999-12-30 Log Thursday's hours · Nothing logged · SOURCE_JIRA · 1999-12-30T17:00:00Z",
				"1999-12-31 Log Friday's hours · Nothing logged · SOURCE_JIRA · 1999-12-31T17:00:00Z",
			},
		},
		{name: "every weekday with something logged", loc: hawaii, logged: weekdaysLogged()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				reports := started(t, test.loc,
					func(context.Context, time.Time, time.Time) ([]jira.Worklog, error) {
						return test.logged, nil
					})

				got, err := list(t.Context(), reports, test.loc)
				if err != nil || !slices.Equal(got, test.want) {
					t.Errorf("got %q and %v, want %q", got, err, test.want)
				}
			})
		})
	}
}

func TestListItemsListsNothingWithoutJira(t *testing.T) {
	if got, err := list(t.Context(), nil, hawaii); err != nil || len(got) != 0 {
		t.Errorf("got %q and %v, want no items", got, err)
	}
}

func TestListItemsFailsOnADayWhoseDateIsNotADate(t *testing.T) {
	look := func(context.Context) (report.Report, error) {
		return report.Report{Days: []report.Day{{Date: "Tuesday", Unlogged: true}}}, nil
	}

	// An interval of 0 polls once.
	reports := poll.New("time reports", 0, look, slog.New(slog.DiscardHandler))
	go reports.Run(t.Context())

	got, err := list(t.Context(), reports, hawaii)
	if err == nil || !strings.HasPrefix(err.Error(), "read the day's date: ") || got != nil {
		t.Errorf("got %q and %v, want no answer and the error read the day's date: …", got, err)
	}
}

// hawaiiAt returns the hour of a day in December 1999, in Hawaii.
func hawaiiAt(day, hour int) time.Time {
	return time.Date(1999, time.December, day, hour, 0, 0, 0, hawaii)
}

// weekdaysLogged returns worklogs of an hour on each weekday of the report, from Monday the 27th to
// Friday the 31st.
func weekdaysLogged() []jira.Worklog {
	logged := make([]jira.Worklog, 0, 5)
	for day := 27; day <= 31; day++ {
		logged = append(logged, jira.Worklog{Started: hawaiiAt(day, 9), Spent: time.Hour})
	}

	return logged
}

// started returns a poller in loc that reads the worklogs with read, and runs it until the test
// ends.
func started(
	t *testing.T,
	loc *time.Location,
	read func(ctx context.Context, start, end time.Time) ([]jira.Worklog, error),
) *worklogs.Poller {
	t.Helper()

	poller := worklogs.NewPoller(read, loc, slog.New(slog.DiscardHandler))
	go poller.Run(t.Context())

	return poller
}

// list returns the items that a lister of reports in loc lists, each as "<id> <title> · <detail> ·
// <source> · <due in loc>", or the error that it returns.
func list(ctx context.Context, reports *worklogs.Poller, loc *time.Location) ([]string, error) {
	res, err := attention.NewLister(reports, loc).ListItems(ctx, &attentionv1.ListItemsRequest{})
	if err != nil {
		return nil, err
	}

	items := make([]string, 0, len(res.GetItems()))
	for _, item := range res.GetItems() {
		items = append(items, fmt.Sprintf(
			"%s %s · %s · %v · %s",
			item.GetId(),
			item.GetTitle(),
			item.GetDetail(),
			item.GetSource(),
			item.GetDue().AsTime().In(loc).Format(time.RFC3339),
		))
	}

	return items, nil
}
