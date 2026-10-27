package poll_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

func TestPollerAnswersWhatItFoundAsJSON(t *testing.T) {
	for _, test := range []struct {
		reply reply
		want  string
	}{
		{finding("Ada", "Grace"), `{"names":["Ada","Grace"],"searched":"2000-01-01T00:00:00Z"}`},
		{finding(), `{"names":[],"searched":"2000-01-01T00:00:00Z"}`},
		{failing("look Acme up: 401 Unauthorized"), `{"failure":"look Acme up: 401 Unauthorized"}`},
	} {
		synctest.Test(t, func(t *testing.T) {
			poller, _, _ := polling(t.Context(), test.reply)

			// Result holds the header as sent; Header is the live map, which a late Set changes.
			rec := get(t.Context(), poller)
			sent := rec.Result().Header.Get("Content-Type")

			if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
				sent != "application/json" {
				t.Errorf("got %d %s (%s), want 200 %s as JSON", rec.Code, rec.Body, sent,
					test.want)
			}
		})
	}
}

func TestPollerAnswersOnceTheFirstLookUpEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		poller, _, _ := polling(t.Context(), func(ctx context.Context) (found, error) {
			time.Sleep(30 * time.Second)

			return finding("Ada")(ctx)
		})

		rec := get(t.Context(), poller)
		waited := time.Since(start)

		want := `{"names":["Ada"],"searched":"2000-01-01T00:00:00Z"}`
		if rec.Body.String() != want || waited != 30*time.Second {
			t.Errorf("got %s after %s, want %s after 30s", rec.Body, waited, want)
		}
	})
}

func TestPollerAnswersNothingToAClientThatLeft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poller, _, _ := polling(t.Context(), cut)

		gone, leave := context.WithCancel(t.Context())
		leave()

		// A recorder's code stays 200 until the handler writes a header.
		rec := get(gone, poller)
		if rec.Code != http.StatusOK || rec.Body.Len() != 0 || len(rec.Header()) != 0 {
			t.Errorf("got %d %q %v, want nothing", rec.Code, rec.Body, rec.Header())
		}
	})
}

// get returns poller's answer to a GET, from a client that leaves when ctx ends.
func get(ctx context.Context, poller *poll.Poller[found]) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	poller.ServeHTTP(rec, req)

	return rec
}
