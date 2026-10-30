package twitch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// issuer stands in for Twitch's OAuth server in memory, so that it answers in a bubble: each
// request gets the next token, issued for 100 seconds.
type issuer struct {
	issued int
}

func (i *issuer) RoundTrip(*http.Request) (*http.Response, error) {
	i.issued++
	answer := fmt.Sprintf(`{"access_token":"token %d","expires_in":100}`, i.issued)

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(answer)),
	}, nil
}

// hanging stands in for a Twitch that never answers.
type hanging struct{}

func (hanging) RoundTrip(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()

	return nil, r.Context().Err()
}

func TestATokenLivesAsLongAsTwitchSays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tokens := keep(standIn(&issuer{}).token)

		wantToken(t, tokens, "token 1")
		time.Sleep(90*time.Second - time.Nanosecond)
		wantToken(t, tokens, "token 1")
		time.Sleep(time.Nanosecond)
		wantToken(t, tokens, "token 2")
	})
}

func TestARequestForATokenGivesUpAfter10Seconds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tokens := keep(standIn(hanging{}).token)
		began := time.Now()

		wantError(t, tokens, context.DeadlineExceeded)

		if waited := time.Since(began); waited != 10*time.Second {
			t.Errorf("gave up after %s, want 10s", waited)
		}
	})
}

// standIn returns the app whose requests go to answer instead of Twitch.
func standIn(answer http.RoundTripper) *app {
	twitchApp := newApp(&url.URL{Scheme: "https", Host: "id.twitch.tv"}, "an-id", "a-secret")
	twitchApp.client.Transport = answer

	return twitchApp
}
