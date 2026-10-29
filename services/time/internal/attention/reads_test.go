package attention_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

// unavailable is the error of a call before a read of the worklogs has succeeded.
const unavailable = "unavailable: no read of the worklogs on Jira has succeeded yet"

func TestListItemsIsUnavailableUntilAReadSucceeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reads := 0
		reports := started(t, hawaii,
			func(context.Context, time.Time, time.Time) ([]jira.Worklog, error) {
				reads++
				if reads == 1 {
					return nil, errors.New("jira is down")
				}

				return onTuesday, nil
			})

		got, err := list(t.Context(), reports, hawaii)
		if err == nil || err.Error() != unavailable {
			t.Errorf("got %q and %v, want the error %s", got, err, unavailable)
		}

		time.Sleep(5 * time.Minute)
		synctest.Wait()

		got, err = list(t.Context(), reports, hawaii)
		if err != nil || !slices.Equal(got, withTuesday) {
			t.Errorf("after %d reads, got %q and %v, want %q", reads, got, err, withTuesday)
		}
	})
}

func TestListItemsKeepsTheItemsOfTheLastReadToSucceed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reads := 0
		reports := started(t, hawaii,
			func(context.Context, time.Time, time.Time) ([]jira.Worklog, error) {
				reads++
				if reads > 1 {
					return nil, errors.New("jira is down")
				}

				return onTuesday, nil
			})

		time.Sleep(5 * time.Minute)
		synctest.Wait()

		got, err := list(t.Context(), reports, hawaii)
		if reads != 2 || err != nil || !slices.Equal(got, withTuesday) {
			t.Errorf("after %d reads, got %q and %v, want %q", reads, got, err, withTuesday)
		}
	})
}

func TestListItemsWaitsForTheFirstRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		reports := started(t, hawaii,
			func(context.Context, time.Time, time.Time) ([]jira.Worklog, error) {
				<-release

				return onTuesday, nil
			})

		answered := make(chan string)

		go func() {
			got, err := list(t.Context(), reports, hawaii)
			answered <- fmt.Sprintf("%d items %v", len(got), err)
		}()

		synctest.Wait()

		select {
		case got := <-answered:
			t.Fatalf("answered %s before the first read ended", got)
		default:
		}

		close(release)

		if got := <-answered; got != "4 items <nil>" {
			t.Errorf("got %s, want 4 items", got)
		}
	})
}

func TestListItemsStopsWaitingWhenItsContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports := started(t, hawaii,
			func(ctx context.Context, _, _ time.Time) ([]jira.Worklog, error) {
				<-ctx.Done()

				return nil, ctx.Err()
			})

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		got, err := list(ctx, reports, hawaii)
		if !errors.Is(err, context.DeadlineExceeded) || got != nil {
			t.Errorf("got %q and %v, want no answer and the context's error", got, err)
		}
	})
}
