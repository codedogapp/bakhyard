package supervisor_test

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

func TestShutdownStopsEveryServiceAndNothingStartsAfterIt(t *testing.T) {
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "a", Command: []string{"sleep", "60"}},
		manifest.Service{Name: "b", Command: []string{"sleep", "60"}},
	)

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	err = sup.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	actions := map[string]func() error{
		"Start":    func() error { return sup.Start("a") },
		"Restart":  func() error { return sup.Restart("a") },
		"StartAll": func() error { return sup.StartAll(t.Context()) },
	}
	for action, call := range actions {
		err = call()
		if err == nil || err.Error() != "start a: the supervisor is shutting down" {
			t.Errorf("%s after Shutdown = %v, want it to fail", action, err)
		}
	}

	assertStopped(t, sup)
}

func TestNoServiceRunsAfterAShutdownThatRacesAStartAll(t *testing.T) {
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "a", Command: []string{"sleep", "60"}},
		manifest.Service{Name: "b", Command: []string{"sleep", "60"}},
		manifest.Service{Name: "c", Command: []string{"sleep", "60"}},
	)

	var (
		group   sync.WaitGroup
		started error
	)

	group.Go(func() { started = sup.StartAll(t.Context()) })

	err := sup.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	group.Wait()

	if started != nil && !strings.HasSuffix(started.Error(), ": the supervisor is shutting down") {
		t.Errorf("StartAll = %v, want nil or the shutdown", started)
	}

	assertStopped(t, sup)
}

func assertStopped(t *testing.T, sup *supervisor.Supervisor) {
	t.Helper()

	for _, status := range sup.Services() {
		if status.State != supervisor.Stopped {
			t.Errorf("%s is %s, want it stopped", status.Name, status.State)
		}
	}
}
