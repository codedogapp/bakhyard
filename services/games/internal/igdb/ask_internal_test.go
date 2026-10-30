package igdb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// memory stands in for IGDB in memory, so that it answers in a bubble: it answers each request
// with no games once delay has passed, and records when each request began.
type memory struct {
	delay time.Duration
	began []time.Time
}

func (m *memory) RoundTrip(*http.Request) (*http.Response, error) {
	m.began = append(m.began, time.Now())
	time.Sleep(m.delay)

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("[]")),
	}, nil
}

// hanging stands in for an IGDB that never answers.
type hanging struct{}

func (hanging) RoundTrip(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()

	return nil, r.Context().Err()
}

// aToken is a token that IGDB never refuses.
type aToken struct{}

func (aToken) Token(context.Context) (string, error) { return "a-token", nil }

func (aToken) Drop(string) {}

func TestRequestsStartAQuarterOfASecondApart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		igdb := &memory{}
		client := standIn(igdb)
		start := time.Now()

		wantNoCover(t, client)
		wantNoCover(t, client)
		time.Sleep(time.Second)
		wantNoCover(t, client)

		want := []time.Duration{0, 250, 500, 750, 1750, 2000}
		if got := since(start, igdb.began); !slices.Equal(got, want) {
			t.Errorf("the requests began after %v ms, want %v", got, want)
		}
	})
}

func TestRequestsGoOneAtATime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		igdb := &memory{delay: time.Second}
		client := standIn(igdb)
		start := time.Now()

		var calls sync.WaitGroup
		for range 2 {
			calls.Go(func() { wantNoCover(t, client) })
		}

		calls.Wait()

		want := []time.Duration{0, 1000, 2000, 3000}
		if got := since(start, igdb.began); !slices.Equal(got, want) {
			t.Errorf("the requests began after %v ms, want %v", got, want)
		}
	})
}

func TestACallThatWaitsForTheRequestUnderWayEndsWithItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		igdb := &memory{delay: time.Second}
		client := standIn(igdb)

		var first sync.WaitGroup
		first.Go(func() { wantNoCover(t, client) })
		synctest.Wait()

		wantGivenUp(t, client)
		first.Wait()

		if len(igdb.began) != 2 {
			t.Errorf("IGDB got %d requests, want the first call's 2", len(igdb.began))
		}
	})
}

func TestACallThatWaitsForItsQuarterOfASecondEndsWithItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		igdb := &memory{}
		client := standIn(igdb)

		wantNoCover(t, client)
		wantGivenUp(t, client)

		if len(igdb.began) != 2 {
			t.Errorf("IGDB got %d requests, want the first call's 2", len(igdb.began))
		}
	})
}

func TestARequestGivesUpAfter10Seconds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()

		got, err := standIn(hanging{}).Cover(t.Context(), "Moon Garden")
		if got != "" || !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("got %q, %v, want the timeout's error", got, err)
		}

		if waited := time.Since(began); waited != 10*time.Second {
			t.Errorf("gave up after %s, want 10s", waited)
		}
	})
}

// standIn returns the Client whose requests go to igdb instead of IGDB.
func standIn(igdb http.RoundTripper) *Client {
	client := New(&url.URL{Scheme: "https", Host: "api.igdb.com"}, "an-id", aToken{})
	client.client.Transport = igdb

	return client
}

// wantNoCover checks that client finds no cover for Moon Garden, and no error.
func wantNoCover(t *testing.T, client *Client) {
	t.Helper()

	if got, err := client.Cover(t.Context(), "Moon Garden"); got != "" || err != nil {
		t.Errorf("got %q, %v, want no cover", got, err)
	}
}

// wantGivenUp checks that client gives up on Moon Garden's cover with its context's error, when
// the context ends 100 ms after the call.
func wantGivenUp(t *testing.T, client *Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	began := time.Now()

	got, err := client.Cover(ctx, "Moon Garden")
	if got != "" || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %q, %v, want the context's error", got, err)
	}

	if waited := time.Since(began); waited != 100*time.Millisecond {
		t.Errorf("gave up after %s, want 100ms", waited)
	}
}

// since returns how many milliseconds after start each of times is.
func since(start time.Time, times []time.Time) []time.Duration {
	waits := make([]time.Duration, 0, len(times))
	for _, at := range times {
		waits = append(waits, at.Sub(start)/time.Millisecond)
	}

	return waits
}
