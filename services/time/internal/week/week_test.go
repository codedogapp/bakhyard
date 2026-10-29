package week_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	timev1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1"
	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/time/internal/week"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// A bubble's clock starts at midnight UTC on Saturday, 1 January 2000, and the pollers' zone is
// UTC, so the week starts on Monday, 27 December 1999, and the report's days on Sunday, 26th.

// logged are the worklogs that the reads find, unless they fail: an hour last week and an hour
// and a half this week.
var logged = []jira.Worklog{
	{Started: time.Date(1999, 12, 26, 9, 0, 0, 0, time.UTC), Spent: time.Hour},
	{Started: time.Date(1999, 12, 27, 9, 0, 0, 0, time.UTC), Spent: 90 * time.Minute},
}

// unavailable is the error of a call before a read of the worklogs has succeeded.
const unavailable = "unavailable: no read of the worklogs on Jira has succeeded yet"

func TestGetWeekAnswersTheWeek(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports := started(t, func(context.Context, time.Time, time.Time) ([]jira.Worklog, error) {
			return logged, nil
		})

		if got, err := get(t.Context(), reports); err != nil || got != "1.5 of 40 hours" {
			t.Errorf("got %q and %v, want 1.5 of 40 hours", got, err)
		}
	})
}

func TestGetWeekAnswersNoWeekWithoutJira(t *testing.T) {
	if got, err := get(t.Context(), nil); err != nil || got != "no week" {
		t.Errorf("got %q and %v, want no week", got, err)
	}
}

func TestGetWeekIsUnavailableUntilAReadSucceeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reads := 0
		reports := started(t, func(context.Context, time.Time, time.Time) ([]jira.Worklog, error) {
			reads++
			if reads == 1 {
				return nil, errors.New("jira is down")
			}

			return logged, nil
		})

		if got, err := get(t.Context(), reports); err == nil || err.Error() != unavailable {
			t.Errorf("got %q and %v, want the error %s", got, err, unavailable)
		}

		time.Sleep(5 * time.Minute)
		synctest.Wait()

		if got, err := get(t.Context(), reports); err != nil || got != "1.5 of 40 hours" {
			t.Errorf("after %d reads, got %q and %v, want 1.5 of 40 hours", reads, got, err)
		}
	})
}

func TestGetWeekKeepsTheWeekOfTheLastReadToSucceed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reads := 0
		reports := started(t, func(context.Context, time.Time, time.Time) ([]jira.Worklog, error) {
			reads++
			if reads > 1 {
				return nil, errors.New("jira is down")
			}

			return logged, nil
		})

		time.Sleep(5 * time.Minute)
		synctest.Wait()

		got, err := get(t.Context(), reports)
		if reads != 2 || err != nil || got != "1.5 of 40 hours" {
			t.Errorf("after %d reads, got %q and %v, want 1.5 of 40 hours", reads, got, err)
		}
	})
}

func TestGetWeekWaitsForTheFirstRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		reports := started(t, func(context.Context, time.Time, time.Time) ([]jira.Worklog, error) {
			<-release

			return logged, nil
		})

		answered := make(chan string)

		go func() {
			got, err := get(t.Context(), reports)
			answered <- fmt.Sprintf("%s %v", got, err)
		}()

		synctest.Wait()

		select {
		case got := <-answered:
			t.Fatalf("answered %s before the first read ended", got)
		default:
		}

		close(release)

		if got := <-answered; got != "1.5 of 40 hours <nil>" {
			t.Errorf("got %s, want 1.5 of 40 hours", got)
		}
	})
}

func TestGetWeekStopsWaitingWhenItsContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports := started(t, func(ctx context.Context, _, _ time.Time) ([]jira.Worklog, error) {
			<-ctx.Done()

			return nil, ctx.Err()
		})

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		res, err := week.NewServer(reports).GetWeek(ctx, &timev1.GetWeekRequest{})
		if !errors.Is(err, context.DeadlineExceeded) || res != nil {
			t.Errorf("got %v and %v, want no answer and the context's error", res, err)
		}
	})
}

// started returns a poller in UTC that reads the worklogs with read, and runs it until the test
// ends.
func started(
	t *testing.T,
	read func(ctx context.Context, start, end time.Time) ([]jira.Worklog, error),
) *worklogs.Poller {
	t.Helper()

	poller := worklogs.NewPoller(read, time.UTC, slog.New(slog.DiscardHandler))
	go poller.Run(t.Context())

	return poller
}

// get returns how the week that a server of reports answers reads, "1.5 of 40 hours" or "no
// week", or the error that it returns.
func get(ctx context.Context, reports *worklogs.Poller) (string, error) {
	res, err := week.NewServer(reports).GetWeek(ctx, &timev1.GetWeekRequest{})
	if err != nil {
		return "", err
	}

	if !res.HasWeek() {
		return "no week", nil
	}

	return fmt.Sprintf("%g of %g hours", res.GetWeek().GetLoggedHours(),
		res.GetWeek().GetTargetHours()), nil
}
