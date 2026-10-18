package screen_test

import (
	"slices"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

func TestNewOrdersByUrgency(t *testing.T) {
	now := time.Date(2026, time.October, 18, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		items []screen.Item
		want  []string
	}{
		{
			name: "earliest due time first",
			items: []screen.Item{
				{ID: "a/soon", Due: now.Add(time.Hour)},
				{ID: "a/late", Due: now.Add(-time.Hour)},
				{ID: "a/later", Due: now.Add(2 * time.Hour)},
			},
			want: []string{"a/late", "a/soon", "a/later"},
		},
		{
			name: "no due time last",
			items: []screen.Item{
				{ID: "a/due", Due: now.Add(time.Hour)},
				{ID: "a/none"},
				{ID: "a/later", Due: now.Add(2 * time.Hour)},
			},
			want: []string{"a/due", "a/later", "a/none"},
		},
		{
			name: "same due time by title",
			items: []screen.Item{
				{ID: "a/1", Title: "b", Due: now},
				{ID: "a/2", Title: "a", Due: now},
			},
			want: []string{"a/2", "a/1"},
		},
		{
			name: "same title by ID",
			items: []screen.Item{
				{ID: "a/2", Title: "same"},
				{ID: "a/1", Title: "same"},
			},
			want: []string{"a/1", "a/2"},
		},
		{
			name:  "no items",
			items: nil,
			want:  []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home := screen.New(test.items, screen.Week{}, now)

			if got := ids(home.Items); !slices.Equal(got, test.want) {
				t.Errorf("order = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNewMarksLateItems(t *testing.T) {
	now := time.Date(2026, time.October, 18, 9, 0, 0, 0, time.UTC)
	items := []screen.Item{
		{ID: "a/past", Due: now.Add(-time.Nanosecond)},
		{ID: "a/now", Due: now},
		{ID: "a/future", Due: now.Add(time.Hour)},
		{ID: "a/none"},
	}
	want := map[string]bool{"a/past": true, "a/now": false, "a/future": false, "a/none": false}

	for _, item := range screen.New(items, screen.Week{}, now).Items {
		if item.Late != want[item.ID] {
			t.Errorf("%s: late = %t, want %t", item.ID, item.Late, want[item.ID])
		}
	}
}

func TestNewKeepsInputAndWeek(t *testing.T) {
	now := time.Date(2026, time.October, 18, 9, 0, 0, 0, time.UTC)
	items := []screen.Item{
		{ID: "a/2", Due: now.Add(-time.Hour)},
		{ID: "a/1", Due: now.Add(-2 * time.Hour)},
	}
	week := screen.Week{LoggedHours: 1, TargetHours: 2}

	home := screen.New(items, week, now)

	if got := ids(items); !slices.Equal(got, []string{"a/2", "a/1"}) {
		t.Errorf("input order = %v, want it unchanged", got)
	}

	if items[0].Late || items[1].Late {
		t.Error("input items marked late, want them unchanged")
	}

	if home.Week != week {
		t.Errorf("week = %+v, want %+v", home.Week, week)
	}
}

func ids(items []screen.Item) []string {
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.ID)
	}

	return got
}
