package attention_test

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"testing"
	"testing/synctest"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/github/internal/attention"
	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

func TestListItemsListsTheReviewRequestsThenTheIssues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reviews := started(t, found(
			github.Item{
				ID:    7,
				Title: "Fix the login",
				Repo:  "acme/web",
				URL:   "https://example.com",
			},
			github.Item{ID: 12, Title: "Add the export", Repo: "acme/api", Draft: true},
		))
		issues := started(t, found(github.Item{ID: 3, Title: "Fix the typo", Repo: "acme/docs"}))

		got, err := list(t.Context(), reviews, issues)

		want := []string{
			"acme/web#7 | Fix the login | acme/web #7 · Review requested | SOURCE_GITHUB | no due",
			"acme/api#12 | Add the export | acme/api #12 · Review requested · Draft | " +
				"SOURCE_GITHUB | no due",
			"acme/docs#3 | Fix the typo | acme/docs #3 · Assigned | SOURCE_GITHUB | no due",
		}
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("got %q and %v, want %q", got, err, want)
		}
	})
}

func TestListItemsListsNoneWhenTheSearchesFindNone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		got, err := list(t.Context(), started(t, found()), started(t, found()))
		if err != nil || len(got) != 0 {
			t.Errorf("got %q and %v, want no items", got, err)
		}
	})
}

func TestListItemsListsNoneWithoutGitHub(t *testing.T) {
	got, err := list(t.Context(), nil, nil)
	if err != nil || len(got) != 0 {
		t.Errorf("got %q and %v, want no items", got, err)
	}
}

// search looks one of the lists up on GitHub, as a Client's Reviews does.
type search = func(context.Context) ([]github.Item, error)

// started returns a poller that lists items with search, and runs it until the test ends.
func started(t *testing.T, search search) *github.Poller {
	t.Helper()

	poller := github.NewPoller("GitHub", search, slog.New(slog.DiscardHandler))
	go poller.Run(t.Context())

	return poller
}

// found returns a search that finds items each time.
func found(items ...github.Item) search {
	return func(context.Context) ([]github.Item, error) {
		return items, nil
	}
}

// list returns the items that a lister of what reviews and issues have found lists, one line
// each, or its error.
func list(ctx context.Context, reviews, issues *github.Poller) ([]string, error) {
	res, err := attention.NewLister(reviews, issues).ListItems(ctx,
		&attentionv1.ListItemsRequest{})
	if err != nil {
		return nil, err
	}

	var lines []string

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
