package refresh_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"time"

	"github.com/codedogapp/bakhyard/services/games/internal/refresh"
)

// found is what the tests' look-ups find.
type found struct {
	Names []string `json:"names,omitzero"`
}

// reply is how a call of a fake look-up replies.
type reply func(ctx context.Context) (found, error)

// fakeLookUp is a look-up whose calls reply in turn as its replies say, and which notes when
// each call began.
type fakeLookUp struct {
	replies []reply
	mu      sync.Mutex
	began   []time.Time
}

func (f *fakeLookUp) look(ctx context.Context) (found, error) {
	f.mu.Lock()
	call := len(f.began)
	f.began = append(f.began, time.Now())
	f.mu.Unlock()

	if call >= len(f.replies) {
		return found{}, errors.New("one look-up too many")
	}

	return f.replies[call](ctx)
}

// calls returns when each call of f began.
func (f *fakeLookUp) calls() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.began)
}

// finding replies with names.
func finding(names ...string) reply {
	return func(context.Context) (found, error) { return found{Names: names}, nil }
}

func failing(message string) reply {
	return func(context.Context) (found, error) { return found{}, errors.New(message) }
}

// slow replies with names after a while.
func slow(took time.Duration, names ...string) reply {
	return func(context.Context) (found, error) {
		time.Sleep(took)

		return found{Names: names}, nil
	}
}

// waiting replies with names once release is closed.
func waiting(release <-chan struct{}, names ...string) reply {
	return func(context.Context) (found, error) {
		<-release

		return found{Names: names}, nil
	}
}

// cut replies once ctx ends, as a look-up does when the service stops.
func cut(ctx context.Context) (found, error) {
	<-ctx.Done()

	return found{}, ctx.Err()
}

// newOnRequest returns an OnRequest of Acme whose look-up replies as the replies say, and its
// fake look-up.
func newOnRequest(replies ...reply) (*refresh.OnRequest[found], *fakeLookUp) {
	fake := &fakeLookUp{replies: replies}

	return refresh.New("Acme", fake.look, slog.New(slog.DiscardHandler)), fake
}

// started returns an OnRequest as newOnRequest does, which runs until ctx ends.
func started(ctx context.Context, replies ...reply) (*refresh.OnRequest[found], *fakeLookUp) {
	onRequest, fake := newOnRequest(replies...)

	go onRequest.Run(ctx)

	return onRequest, fake
}

// get returns the answer of onRequest to a request with ctx.
func get(ctx context.Context, onRequest *refresh.OnRequest[found]) string {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/games/recent", nil)
	recorder := httptest.NewRecorder()
	onRequest.ServeHTTP(recorder, req)

	return recorder.Body.String()
}
