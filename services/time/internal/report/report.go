// Package report works out the time report from the user's worklogs: the hours logged on each of
// the last 7 days, the weekdays among them with nothing logged, and this week's hours against the
// week's target (docs/plan.md decisions 180 and 181).
package report

import (
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

const (
	// days is the number of days that the report covers, today the last.
	days = 7
	// weekTarget is the time to log each week: 8 hours each weekday, Monday to Friday.
	weekTarget = 40 * time.Hour
)

// Report is what the user logged in the last 7 days. A report that New works out always has both
// fields; the zero Report, which a poller keeps until a poll succeeds, encodes as {}.
type Report struct {
	// Days are the 7 days, the earliest first and today last.
	Days []Day `json:"days,omitzero"`
	// Week is what the user logged this week, from Monday through today.
	Week Week `json:"week,omitzero"`
}

// Day is what the user logged on one day.
type Day struct {
	// Date is the day's date, such as 2026-10-29.
	Date string `json:"date"`
	// LoggedHours is the time logged on the worklogs that start on the day, in hours.
	LoggedHours float64 `json:"loggedHours"`
	// Missing says that the day is a weekday before today with nothing logged.
	Missing bool `json:"missing"`
	// Unlogged says that the day is a weekday with nothing logged, today included: a missing day,
	// or today until something is logged.
	Unlogged bool `json:"-"`
}

// Week is the time logged this week against the week's target, in hours.
type Week struct {
	LoggedHours float64 `json:"loggedHours"`
	TargetHours float64 `json:"targetHours"`
}

// Span returns the span of time that the report at now covers, in now's location: from the start
// of the day six days before now's up to, but not including, the start of the day after now's.
func Span(now time.Time) (start, end time.Time) {
	today := dateOf(now)

	return dayStart(today.AddDate(0, 0, 1-days), now.Location()),
		dayStart(today.AddDate(0, 0, 1), now.Location())
}

// New returns the report at now of the worklogs. Days and weeks follow now's location: a worklog
// counts on the day it starts, and a week runs from Monday to Sunday. Worklogs that start outside
// Span(now) are left out.
func New(worklogs []jira.Worklog, now time.Time) Report {
	logged := make(map[string]time.Duration)
	for _, worklog := range worklogs {
		logged[worklog.Started.In(now.Location()).Format(time.DateOnly)] += worklog.Spent
	}

	today := dateOf(now)
	monday := mondayOf(today)
	first := today.AddDate(0, 0, 1-days)
	report := Report{Days: make([]Day, 0, days)}

	var week time.Duration

	for i := range days {
		date := first.AddDate(0, 0, i)
		spent := logged[date.Format(time.DateOnly)]
		unlogged := weekday(date) && spent == 0

		report.Days = append(report.Days, Day{
			Date:        date.Format(time.DateOnly),
			LoggedHours: spent.Hours(),
			Missing:     unlogged && date.Before(today),
			Unlogged:    unlogged,
		})

		if !date.Before(monday) {
			week += spent
		}
	}

	report.Week = Week{LoggedHours: week.Hours(), TargetHours: weekTarget.Hours()}

	return report
}

// dateOf returns t's date in t's location, at midnight UTC, where days can be counted without
// clocks that change.
func dateOf(t time.Time) time.Time {
	year, month, day := t.Date()

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// dayStart returns when date's day starts in loc: at midnight, or, where the clocks skip midnight,
// as they skip it.
func dayStart(date time.Time, loc *time.Location) time.Time {
	year, month, day := date.Date()

	start := time.Date(year, month, day, 0, 0, 0, 0, loc)
	if start.Day() != day {
		// time.Date put the skipped midnight before the skip, on the day before; the day starts as
		// the zone in effect then ends.
		_, start = start.ZoneBounds()
	}

	return start
}

// mondayOf returns the Monday of date's week.
func mondayOf(date time.Time) time.Time {
	for date.Weekday() != time.Monday {
		date = date.AddDate(0, 0, -1)
	}

	return date
}

// weekday says whether date is Monday to Friday.
func weekday(date time.Time) bool {
	return date.Weekday() != time.Saturday && date.Weekday() != time.Sunday
}
