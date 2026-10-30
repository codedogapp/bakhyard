package attention_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/notify/internal/attention"
	"github.com/codedogapp/bakhyard/services/notify/internal/history"
)

func TestListItemsListsTheAlertsThatFire(t *testing.T) {
	tokyo := time.FixedZone("Tokyo", 9*60*60)
	entries := []history.Entry{
		{
			Fingerprint: "a1b2",
			Name:        "OnBattery",
			Summary:     "The laptop runs on its battery",
			Description: "Plug it in",
			StartedAt:   time.Date(2026, 10, 30, 22, 0, 0, 512304000, tokyo),
		},
		{
			Fingerprint: "c3d4",
			Name:        "DiskFull",
			StartedAt:   time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC),
		},
		{
			Fingerprint: "e5f6",
			Name:        "CPUHot",
			Summary:     "The CPU runs hot",
			StartedAt:   time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC),
			ResolvedAt:  time.Date(2026, 10, 29, 8, 15, 0, 0, time.UTC),
		},
	}

	got, err := list(t, func(context.Context) ([]history.Entry, error) { return entries, nil })

	want := []string{
		"a1b2@2026-10-30T13:00:00.512304Z | The laptop runs on its battery | Plug it in | " +
			"SOURCE_UNSPECIFIED | no due",
		"c3d4@2026-10-30T12:00:00Z | DiskFull |  | SOURCE_UNSPECIFIED | no due",
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %q (%v), want %q", got, err, want)
	}
}

func TestListItemsListsNoneWhenNoAlertFires(t *testing.T) {
	got, err := list(t, func(context.Context) ([]history.Entry, error) { return nil, nil })
	if err != nil || len(got) != 0 {
		t.Errorf("got %q (%v), want no items", got, err)
	}
}

func TestListItemsFailsWhenTheAlertsCantBeRead(t *testing.T) {
	failure := errors.New("the database is down")

	got, err := list(t, func(context.Context) ([]history.Entry, error) { return nil, failure })
	if !errors.Is(err, failure) || got != nil {
		t.Errorf("got %q (%v), want the failure", got, err)
	}
}

// list returns the items that a lister of what alerts lists lists, one line each, or its error.
func list(t *testing.T, alerts func(context.Context) ([]history.Entry, error)) ([]string, error) {
	t.Helper()

	res, err := attention.NewLister(alerts).ListItems(t.Context(), &attentionv1.ListItemsRequest{})
	if err != nil {
		return nil, err
	}

	lines := []string{}

	for _, item := range res.GetItems() {
		due := "no due"
		if item.HasDue() {
			due = item.GetDue().AsTime().String()
		}

		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | %s", item.GetId(),
			item.GetTitle(), item.GetDetail(), item.GetSource(), due))
	}

	return lines, nil
}
