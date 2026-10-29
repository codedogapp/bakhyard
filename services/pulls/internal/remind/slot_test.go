package remind

import (
	"testing"
	"time"
)

func TestNextSlot(t *testing.T) {
	// west is twelve hours behind UTC, so from noon on, its date and weekday are a day behind UTC's.
	west := time.FixedZone("UTC-12", -12*60*60)
	// 2026-11-02 is a Monday.
	inWest := func(day, hour, minute int) time.Time {
		return time.Date(2026, time.November, day, hour, minute, 0, 0, west)
	}

	tests := []struct {
		name     string
		from     time.Time
		wantSlot time.Time
	}{
		{"before the morning slot", inWest(2, 9, 59), inWest(2, 10, 0)},
		{"a nanosecond before it", inWest(2, 10, 0).Add(-time.Nanosecond), inWest(2, 10, 0)},
		{"at the morning slot", inWest(2, 10, 0), inWest(2, 15, 0)},
		{"between the slots", inWest(2, 12, 30), inWest(2, 15, 0)},
		{
			"a nanosecond before the afternoon",
			inWest(2, 15, 0).Add(-time.Nanosecond),
			inWest(2, 15, 0),
		},
		{"at the afternoon slot", inWest(2, 15, 0), inWest(3, 10, 0)},
		{"in the evening", inWest(2, 23, 59), inWest(3, 10, 0)},
		{"at midnight", inWest(3, 0, 0), inWest(3, 10, 0)},
		{"on Friday afternoon", inWest(6, 15, 0), inWest(9, 10, 0)},
		{"on Saturday morning", inWest(7, 10, 0), inWest(9, 10, 0)},
		{"on Sunday evening", inWest(8, 23, 0), inWest(9, 10, 0)},
		{"on Monday afternoon in west, Tuesday in UTC", inWest(2, 12, 30).UTC(), inWest(2, 15, 0)},
		{"on Friday afternoon in west, Saturday in UTC", inWest(6, 12, 30).UTC(), inWest(6, 15, 0)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := nextSlot(test.from, west); !got.Equal(test.wantSlot) {
				t.Errorf("got %v, want %v", got, test.wantSlot)
			}
		})
	}
}

func TestNextSlotKeepsLocalTimeOverADaylightSavingChange(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	// New York's clocks went forward an hour on Sunday 2026-03-08.
	from := time.Date(2026, time.March, 6, 15, 0, 0, 0, newYork)
	want := time.Date(2026, time.March, 9, 10, 0, 0, 0, newYork)

	if got := nextSlot(from, newYork); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLater(t *testing.T) {
	early := time.Date(2026, time.November, 2, 10, 0, 0, 0, time.UTC)
	late := early.Add(time.Nanosecond)

	for _, pair := range [][2]time.Time{{early, late}, {late, early}} {
		if got := later(pair[0], pair[1]); !got.Equal(late) {
			t.Errorf("later(%v, %v): got %v, want %v", pair[0], pair[1], got, late)
		}
	}
}
