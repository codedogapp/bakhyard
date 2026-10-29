package week

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	timev1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// These tests run in a bubble, whose fake clock moves only once every goroutine in it waits, so
// they can tell how long Ask waits. Real connections don't wait that way, hence the fake client.

func TestAskWaitsForTimeOnlyUntilTheTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer

		asker := Asker{client: silent{}, logger: slog.New(slog.NewTextHandler(&logs, nil))}

		start := time.Now()
		got, state := asker.Ask(t.Context())

		if waited, want := time.Since(start), 1500*time.Millisecond; waited != want {
			t.Errorf("Ask waited %v, want %v", waited, want)
		}

		if got != nil || state != screen.StateUnavailable {
			t.Errorf("got %+v and %q, want no week, unavailable", got, state)
		}

		if !strings.Contains(logs.String(), "peer=time err=") {
			t.Errorf("logged %q, want a line about time", logs.String())
		}
	})
}

func TestAskLogsNothingOnceTheClientHasGone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer

		asker := Asker{client: silent{}, logger: slog.New(slog.NewTextHandler(&logs, nil))}

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		if got, state := asker.Ask(ctx); got != nil || state != screen.StateUnavailable {
			t.Errorf("got %+v and %q, want no week, unavailable", got, state)
		}

		if logs.Len() != 0 {
			t.Errorf("logged %q, want nothing", logs.String())
		}
	})
}

// silent never answers: it waits until the call is given up.
type silent struct{}

func (silent) GetWeek(
	ctx context.Context, _ *timev1.GetWeekRequest,
) (*timev1.GetWeekResponse, error) {
	<-ctx.Done()

	return nil, ctx.Err()
}
