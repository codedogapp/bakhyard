package twitch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// answer is how the fake Twitch answers a request for a token, after took: with token and its
// life, or with err when it isn't nil.
type answer struct {
	token string
	life  time.Duration
	took  time.Duration
	err   error
}

// fakeTwitch answers the requests for a token in turn, as its answers say.
type fakeTwitch struct {
	answers []answer
	asked   int
}

func (f *fakeTwitch) get(context.Context) (string, time.Duration, error) {
	if f.asked == len(f.answers) {
		return "", 0, errors.New("one request too many")
	}

	got := f.answers[f.asked]
	f.asked++

	time.Sleep(got.took)

	return got.token, got.life, got.err
}

func TestTokensKeepATokenFor9TenthsOfItsLife(t *testing.T) {
	for _, took := range []time.Duration{0, 10 * time.Second} {
		t.Run(took.String(), func(t *testing.T) {
			// A bubble's clock starts at midnight UTC on 1 January 2000.
			synctest.Test(t, func(t *testing.T) {
				twitch := &fakeTwitch{answers: []answer{
					{token: "first", life: 100 * time.Second, took: took},
					{token: "second", life: 100 * time.Second},
				}}
				tokens := keep(twitch.get)
				began := time.Now()

				wantToken(t, tokens, "first")
				time.Sleep(time.Until(began.Add(90*time.Second - time.Nanosecond)))
				wantToken(t, tokens, "first")
				time.Sleep(time.Nanosecond)
				wantToken(t, tokens, "second")
				wantToken(t, tokens, "second")

				if twitch.asked != 2 {
					t.Errorf("Twitch was asked %d times, want 2", twitch.asked)
				}
			})
		})
	}
}

func TestTokensAskTwitchOnceAtATime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		twitch := &fakeTwitch{answers: []answer{
			{token: "first", life: time.Hour, took: time.Second},
		}}
		tokens := keep(twitch.get)

		var calls sync.WaitGroup
		for range 3 {
			calls.Go(func() { wantToken(t, tokens, "first") })
		}

		calls.Wait()

		if twitch.asked != 1 {
			t.Errorf("Twitch was asked %d times, want once", twitch.asked)
		}
	})
}

func TestATokenThatWaitsEndsWithItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		twitch := &fakeTwitch{answers: []answer{
			{token: "first", life: time.Hour, took: time.Minute},
		}}
		tokens := keep(twitch.get)

		var first sync.WaitGroup
		first.Go(func() { wantToken(t, tokens, "first") })

		synctest.Wait()

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		began := time.Now()

		token, err := tokens.Token(ctx)
		if token != "" || !errors.Is(err, context.DeadlineExceeded) ||
			time.Since(began) != time.Second {
			t.Errorf("got %q, %v after %s, want the context's error after 1s", token, err,
				time.Since(began))
		}

		first.Wait()
	})
}

func TestAFailedRequestForATokenChangesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		down := errors.New("Twitch is down")
		twitch := &fakeTwitch{answers: []answer{
			{err: down},
			{token: "first", life: 100 * time.Second},
			{err: down},
			{token: "second", life: 100 * time.Second},
		}}
		tokens := keep(twitch.get)

		wantError(t, tokens, down)
		wantToken(t, tokens, "first")
		time.Sleep(90 * time.Second)
		wantError(t, tokens, down)
		wantToken(t, tokens, "second")
	})
}

func TestDropForgetsTheTokenThatIGDBRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		twitch := &fakeTwitch{answers: []answer{
			{token: "first", life: time.Hour},
			{token: "second", life: time.Hour},
		}}
		tokens := keep(twitch.get)

		wantToken(t, tokens, "first")
		tokens.Drop("first")
		wantToken(t, tokens, "second")
		tokens.Drop("first")
		wantToken(t, tokens, "second")
	})
}

func TestDropWaitsForTheTokenThatIsComing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		twitch := &fakeTwitch{answers: []answer{
			{token: "first", life: 100 * time.Second},
			{token: "second", life: 100 * time.Second, took: time.Second},
		}}
		tokens := keep(twitch.get)

		wantToken(t, tokens, "first")
		time.Sleep(90 * time.Second)

		var renewal sync.WaitGroup
		renewal.Go(func() { wantToken(t, tokens, "second") })

		synctest.Wait()
		tokens.Drop("first")
		renewal.Wait()
		wantToken(t, tokens, "second")
	})
}

// wantToken checks that tokens' Token returns want.
func wantToken(t *testing.T, tokens *Tokens, want string) {
	t.Helper()

	got, err := tokens.Token(t.Context())
	if got != want || err != nil {
		t.Errorf("at %s: got %q, %v, want %q", time.Now().UTC(), got, err, want)
	}
}

// wantError checks that tokens' Token fails with want.
func wantError(t *testing.T, tokens *Tokens, want error) {
	t.Helper()

	got, err := tokens.Token(t.Context())
	if got != "" || !errors.Is(err, want) {
		t.Errorf("at %s: got %q, %v, want the error %v", time.Now().UTC(), got, err, want)
	}
}
