package report

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

func TestNewSumsTheTimeLoggedOnEachDay(t *testing.T) {
	budapest := zone(t, "Europe/Budapest")
	// 2026-10-29 is a Thursday.
	inBudapest := func(day, hour, minute int) time.Time {
		return time.Date(2026, time.October, day, hour, minute, 0, 0, budapest)
	}
	india := time.FixedZone("IST", 5*60*60+30*60)
	worklogs := []jira.Worklog{
		{Started: inBudapest(22, 10, 0), Spent: 2 * time.Hour},
		{Started: inBudapest(23, 9, 0), Spent: 90 * time.Minute},
		// Budapest's clocks went back at 03:00 that day.
		{Started: inBudapest(25, 2, 30), Spent: time.Hour},
		// 0.1 + 0.2 hours make 0.30000000000000004 in floating point.
		{Started: inBudapest(26, 9, 0), Spent: 6 * time.Minute},
		{Started: inBudapest(26, 11, 0), Spent: 12 * time.Minute},
		{Started: inBudapest(28, 23, 0), Spent: 2 * time.Hour},
		// 21:30 on the 28th in Budapest.
		{Started: time.Date(2026, time.October, 29, 2, 0, 0, 0, india), Spent: 2 * time.Hour},
		{Started: inBudapest(29, 8, 0), Spent: 15 * time.Minute},
		{Started: inBudapest(30, 9, 0), Spent: 8 * time.Hour},
	}

	report := New(worklogs, inBudapest(29, 15, 4))

	wantDays := []Day{
		{Date: "2026-10-23", LoggedHours: 1.5},
		{Date: "2026-10-24"},
		{Date: "2026-10-25", LoggedHours: 1},
		{Date: "2026-10-26", LoggedHours: 0.3},
		{Date: "2026-10-27", Missing: true},
		{Date: "2026-10-28", LoggedHours: 4},
		{Date: "2026-10-29", LoggedHours: 0.25},
	}
	if !slices.Equal(report.Days, wantDays) {
		t.Errorf("got the days %+v, want %+v", report.Days, wantDays)
	}

	// From Monday the 26th through today: 18 minutes, 4 hours and 15 minutes.
	if wantWeek := (Week{LoggedHours: 4.55, TargetHours: 40}); report.Week != wantWeek {
		t.Errorf("got the week %+v, want %+v", report.Week, wantWeek)
	}
}

func TestNewCountsTheWeekFromMonday(t *testing.T) {
	monday := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	worklogs := make([]jira.Worklog, 0, 14)

	// A tenth of an hour on each day from Monday the 19th to Sunday 1 November.
	for day := -7; day < 7; day++ {
		started := monday.AddDate(0, 0, day)
		worklogs = append(worklogs, jira.Worklog{Started: started, Spent: 6 * time.Minute})
	}

	// Three tenths added up as hours make 0.30000000000000004.
	for day, want := range []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7} {
		now := monday.AddDate(0, 0, day)
		t.Run(now.Weekday().String(), func(t *testing.T) {
			if got := New(worklogs, now).Week.LoggedHours; got != want {
				t.Errorf("got %v hours this week, want %v", got, want)
			}
		})
	}
}

func TestNewCountsTheDaysInNowsZone(t *testing.T) {
	// Kiritimati is 14 hours ahead of UTC, so its dates differ from UTC's and, almost surely, from
	// those of the machine running the test.
	kiritimati := zone(t, "Pacific/Kiritimati")
	// 00:30 on the 28th in Kiritimati.
	worklogs := []jira.Worklog{
		{Started: time.Date(2026, time.October, 27, 10, 30, 0, 0, time.UTC), Spent: time.Hour},
	}

	days := New(worklogs, time.Date(2026, time.October, 29, 12, 0, 0, 0, kiritimati)).Days

	want := []Day{{Date: "2026-10-28", LoggedHours: 1}, {Date: "2026-10-29"}}
	if got := days[len(days)-2:]; !slices.Equal(got, want) {
		t.Errorf("got the last two days %+v, want %+v", got, want)
	}
}

func TestNewMarksTheWeekdaysWithNothingLoggedAsMissing(t *testing.T) {
	noonOn := func(month time.Month, day int) time.Time {
		return time.Date(2026, month, day, 12, 0, 0, 0, time.UTC)
	}
	worklogs := []jira.Worklog{
		{Started: noonOn(time.October, 29), Spent: time.Minute},
		{Started: noonOn(time.October, 30)},
	}

	report := New(worklogs, noonOn(time.November, 2))

	// From Tuesday 27 October to today, Monday 2 November.
	want := []bool{true, true, false, true, false, false, false}

	got := make([]bool, 0, len(report.Days))
	for _, day := range report.Days {
		got = append(got, day.Missing)
	}

	if !slices.Equal(got, want) {
		t.Errorf("got missing %v, want %v", got, want)
	}
}

func TestReportJSON(t *testing.T) {
	now := time.Date(2026, time.October, 28, 12, 0, 0, 0, time.UTC)
	worklogs := []jira.Worklog{{Started: now.AddDate(0, 0, -1), Spent: 90 * time.Minute}}

	got, err := json.Marshal(New(worklogs, now))
	if err != nil {
		t.Fatal(err)
	}

	want := jsontext.Value(`{
		"days": [
			{"date": "2026-10-22", "loggedHours": 0, "missing": true},
			{"date": "2026-10-23", "loggedHours": 0, "missing": true},
			{"date": "2026-10-24", "loggedHours": 0, "missing": false},
			{"date": "2026-10-25", "loggedHours": 0, "missing": false},
			{"date": "2026-10-26", "loggedHours": 0, "missing": true},
			{"date": "2026-10-27", "loggedHours": 1.5, "missing": false},
			{"date": "2026-10-28", "loggedHours": 0, "missing": false}
		],
		"week": {"loggedHours": 1.5, "targetHours": 40}
	}`)
	if err := want.Compact(); err != nil {
		t.Fatal(err)
	}

	if string(got) != string(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestZeroReportJSON(t *testing.T) {
	got, err := json.Marshal(Report{})
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "{}" {
		t.Errorf("got %s, want {}", got)
	}
}
