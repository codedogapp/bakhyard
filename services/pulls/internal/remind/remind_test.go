package remind_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/internal/notifier"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

func TestReminderRemindsAtEachSlotOfTheWeek(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pushes := &pusher{}
		start(t.Context(), listing(pull(1, "Fix the login", opened)), needing(0, 1, 1).read, pushes)

		time.Sleep(time.Until(at(7, 0, 0)))
		synctest.Wait()

		var want []time.Time
		for day := range 5 {
			want = append(want, at(day, 10, 0), at(day, 15, 0))
		}

		wantTimes(t, pushes, want...)
	})
}

func TestReminderListsThePullsThatNeedApprovals(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		slot := at(0, 10, 0)
		draft := pull(2, "Try a theme", opened)
		draft.Draft = true
		reads := &approvals{answers: map[int]bitbucket.Approvals{
			1: {Given: 1, Required: 2}, 2: {Required: 1}, 3: {Required: 1}, 4: {Required: 1},
			5: {Given: 2, Required: 2}, 6: {Given: 3, Required: 2}, 8: {Given: 2, Required: 3},
		}}
		pushes := &pusher{}
		logs, _ := start(t.Context(), listing(
			pull(1, "Fix the login", opened), draft,
			pull(3, "Open for a day", slot.Add(-24*time.Hour)),
			pull(4, "Open for a day and a moment", slot.Add(-24*time.Hour-time.Nanosecond)),
			pull(5, "Approved", opened), pull(6, "Approved twice over", opened),
			pull(7, "Unreadable", opened), pull(8, "Add a page", opened),
		), reads.read, pushes)

		time.Sleep(time.Until(slot.Add(time.Hour)))
		synctest.Wait()

		want := notifier.Message{
			Title: "Remind your reviewers",
			Body: "Fix the login (acme/web #1): 1 of 2 approvals\n" +
				"Open for a day and a moment (acme/web #4): 0 of 1 approval\n" +
				"Add a page (acme/web #8): 2 of 3 approvals",
			URL: "/pulls",
		}
		if got := pushes.messages(); !slices.Equal(got, []notifier.Message{want}) {
			t.Errorf("got %q, want one push of %q", got, want)
		}

		wantTimes(t, pushes, slot)

		if want := []int{1, 4, 5, 6, 7, 8}; !slices.Equal(reads.asked, want) {
			t.Errorf("asked the approvals of %v, want %v", reads.asked, want)
		}

		wantLogs(t, logs.String(), `level=WARN msg="check a PR for the reminder" `+
			`err="read the merge checks of acme/web #7: 500 Internal Server Error"`)
	})
}

func TestReminderPushesNothingWhenNoPullNeedsApprovals(t *testing.T) {
	failing := func(context.Context) ([]bitbucket.Pull, error) {
		return nil, errors.New("list the pull requests: 500 Internal Server Error")
	}

	for name, mine := range map[string]func(context.Context) ([]bitbucket.Pull, error){
		"no pull request":    listing(),
		"only approved ones": listing(pull(1, "Fix the login", opened)),
		"no poll succeeded":  failing,
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				pushes := &pusher{}
				start(t.Context(), mine, needing(2, 2, 1).read, pushes)

				time.Sleep(time.Until(at(1, 0, 0)))
				synctest.Wait()

				wantTimes(t, pushes)
			})
		})
	}
}

func TestReminderRemindsOfTheLatestPollThatSucceeded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var polls atomic.Int32

		mine := func(context.Context) ([]bitbucket.Pull, error) {
			if polls.Add(1) == 1 {
				return []bitbucket.Pull{pull(1, "Fix the login", opened)}, nil
			}

			return nil, errors.New("list the pull requests: 500 Internal Server Error")
		}
		pushes := &pusher{}
		start(t.Context(), mine, needing(0, 1, 1).read, pushes)

		time.Sleep(time.Until(at(0, 11, 0)))
		synctest.Wait()

		wantTimes(t, pushes, at(0, 10, 0))
	})
}

func TestReminderGoesOnAfterAPushFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pushes := &pusher{err: errors.New("send the notification: 502 Bad Gateway")}
		logs, _ := start(t.Context(), listing(pull(1, "Fix the login", opened)),
			needing(0, 1, 1).read, pushes)

		time.Sleep(time.Until(at(0, 16, 0)))
		synctest.Wait()

		wantTimes(t, pushes, at(0, 10, 0), at(0, 15, 0))

		line := `level=WARN msg="push the PR reminder" err="send the notification: 502 Bad Gateway"`
		wantLogs(t, logs.String(), line, line)
	})
}

func TestReminderWaitsForTheFirstPoll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mine := func(context.Context) ([]bitbucket.Pull, error) {
			time.Sleep(time.Until(at(0, 10, 30)))

			return []bitbucket.Pull{
				pull(1, "Fix the login", opened),
				// A day old by the time the poll ends, but not at the slot.
				pull(2, "Add a page", at(0, 10, 10).Add(-24*time.Hour)),
			}, nil
		}
		pushes := &pusher{}
		start(t.Context(), mine, needing(0, 1, 1, 2).read, pushes)

		time.Sleep(time.Until(at(0, 11, 0)))
		synctest.Wait()

		wantTimes(t, pushes, at(0, 10, 30))

		want := "Fix the login (acme/web #1): 0 of 1 approval"
		if got := pushes.messages(); len(got) != 1 || got[0].Body != want {
			t.Errorf("got %q, want one push of %q", got, want)
		}
	})
}

func TestReminderRunningLateSkipsTheSlotsItMissed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var reads atomic.Int32

		// The first read takes until after the afternoon slot.
		read := func(context.Context, bitbucket.Pull) (bitbucket.Approvals, error) {
			if reads.Add(1) == 1 {
				time.Sleep(time.Until(at(0, 15, 30)))
			}

			return bitbucket.Approvals{Required: 1}, nil
		}
		pushes := &pusher{}
		start(t.Context(), listing(pull(1, "Fix the login", opened)), read, pushes)

		time.Sleep(time.Until(at(1, 11, 0)))
		synctest.Wait()

		wantTimes(t, pushes, at(0, 15, 30), at(1, 10, 0))
	})
}

// wantLogs checks that logs has one line for each of want, in order, which ends with it.
func wantLogs(t *testing.T, logs string, want ...string) {
	t.Helper()

	endsWith := func(line, end string) bool { return strings.HasSuffix(line, end+"\n") }
	if !slices.EqualFunc(slices.Collect(strings.Lines(logs)), want, endsWith) {
		t.Errorf("got the logs %q, want a line that ends with each of %q", logs, want)
	}
}
