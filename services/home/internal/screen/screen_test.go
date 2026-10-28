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
			home := screen.New("", test.items, now)

			if got := ids(home.Items); !slices.Equal(got, test.want) {
				t.Errorf("order = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNewName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "kept", input: "Ada", want: "Ada"},
		{name: "inner spaces kept", input: "Ada Lovelace", want: "Ada Lovelace"},
		{name: "non-ASCII kept", input: "Zoë", want: "Zoë"},
		{name: "surrounding whitespace trimmed", input: " \tAda\n", want: "Ada"},
		{name: "invalid UTF-8 replaced", input: "Ad\xffa", want: "Ad\uFFFDa"},
		{name: "empty falls back", input: "", want: "there"},
		{name: "blank falls back", input: " \t\n\u00a0", want: "there"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home := screen.New(test.input, nil, time.Time{})

			if home.Name != test.want {
				t.Errorf("name = %q, want %q", home.Name, test.want)
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

	for _, item := range screen.New("", items, now).Items {
		if item.Late != want[item.ID] {
			t.Errorf("%s: late = %t, want %t", item.ID, item.Late, want[item.ID])
		}
	}
}

func TestNewKeepsTheInput(t *testing.T) {
	now := time.Date(2026, time.October, 18, 9, 0, 0, 0, time.UTC)
	items := []screen.Item{
		{ID: "a/2", Due: now.Add(-time.Hour)},
		{ID: "a/1", Due: now.Add(-2 * time.Hour)},
	}

	screen.New("", items, now)

	if got := ids(items); !slices.Equal(got, []string{"a/2", "a/1"}) {
		t.Errorf("input order = %v, want it unchanged", got)
	}

	if items[0].Late || items[1].Late {
		t.Error("input items marked late, want them unchanged")
	}
}

func ids(items []screen.Item) []string {
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.ID)
	}

	return got
}
