package report

import (
	"testing"
	"time"
)

// zone loads a time zone from the system's database, as the service does.
func zone(t *testing.T, name string) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}

	return loc
}

func TestSpan(t *testing.T) {
	budapest := zone(t, "Europe/Budapest")
	// Each of these zones skipped a midnight in 2026: the clocks went from 00:00 to 01:00.
	havana := zone(t, "America/Havana")
	santiago := zone(t, "America/Santiago")
	beirut := zone(t, "Asia/Beirut")
	cairo := zone(t, "Africa/Cairo")
	hourOn := func(loc *time.Location, month time.Month, day, hour int) time.Time {
		return time.Date(2026, month, day, hour, 0, 0, 0, loc)
	}

	tests := []struct {
		name      string
		now       time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "over the clocks going back",
			now:       hourOn(budapest, time.October, 29, 15),
			wantStart: hourOn(budapest, time.October, 23, 0),
			wantEnd:   hourOn(budapest, time.October, 30, 0),
		},
		{
			name:      "over the clocks going forward",
			now:       hourOn(budapest, time.March, 31, 9),
			wantStart: hourOn(budapest, time.March, 25, 0),
			wantEnd:   hourOn(budapest, time.April, 1, 0),
		},
		{
			name:      "at midnight, a day later than in UTC",
			now:       hourOn(budapest, time.October, 29, 0),
			wantStart: hourOn(budapest, time.October, 23, 0),
			wantEnd:   hourOn(budapest, time.October, 30, 0),
		},
		{
			name:      "a nanosecond before midnight",
			now:       hourOn(budapest, time.October, 30, 0).Add(-time.Nanosecond),
			wantStart: hourOn(budapest, time.October, 23, 0),
			wantEnd:   hourOn(budapest, time.October, 30, 0),
		},
		{
			name:      "in UTC",
			now:       hourOn(time.UTC, time.October, 29, 0),
			wantStart: hourOn(time.UTC, time.October, 23, 0),
			wantEnd:   hourOn(time.UTC, time.October, 30, 0),
		},
		{
			name:      "from a skipped midnight behind UTC",
			now:       hourOn(havana, time.March, 14, 12),
			wantStart: hourOn(havana, time.March, 8, 1),
			wantEnd:   hourOn(havana, time.March, 15, 0),
		},
		{
			name:      "to a skipped midnight behind UTC",
			now:       hourOn(havana, time.March, 7, 12),
			wantStart: hourOn(havana, time.March, 1, 0),
			wantEnd:   hourOn(havana, time.March, 8, 1),
		},
		{
			name:      "to a skipped midnight on a Saturday night",
			now:       hourOn(santiago, time.September, 5, 12),
			wantStart: hourOn(santiago, time.August, 30, 0),
			wantEnd:   hourOn(santiago, time.September, 6, 1),
		},
		{
			name:      "to a skipped midnight ahead of UTC",
			now:       hourOn(beirut, time.March, 28, 12),
			wantStart: hourOn(beirut, time.March, 22, 0),
			wantEnd:   hourOn(beirut, time.March, 29, 1),
		},
		{
			name:      "from a skipped midnight ahead of UTC",
			now:       hourOn(cairo, time.April, 30, 12),
			wantStart: hourOn(cairo, time.April, 24, 1),
			wantEnd:   hourOn(cairo, time.May, 1, 0),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end := Span(test.now)
			if !start.Equal(test.wantStart) || !end.Equal(test.wantEnd) {
				t.Errorf("got %v to %v, want %v to %v", start, end, test.wantStart, test.wantEnd)
			}
		})
	}
}
