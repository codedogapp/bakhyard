package refresh_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/games/internal/refresh"
)

// A bubble's clock starts at midnight UTC on 1 January 2000.
const (
	foundA = `{"names":["A"],"searched":"2000-01-01T00:00:00Z"}`
	foundB = `{"names":["B"],"searched":"2000-01-01T01:00:00Z"}`
)

func TestNothingIsPolledBeforeARequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, fake := started(t.Context(), finding("A"))

		time.Sleep(24 * time.Hour)
		synctest.Wait()

		if calls := fake.calls(); len(calls) != 0 {
			t.Errorf("polled at %v, want no poll", calls)
		}
	})
}

func TestTheFirstRequestsWaitForTheFirstPoll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		onRequest, fake := started(t.Context(), waiting(release, "A"))

		answers := make(chan string, 2)

		for range 2 {
			go func() { answers <- get(t.Context(), onRequest) }()
		}

		synctest.Wait()

		if len(answers) != 0 {
			t.Fatal("got an answer before the first poll ended")
		}

		close(release)

		for range 2 {
			if got := <-answers; got != foundA {
				t.Errorf("got %s, want %s", got, foundA)
			}
		}

		if calls := fake.calls(); len(calls) != 1 {
			t.Errorf("polled at %v, want one poll", calls)
		}
	})
}

func TestWhatWasFoundIsFreshForAnHourAfterItsPollBegan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		onRequest, fake := started(t.Context(), slow(10*time.Minute, "A"), finding("B"))

		get(t.Context(), onRequest)
		time.Sleep(50*time.Minute - time.Second)

		if got := get(t.Context(), onRequest); got != foundA {
			t.Errorf("got %s, want %s", got, foundA)
		}

		synctest.Wait()

		if calls := fake.calls(); len(calls) != 1 {
			t.Errorf("polled at %v, want one poll within the hour", calls)
		}

		time.Sleep(time.Second)
		get(t.Context(), onRequest)
		synctest.Wait()

		if calls := fake.calls(); len(calls) != 2 {
			t.Errorf("polled at %v, want a second poll an hour after the first began", calls)
		}
	})
}

func TestAStaleRequestPollsInTheBackground(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		onRequest, fake := started(t.Context(), finding("A"), waiting(release, "B"))

		get(t.Context(), onRequest)
		time.Sleep(time.Hour)

		// The stale request and those that come while its poll runs get what was found at once,
		// and start no poll of their own.
		for range 3 {
			if got := get(t.Context(), onRequest); got != foundA {
				t.Errorf("got %s while polling, want %s", got, foundA)
			}

			time.Sleep(time.Minute)
		}

		if calls := fake.calls(); len(calls) != 2 {
			t.Errorf("polled at %v, want two polls", calls)
		}

		close(release)
		synctest.Wait()

		if got := get(t.Context(), onRequest); got != foundB {
			t.Errorf("got %s after the poll, want %s", got, foundB)
		}
	})
}

func TestRunEndsWithItsContext(t *testing.T) {
	for _, test := range []struct {
		name    string
		replies []reply
		// before is what happens before Run's context ends; ctx is the requests' context.
		before func(ctx context.Context, onRequest *refresh.OnRequest[found])
	}{
		{"before a request", nil, func(context.Context, *refresh.OnRequest[found]) {}},
		{"during the first poll", []reply{cut}, func(
			ctx context.Context, onRequest *refresh.OnRequest[found],
		) {
			go get(ctx, onRequest)
		}},
		{"between polls", []reply{finding("A")}, func(
			ctx context.Context, onRequest *refresh.OnRequest[found],
		) {
			get(ctx, onRequest)
		}},
		{"during a later poll", []reply{finding("A"), cut}, func(
			ctx context.Context, onRequest *refresh.OnRequest[found],
		) {
			get(ctx, onRequest)
			time.Sleep(time.Hour)
			get(ctx, onRequest)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				onRequest, _ := newOnRequest(test.replies...)

				ended := make(chan struct{})

				go func() {
					onRequest.Run(ctx)
					close(ended)
				}()

				test.before(t.Context(), onRequest)
				synctest.Wait()
				cancel()
				synctest.Wait()

				select {
				case <-ended:
				default:
					t.Error("Run goes on after its context ended")
				}
			})
		})
	}
}

func TestARequestAfterRunHasEndedGetsWhatWasFound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		onRequest, fake := started(ctx, finding("A"))

		get(t.Context(), onRequest)
		cancel()
		time.Sleep(time.Hour)

		for range 2 {
			if got := get(t.Context(), onRequest); got != foundA {
				t.Errorf("got %s, want %s", got, foundA)
			}
		}

		if calls := fake.calls(); len(calls) != 1 {
			t.Errorf("polled at %v, want one poll", calls)
		}
	})
}
