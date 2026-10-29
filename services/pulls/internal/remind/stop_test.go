package remind_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

func TestReminderStopsWithoutPushingOrWarning(t *testing.T) {
	two := listing(pull(1, "Fix the login", opened), pull(2, "Add a page", opened))
	// The approvals of the first pull request come at once, those of the second only at the stop.
	secondHeld := func(ctx context.Context, pull bitbucket.Pull) (bitbucket.Approvals, error) {
		if pull.ID == 2 {
			return bitbucket.Approvals{}, held(ctx)
		}

		return bitbucket.Approvals{Required: 1}, nil
	}
	heldPoll := func(ctx context.Context) ([]bitbucket.Pull, error) { return nil, held(ctx) }

	tests := []struct {
		name   string
		mine   func(context.Context) ([]bitbucket.Pull, error)
		read   func(context.Context, bitbucket.Pull) (bitbucket.Approvals, error)
		pusher *pusher
		stop   time.Time
		// pushes is how many pushes began before the stop.
		pushes int
	}{
		{"while it waits for a slot", two, needing(0, 1, 1, 2).read, &pusher{}, at(0, 9, 0), 0},
		{"while the first poll runs", heldPoll, needing(0, 1).read, &pusher{}, at(0, 11, 0), 0},
		{"while it reads the approvals", two, secondHeld, &pusher{}, at(0, 11, 0), 0},
		{"while it pushes", two, needing(0, 1, 1, 2).read, &pusher{hold: true}, at(0, 11, 0), 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, stop := context.WithCancel(t.Context())
				logs, stopped := start(ctx, test.mine, test.read, test.pusher)

				time.Sleep(time.Until(test.stop))
				stop()
				synctest.Wait()

				select {
				case <-stopped:
				default:
					t.Error("Run goes on after the stop")
				}

				if got := len(test.pusher.times()); got != test.pushes {
					t.Errorf("began %d pushes, want %d", got, test.pushes)
				}

				wantLogs(t, logs.String())
			})
		})
	}
}
