package attention_test

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// unavailable is the error of a call before all three searches have succeeded once.
const unavailable = "unavailable: GitHub's searches haven't all succeeded yet"

// beta, login and typo are a PR of mine, a review request and an assigned issue, and all are the
// lines of their items.
var (
	beta  = github.Item{ID: 5, Title: "Ship the beta", Repo: "acme/web"}
	login = github.Item{ID: 7, Title: "Fix the login", Repo: "acme/web"}
	typo  = github.Item{ID: 3, Title: "Fix the typo", Repo: "acme/docs"}
	all   = []string{
		"acme/web#5 | Ship the beta | acme/web #5 | SOURCE_GITHUB | no due",
		"acme/web#7 | Fix the login | acme/web #7 · Review requested | SOURCE_GITHUB | no due",
		"acme/docs#3 | Fix the typo | acme/docs #3 · Assigned | SOURCE_GITHUB | no due",
	}
)

func TestListItemsIsUnavailableUntilAllThreeSearchesSucceed(t *testing.T) {
	tests := map[string]struct{ mine, reviews, issues search }{
		"mine":    {mine: failsFirst(beta), reviews: found(login), issues: found(typo)},
		"reviews": {mine: found(beta), reviews: failsFirst(login), issues: found(typo)},
		"issues":  {mine: found(beta), reviews: found(login), issues: failsFirst(typo)},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				mine, reviews := started(t, test.mine), started(t, test.reviews)
				issues := started(t, test.issues)

				got, err := list(t.Context(), mine, reviews, issues)
				if err == nil || err.Error() != unavailable {
					t.Errorf("got %q and %v, want the error %s", got, err, unavailable)
				}

				time.Sleep(5 * time.Minute)
				synctest.Wait()

				got, err = list(t.Context(), mine, reviews, issues)
				if err != nil || !slices.Equal(got, all) {
					t.Errorf("after the second searches, got %q and %v, want %q", got, err, all)
				}
			})
		})
	}
}

func TestListItemsKeepsTheItemsOfTheLastSearchesToSucceed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var searches atomic.Int32

		mine := started(t, failsLater(&searches, beta))
		reviews := started(t, failsLater(&searches, login))
		issues := started(t, failsLater(&searches, typo))

		time.Sleep(5 * time.Minute)
		synctest.Wait()

		got, err := list(t.Context(), mine, reviews, issues)
		if searches.Load() != 6 || err != nil || !slices.Equal(got, all) {
			t.Errorf("after %d searches, got %q and %v, want %q", searches.Load(), got, err,
				all)
		}
	})
}

func TestListItemsWaitsForTheFirstSearches(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		issues := started(t, func(context.Context) ([]github.Item, error) {
			<-release

			return []github.Item{typo}, nil
		})

		type answer struct {
			lines []string
			err   error
		}

		listed := make(chan answer)

		go func() {
			lines, err := list(t.Context(), started(t, found(beta)), started(t, found(login)),
				issues)
			listed <- answer{lines, err}
		}()

		synctest.Wait()

		select {
		case got := <-listed:
			t.Fatalf("answered %v before the first search of the issues ended", got)
		default:
		}

		close(release)

		if got := <-listed; got.err != nil || !slices.Equal(got.lines, all) {
			t.Errorf("got %v, want %q", got, all)
		}
	})
}

func TestListItemsStopsWaitingWhenItsContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reviews := started(t, func(ctx context.Context) ([]github.Item, error) {
			<-ctx.Done()

			return nil, ctx.Err()
		})

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		got, err := list(ctx, started(t, found(beta)), reviews, started(t, found(typo)))
		if !errors.Is(err, context.DeadlineExceeded) || got != nil {
			t.Errorf("got %q and %v, want no items and the context's error", got, err)
		}
	})
}

// failsFirst returns a search that fails the first time and finds item after that.
func failsFirst(item github.Item) search {
	var tries atomic.Int32

	return func(context.Context) ([]github.Item, error) {
		if tries.Add(1) == 1 {
			return nil, errors.New("github is down")
		}

		return []github.Item{item}, nil
	}
}

// failsLater returns a search that finds item the first time and fails after that, and counts
// its searches in searches.
func failsLater(searches *atomic.Int32, item github.Item) search {
	var tries atomic.Int32

	return func(context.Context) ([]github.Item, error) {
		searches.Add(1)

		if tries.Add(1) > 1 {
			return nil, errors.New("github is down")
		}

		return []github.Item{item}, nil
	}
}
