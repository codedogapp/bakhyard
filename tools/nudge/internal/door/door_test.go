package door_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/door"
)

const questionID = "11111111-1111-4111-8111-111111111111"

// gateway is a fake gateway over TLS that records each request as "METHOD path body".
type gateway struct {
	server *httptest.Server
	mu     sync.Mutex
	seen   []string
}

func newGateway(t *testing.T, handle http.HandlerFunc) *gateway {
	t.Helper()

	srv := &gateway{}
	srv.server = httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)

			srv.mu.Lock()
			srv.seen = append(srv.seen, r.Method+" "+r.URL.Path+" "+string(body))
			srv.mu.Unlock()

			handle(w, r)
		}),
	)
	t.Cleanup(srv.server.Close)

	return srv
}

func (srv *gateway) requests() []string {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	return append([]string(nil), srv.seen...)
}

func (srv *gateway) client(t *testing.T) *door.Client {
	t.Helper()

	client, err := door.New(srv.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client.HTTP = srv.server.Client()
	client.Backoff = time.Microsecond

	return client
}

var question = door.Question{
	ID:      questionID,
	Project: "atlas",
	Text:    "Merge?",
	Options: []string{"Yes"},
}

const askBody = `{"id":"` + questionID + `","project":"atlas","text":"Merge?","details":"","options":["Yes"]}`

func TestNewWantsAnHTTPSURL(t *testing.T) {
	for _, server := range []string{"", "http://x.example", "x.example", "https://", "https://%zz"} {
		if _, err := door.New(server); err == nil {
			t.Errorf("New(%q) accepted it", server)
		}
	}
}

func TestAskReturnsTheAnswer(t *testing.T) {
	srv := newGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"option":"Yes","text":"go"}`))
	})

	got, err := srv.client(t).Ask(t.Context(), question)
	if err != nil || got != (door.Answer{Option: "Yes", Text: "go"}) {
		t.Errorf("got %+v, %v", got, err)
	}

	if want := []string{
		"POST /agent/questions " + askBody,
	}; strings.Join(
		srv.requests(),
		"|",
	) != want[0] {
		t.Errorf("requests %q, want %q", srv.requests(), want)
	}
}

func TestAskAsksAgainWithTheSameIDAfterABrokenConnectionOrAnUnavailableService(t *testing.T) {
	var calls int

	srv := newGateway(t, func(w http.ResponseWriter, r *http.Request) {
		calls++

		switch calls {
		case 1:
			http.Error(w, "nudge is stopping", http.StatusServiceUnavailable)
		case 2:
			conn, _, _ := w.(http.Hijacker).Hijack()
			_ = conn.Close()
		default:
			_, _ = w.Write([]byte(`{"option":"","text":"go"}`))
		}
	})

	got, err := srv.client(t).Ask(t.Context(), question)
	if err != nil || got.Text != "go" {
		t.Fatalf("got %+v, %v", got, err)
	}

	for _, request := range srv.requests() {
		if request != "POST /agent/questions "+askBody {
			t.Errorf("request %q, want the same question each time", request)
		}
	}

	if len(srv.requests()) != 3 {
		t.Errorf("%d requests, want 3", len(srv.requests()))
	}
}

func TestAskFailsWhenTheServiceRefusesOrTheQuestionIsGone(t *testing.T) {
	tests := []struct {
		status int
		reason string
		want   string
	}{
		{http.StatusBadRequest, "the question has no text", "the question has no text"},
		{http.StatusForbidden, "", "403 Forbidden"},
		{http.StatusGone, "gone", door.ErrGone.Error()},
	}

	for _, test := range tests {
		srv := newGateway(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, test.reason, test.status)
		})

		_, err := srv.client(t).Ask(t.Context(), question)
		if err == nil || err.Error() != test.want {
			t.Errorf("status %d: got %v, want %q", test.status, err, test.want)
		}

		if len(srv.requests()) != 1 {
			t.Errorf("status %d: %d requests, want 1", test.status, len(srv.requests()))
		}
	}
}

func TestAskGivesUpOnAServiceThatStaysUnreachable(t *testing.T) {
	srv := newGateway(t, func(http.ResponseWriter, *http.Request) {})
	client := srv.client(t)
	srv.server.Close()

	_, err := client.Ask(t.Context(), question)
	if err == nil || !strings.Contains(err.Error(), "giving up after 8 failed attempts") {
		t.Errorf("got %v", err)
	}
}

func TestAskWithdrawsTheQuestionWhenTheAgentLeaves(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	held := make(chan struct{})

	srv := newGateway(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			held <- struct{}{}

			<-r.Context().Done()
		}
	})

	done := make(chan error, 1)

	go func() {
		_, err := srv.client(t).Ask(ctx, question)
		done <- err
	}()

	<-held
	cancel()

	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want a cancellation", err)
	}

	requests := srv.requests()
	if last := requests[len(requests)-1]; last != "DELETE /agent/questions/"+questionID+" " {
		t.Errorf("last request %q, want the withdrawal", last)
	}
}
