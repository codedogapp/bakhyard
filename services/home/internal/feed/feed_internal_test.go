package feed

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
)

// These tests run in a bubble, whose fake clock moves only once every goroutine in it waits, so
// they can tell how long Items waits. Real connections don't wait that way, hence the fake
// clients.

func TestItemsWaitsForTheSlowestPeerOnlyUntilTheTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer

		feed := &Feed{
			peers: []peer{
				{name: "todos", client: silent{}},
				{name: "pulls", client: silent{}},
				{name: "notes", client: listing{attentionv1.Item_builder{Id: new("a")}.Build()}},
			},
			logger: slog.New(slog.NewTextHandler(&logs, nil)),
		}

		start := time.Now()
		got := feed.Items(t.Context())

		if waited, want := time.Since(start), 1500*time.Millisecond; waited != want {
			t.Errorf("Items waited %v, want %v: the peers are asked at once", waited, want)
		}

		if len(got) != 1 || got[0].ID != "notes/a" {
			t.Errorf("got %+v, want notes/a alone", got)
		}

		for _, name := range []string{"todos", "pulls"} {
			if !strings.Contains(logs.String(), "peer="+name+" err=") {
				t.Errorf("logged %q, want a line about %s", logs.String(), name)
			}
		}
	})
}

func TestItemsLogsNothingOnceTheClientHasGone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer

		feed := &Feed{
			peers:  []peer{{name: "todos", client: silent{}}},
			logger: slog.New(slog.NewTextHandler(&logs, nil)),
		}

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		if got := feed.Items(ctx); len(got) != 0 {
			t.Errorf("got %+v, want none", got)
		}

		if logs.Len() != 0 {
			t.Errorf("logged %q, want nothing", logs.String())
		}
	})
}

// silent never answers: it waits until the call is given up.
type silent struct{}

func (silent) ListItems(
	ctx context.Context, _ *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	<-ctx.Done()

	return nil, ctx.Err()
}

// listing answers with its items at once.
type listing []*attentionv1.Item

func (l listing) ListItems(
	context.Context, *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	return attentionv1.ListItemsResponse_builder{Items: l}.Build(), nil
}
