package tokenexpiry_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/internal/tokenexpiry"
)

func TestWatchTellsTheLatestExpiryThatAnAskFound(t *testing.T) {
	const failure = "ask GitHub when the token expires: 502 Bad Gateway"

	expires := time.Date(2027, time.October, 27, 22, 0, 0, 0, time.UTC)
	answers := []error{errors.New(failure), nil, errors.New(failure)}

	var calls atomic.Int32

	ask := func(context.Context) (time.Time, error) {
		err := answers[min(int(calls.Add(1))-1, len(answers)-1)]
		if err != nil {
			return time.Time{}, err
		}

		return expires, nil
	}

	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer

		expiry := tokenexpiry.Watch(t.Context(), "GitHub token", ask,
			slog.New(slog.NewTextHandler(&logs, nil)))

		// The first ask fails, the second succeeds, and the third fails.
		for _, test := range []struct {
			wait time.Duration
			want time.Time
		}{
			{0, time.Time{}},
			{5 * time.Minute, expires},
			{5 * time.Minute, expires},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			got, err := expiry(t.Context())
			if err != nil || !got.Equal(test.want) {
				t.Errorf("at %s: got %v and %v, want %v", time.Now().UTC(), got, err, test.want)
			}
		}

		want := `level=ERROR msg="poll GitHub token" err="` + failure + `"`
		if lines := strings.Split(strings.TrimSuffix(logs.String(), "\n"), "\n"); len(lines) != 2 ||
			!strings.HasSuffix(lines[0], want) || !strings.HasSuffix(lines[1], want) {
			t.Errorf("got the logs %q, want two lines that each end %q", logs.String(), want)
		}
	})
}

func TestWatchedExpiryFailsWhenItsCallEndsBeforeTheFirstAsk(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// The first ask lasts until the test ends.
		ask := func(ctx context.Context) (time.Time, error) {
			<-ctx.Done()

			return time.Time{}, ctx.Err()
		}
		expiry := tokenexpiry.Watch(t.Context(), "GitHub token", ask,
			slog.New(slog.DiscardHandler))

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		got, err := expiry(ctx)
		if !errors.Is(err, context.DeadlineExceeded) || !got.IsZero() {
			t.Errorf("got %v and %v, want the call's context.DeadlineExceeded", got, err)
		}
	})
}
