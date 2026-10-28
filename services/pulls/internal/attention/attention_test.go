package attention_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/pulls/internal/attention"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

func TestListItemsListsThePulls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mine := started(t, func(context.Context) ([]bitbucket.Pull, error) {
			return []bitbucket.Pull{
				{ID: 7, Title: "Fix the login", Repo: "acme/web", URL: "https://example.com/7"},
				{ID: 12, Title: "Add the export", Repo: "acme/api", Draft: true},
			}, nil
		})

		got := list(t, mine)

		want := []string{
			"acme/web#7 | Fix the login | acme/web #7 | SOURCE_BITBUCKET | no due",
			"acme/api#12 | Add the export | acme/api #12 · Draft | SOURCE_BITBUCKET | no due",
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestListItemsListsNoneWithoutBitbucket(t *testing.T) {
	if got := list(t, nil); len(got) != 0 {
		t.Errorf("got %q, want no items", got)
	}
}

func TestListItemsListsNoneBeforeAPollSucceeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mine := started(t, func(context.Context) ([]bitbucket.Pull, error) {
			return nil, errors.New("bitbucket is down")
		})

		if got := list(t, mine); len(got) != 0 {
			t.Errorf("got %q, want no items", got)
		}
	})
}

func TestListItemsKeepsThePullsOfTheLastPollToSucceed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		polls := 0
		mine := started(t, func(context.Context) ([]bitbucket.Pull, error) {
			polls++
			if polls > 1 {
				return nil, errors.New("bitbucket is down")
			}

			return []bitbucket.Pull{{ID: 7, Title: "Fix the login", Repo: "acme/web"}}, nil
		})

		list(t, mine)
		time.Sleep(5 * time.Minute)
		synctest.Wait()

		want := []string{"acme/web#7 | Fix the login | acme/web #7 | SOURCE_BITBUCKET | no due"}
		if got := list(t, mine); polls != 2 || !slices.Equal(got, want) {
			t.Errorf("after %d polls, got %q, want %q", polls, got, want)
		}
	})
}

func TestListItemsWaitsForTheFirstPoll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		mine := started(t, func(context.Context) ([]bitbucket.Pull, error) {
			<-release

			return []bitbucket.Pull{{ID: 7, Title: "Fix the login", Repo: "acme/web"}}, nil
		})

		type answer struct {
			res *attentionv1.ListItemsResponse
			err error
		}

		listed := make(chan answer)

		go func() {
			res, err := attention.NewLister(mine).ListItems(t.Context(),
				&attentionv1.ListItemsRequest{})
			listed <- answer{res, err}
		}()

		synctest.Wait()

		select {
		case got := <-listed:
			t.Fatalf("answered %v before the first poll ended", got)
		default:
		}

		close(release)

		if got := <-listed; got.err != nil || len(got.res.GetItems()) != 1 {
			t.Errorf("got %v, want the pull request", got)
		}
	})
}

func TestListItemsStopsWaitingWhenItsContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mine := started(t, func(ctx context.Context) ([]bitbucket.Pull, error) {
			<-ctx.Done()

			return nil, ctx.Err()
		})

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		res, err := attention.NewLister(mine).ListItems(ctx, &attentionv1.ListItemsRequest{})
		if !errors.Is(err, context.DeadlineExceeded) || res != nil {
			t.Errorf("got %v and %v, want no answer and the context's error", res, err)
		}
	})
}

// started returns a poller that lists the user's pull requests with mine, and runs it until the
// test ends.
func started(
	t *testing.T,
	mine func(context.Context) ([]bitbucket.Pull, error),
) *bitbucket.Poller {
	t.Helper()

	poller := bitbucket.NewPoller(mine, slog.New(slog.DiscardHandler))
	go poller.Run(t.Context())

	return poller
}

// list returns the items that a lister of what mine has found lists, one line each.
func list(t *testing.T, mine *bitbucket.Poller) []string {
	t.Helper()

	res, err := attention.NewLister(mine).ListItems(t.Context(), &attentionv1.ListItemsRequest{})
	if err != nil {
		t.Fatal(err)
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

	return lines
}
