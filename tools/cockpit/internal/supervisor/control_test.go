package supervisor_test

import (
	"errors"
	"io"
	"io/fs"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

var sleeper = []string{"sleep", "60"}

func TestStartAndStopOneService(t *testing.T) {
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "a", Command: sleeper},
		manifest.Service{Name: "b", Command: sleeper},
	)

	err := sup.Start("b")
	if err != nil {
		t.Fatal(err)
	}

	got := sup.Services()
	if got[0] != (supervisor.Status{Name: "a", State: supervisor.Stopped}) {
		t.Errorf("got %+v, want a stopped", got[0])
	}

	if got[1].State != supervisor.Running || got[1].PID == 0 || got[1].Started.IsZero() {
		t.Errorf("got %+v, want b running, with its pid and start time", got[1])
	}

	err = sup.Stop("b")
	if err != nil {
		t.Fatal(err)
	}

	if b := statusOf(sup, "b"); b != (supervisor.Status{Name: "b", State: supervisor.Stopped}) {
		t.Errorf("got %+v, want b stopped", b)
	}
}

func TestServicesGivesEachServicesAddress(t *testing.T) {
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "a", Command: sleeper, Listen: "tcp://127.0.0.1:8081"},
		manifest.Service{Name: "b", Command: sleeper, Listen: "tcp://127.0.0.1:8082"},
	)

	err := sup.Start("b")
	if err != nil {
		t.Fatal(err)
	}

	got := sup.Services()
	if got[0].Listen != "tcp://127.0.0.1:8081" || got[1].Listen != "tcp://127.0.0.1:8082" {
		t.Errorf("got %+v, want each service's address, whether it runs or not", got)
	}
}

func TestActionsThatHaveNothingToDo(t *testing.T) {
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "a", Command: sleeper},
		manifest.Service{Name: "b", Command: sleeper},
	)

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	before := statusOf(sup, "a")

	err = errors.Join(sup.Start("a"), sup.Stop("b"), sup.Kill("b"))
	if err != nil {
		t.Fatal(err)
	}

	if a := statusOf(sup, "a"); a != before {
		t.Errorf("got %+v, want a still running as %+v", a, before)
	}

	if b := statusOf(sup, "b"); b.State != supervisor.Stopped {
		t.Errorf("got %+v, want b stopped", b)
	}
}

func TestAServiceThatExitsOnItsOwn(t *testing.T) {
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "a", Command: sleeper},
		manifest.Service{Name: "b", Command: sleeper},
	)

	err := sup.StartAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	err = syscall.Kill(statusOf(sup, "b").PID, syscall.SIGTERM)
	if err != nil {
		t.Fatal(err)
	}

	status := waitFor(sup, "b", supervisor.Exited)
	if status.Err == nil || status.Err.Error() != "signal: terminated" || status.PID != 0 {
		t.Errorf("got %+v, want b exited on SIGTERM", status)
	}

	if a := statusOf(sup, "a"); a.State != supervisor.Running {
		t.Errorf("got %+v, want a still running", a)
	}

	err = sup.Start("b")
	if err != nil {
		t.Fatal(err)
	}

	if status = statusOf(sup, "b"); status.State != supervisor.Running || status.Err != nil {
		t.Errorf("got %+v, want b running again", status)
	}
}

func TestKillSendsNoSIGTERM(t *testing.T) {
	out := make(lines, bufferedLines)
	sup := newSupervisor(t, out, manifest.Service{Name: "a", Command: stoppable})

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	out.readUntil("a", "ready")

	err = sup.Kill("a")
	if err != nil {
		t.Fatal(err)
	}

	if a := statusOf(sup, "a"); a.State != supervisor.Stopped {
		t.Errorf("got %+v, want a stopped", a)
	}

	got := out.rest()
	if !slices.Contains(linesOf("cockpit", got), "killing a") ||
		slices.Contains(linesOf("a", got), "stopped") {
		t.Errorf("got %q, want the kill, and a to get no SIGTERM", got)
	}
}

func TestRestartStartsANewProcess(t *testing.T) {
	sup := newSupervisor(t, io.Discard, manifest.Service{Name: "a", Command: sleeper})

	// A service that isn't running just starts.
	err := sup.Restart("a")
	if err != nil {
		t.Fatal(err)
	}

	first := statusOf(sup, "a")

	err = sup.Restart("a")
	if err != nil {
		t.Fatal(err)
	}

	second := statusOf(sup, "a")
	if first.State != supervisor.Running || second.State != supervisor.Running ||
		second.PID == first.PID {
		t.Errorf("got %+v, then %+v, want a running in a new process", first, second)
	}
}

func TestAStartThatFailsChangesNothing(t *testing.T) {
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "a", Command: []string{"./no-such-program"}},
	)

	err := sup.Start("a")
	if !errors.Is(err, fs.ErrNotExist) || !strings.HasPrefix(err.Error(), "start a: ") {
		t.Errorf("got %v, want a's start to fail", err)
	}

	if a := statusOf(sup, "a"); a != (supervisor.Status{Name: "a", State: supervisor.Stopped}) {
		t.Errorf("got %+v, want a stopped", a)
	}
}

func TestAnUnknownServiceIsAnError(t *testing.T) {
	sup := newSupervisor(t, io.Discard, manifest.Service{Name: "a", Command: sleeper})

	actions := map[string]func(string) error{
		"Start": sup.Start, "Stop": sup.Stop, "Kill": sup.Kill, "Restart": sup.Restart,
	}
	for name, action := range actions {
		err := action("nope")
		if err == nil || err.Error() != `no service "nope"` {
			t.Errorf("%s: got %v, want an unknown service", name, err)
		}
	}
}

// newSupervisor returns a supervisor for services that stops them all when the test ends.
func newSupervisor(
	t *testing.T, out io.Writer, services ...manifest.Service,
) *supervisor.Supervisor {
	t.Helper()

	sup := supervisor.New(services, out)

	t.Cleanup(func() {
		err := sup.StopAll()
		if err != nil {
			t.Error(err)
		}
	})

	return sup
}

func statusOf(sup *supervisor.Supervisor, name string) supervisor.Status {
	statuses := sup.Services()
	index := slices.IndexFunc(statuses, func(status supervisor.Status) bool {
		return status.Name == name
	})

	return statuses[index]
}

// waitFor returns the status of the service called name once it is in state.
func waitFor(sup *supervisor.Supervisor, name string, state supervisor.State) supervisor.Status {
	for {
		status := statusOf(sup, name)
		if status.State == state {
			return status
		}

		<-sup.Changed()
	}
}

// readUntil reads lines until the service called name writes text.
func (l lines) readUntil(name, text string) {
	for line := range l {
		if slices.Equal(linesOf(name, []string{line}), []string{text}) {
			return
		}
	}
}
