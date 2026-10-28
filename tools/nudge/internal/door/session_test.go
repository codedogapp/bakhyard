package door_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/door"
)

// sessionGateway answers a start with a session's ID and holds the request open, and answers a
// report with status.
func sessionGateway(t *testing.T, reportStatus *int) *gateway {
	t.Helper()

	return newGateway(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"id":"abc"}`))
			w.(http.Flusher).Flush()
			<-r.Context().Done()

			return
		}

		w.WriteHeader(*reportStatus)
	})
}

func TestASessionReportsItsNote(t *testing.T) {
	status := http.StatusNoContent
	srv := sessionGateway(t, &status)
	client := srv.client(t)

	session, err := client.Register(t.Context(), "atlas")
	if err != nil {
		t.Fatal(err)
	}

	if err := session.Report(t.Context(), "reading"); err != nil {
		t.Errorf("report: %v", err)
	}

	want := `POST /agent/sessions {"project":"atlas"}|PUT /agent/sessions/abc/note {"note":"reading"}`
	if got := strings.Join(srv.requests(), "|"); got != want {
		t.Errorf("requests %q, want %q", got, want)
	}

	status = http.StatusNotFound

	if err := session.Report(t.Context(), "x"); !errors.Is(err, door.ErrSessionGone) {
		t.Errorf("a forgotten session: got %v", err)
	}

	status = http.StatusBadRequest

	if err := session.Report(t.Context(), "x"); err == nil || errors.Is(err, door.ErrSessionGone) {
		t.Errorf("a refused note: got %v", err)
	}
}

func TestRegisterFailsWhenTheServiceRefusesOrIsGone(t *testing.T) {
	srv := newGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusForbidden)
	})
	client := srv.client(t)

	if _, err := client.Register(t.Context(), "atlas"); err == nil || err.Error() != "no" {
		t.Errorf("a refusal: got %v", err)
	}

	srv.server.Close()

	if _, err := client.Register(t.Context(), "atlas"); err == nil {
		t.Error("registered with a gateway that is gone")
	}
}

func TestRegisterFailsOnAReplyThatIsNotASession(t *testing.T) {
	srv := newGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`nope`))
	})

	if _, err := srv.client(t).Register(t.Context(), "atlas"); err == nil {
		t.Error("registered on garbage")
	}
}

func TestTheReporterStartsAgainWhenTheServiceForgotTheSession(t *testing.T) {
	status := http.StatusNoContent
	srv := sessionGateway(t, &status)
	reporter := door.NewReporter(srv.client(t), "atlas")
	t.Cleanup(reporter.Close)

	if err := reporter.Register(); err != nil {
		t.Fatal(err)
	}

	if err := reporter.Report(t.Context(), "one"); err != nil {
		t.Fatal(err)
	}

	status = http.StatusNotFound

	if err := reporter.Report(t.Context(), "two"); !errors.Is(err, door.ErrSessionGone) {
		// The second session's note is refused as gone too, so the error says so.
		t.Errorf("got %v", err)
	}

	starts := 0

	for _, request := range srv.requests() {
		if strings.HasPrefix(request, "POST /agent/sessions") {
			starts++
		}
	}

	if starts != 2 {
		t.Errorf("%d sessions started, want 2", starts)
	}
}

func TestAClosedReporterStartsNoSession(t *testing.T) {
	status := http.StatusNoContent
	srv := sessionGateway(t, &status)
	reporter := door.NewReporter(srv.client(t), "atlas")

	reporter.Close()
	reporter.Close()

	if err := reporter.Report(t.Context(), "x"); !errors.Is(err, door.ErrClosed) {
		t.Errorf("reported after Close: got %v", err)
	}

	if got := srv.requests(); len(got) != 0 {
		t.Errorf("requests %q, want none", got)
	}
}

func TestClosingTheReporterEndsItsSession(t *testing.T) {
	ended := make(chan struct{})
	srv := newGateway(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"abc"}`))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(ended)
	})
	reporter := door.NewReporter(srv.client(t), "atlas")

	if err := reporter.Register(); err != nil {
		t.Fatal(err)
	}

	reporter.Close()
	<-ended
}
