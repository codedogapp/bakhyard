package demo_test

import (
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/home/internal/demo"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

func TestItemsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}

	for _, item := range demo.Items(time.Now()) {
		if seen[item.ID] {
			t.Errorf("ID %q is used twice", item.ID)
		}

		seen[item.ID] = true

		if prefix := string(item.App) + "/"; !strings.HasPrefix(item.ID, prefix) {
			t.Errorf("ID %q does not start with %q", item.ID, prefix)
		}

		if item.Title == "" || item.Source == "" {
			t.Errorf("%s: title %q, source %q, want both set", item.ID, item.Title, item.Source)
		}
	}
}

func TestItemsAreDueRelativeToNow(t *testing.T) {
	// Far from the real clock, so the test fails if Items ignores now.
	now := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

	var late, undated int

	for _, item := range demo.Items(now) {
		switch {
		case item.Due.IsZero():
			undated++
		case item.Due.Before(now):
			late++
		}
	}

	if late != 2 || undated != 2 {
		t.Errorf("late = %d, undated = %d, want 2 and 2", late, undated)
	}
}

func TestWeek(t *testing.T) {
	want := screen.Week{LoggedHours: 31.5, TargetHours: 40}

	if got := demo.Week(); got != want {
		t.Errorf("week = %+v, want %+v", got, want)
	}
}
