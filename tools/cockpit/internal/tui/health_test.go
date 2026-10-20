package tui

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/probe"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/resources"
)

func TestCheckingFillsTheHealthColumn(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a", "b")...)
	screen := newModel(sup, time.Now())

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	pid := sup.Services()[0].PID

	screen, next := update(t, screen, checkedMsg{pid: nil})
	if rows := screen.grid.Rows(); rows[0][2] != "up" || rows[1][2] != "" || next == nil {
		t.Errorf("got %q, want a up, nothing for b, and to check again", rows)
	}

	screen, _ = update(t, screen, checkedMsg{pid: errors.New("connection refused")})
	if row := screen.grid.Rows()[0]; row[2] != "down" {
		t.Errorf("got %q, want a down", row)
	}
}

func TestARestartedServiceHasNoHealthUntilItIsChecked(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a")...)
	screen := newModel(sup, time.Now())

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	screen, _ = update(t, screen, checkedMsg{sup.Services()[0].PID: nil})

	err = sup.Restart("a")
	if err != nil {
		t.Fatal(err)
	}

	screen, _ = update(t, screen, changedMsg{})
	if row := screen.grid.Rows()[0]; row[1] != "running" || row[2] != "" {
		t.Errorf("got %q, want a running, with no health", row)
	}
}

func TestWhyTheSelectedServiceIsDownComesAfterEveryOtherMessage(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a", "b")...)
	screen := newModel(sup, time.Now())

	err := sup.StartAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	statuses := sup.Services()
	checked := checkedMsg{statuses[0].PID: nil, statuses[1].PID: errors.New("connection refused")}

	screen, _ = update(t, screen, checked)
	if got := screen.status(); got != "" {
		t.Errorf("got %q with a selected and up, want nothing", got)
	}

	screen = act(t, screen, press('j'))
	if got := screen.status(); got != "b is down: connection refused" {
		t.Errorf("got %q with b selected, want why b is down", got)
	}

	screen, _ = update(t, screen, measuredMsg{err: errors.New("measure resources: no ps")})
	if got := screen.status(); got != "measure resources: no ps" {
		t.Errorf("got %q, want the measuring error first", got)
	}

	screen, _ = update(t, screen, measuredMsg{usage: map[int]resources.Usage{}})
	if !strings.Contains(screen.View().Content, "b is down: connection refused") {
		t.Errorf("the view doesn't show why b is down:\n%s", screen.View().Content)
	}
}

func TestWithNoServicesNothingIsDown(t *testing.T) {
	screen := newModel(newSupervisor(t, io.Discard), time.Now())

	if got := screen.status(); got != "" {
		t.Errorf("got %q, want nothing", got)
	}
}

func TestCheckChecksEveryServiceWithAProcess(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(healthy.Close)

	up := "tcp://" + healthy.Listener.Addr().String()
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "a", Command: []string{"sleep", "60"}, Listen: up},
		manifest.Service{Name: "b", Command: []string{"sleep", "60"}, Listen: closedPort(t)},
		manifest.Service{Name: "c", Command: []string{"sleep", "60"}, Listen: up},
	)

	for _, name := range []string{"a", "b"} {
		err := sup.Start(name)
		if err != nil {
			t.Fatal(err)
		}
	}

	statuses := sup.Services()
	checked := check(sup, probe.New())

	errA, okA := checked[statuses[0].PID]
	errB := checked[statuses[1].PID]

	if len(checked) != 2 || !okA || errA != nil || errB == nil {
		t.Errorf("got %v, want a up and b down, and nothing for c", checked)
	}
}

// closedPort returns a listen address that nothing listens on.
func closedPort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	listen := "tcp://" + listener.Addr().String()

	err = listener.Close()
	if err != nil {
		t.Fatal(err)
	}

	return listen
}
