package supervisor_test

import (
	"context"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

func TestStartAllStartsAServiceOnceItsDependenciesAreHealthy(t *testing.T) {
	home, web := newHealthServer(t, true), newHealthServer(t, false)
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "home", Command: sleeper, Listen: home.listen},
		manifest.Service{Name: "web", Command: sleeper, Listen: web.listen},
		manifest.Service{Name: "gateway", Command: sleeper, DependsOn: []string{"home", "web"}},
	)
	result := make(chan error, 1)

	go func() {
		result <- sup.StartAll(t.Context())
	}()

	// web fails a check, then the next.
	<-web.checked
	<-web.checked

	if gateway := statusOf(sup, "gateway"); gateway.State != supervisor.Stopped {
		t.Errorf("got %+v, want gateway stopped while web isn't healthy", gateway)
	}

	web.healthy.Store(true)

	err := <-result
	if err != nil {
		t.Fatal(err)
	}

	if gateway := statusOf(sup, "gateway"); gateway.State != supervisor.Running {
		t.Errorf("got %+v, want gateway running", gateway)
	}

	if len(home.checked) == 0 {
		t.Error("got no check of home, want one")
	}
}

func TestStartAllGivesUpOnADependencyThatIsNotHealthyInTime(t *testing.T) {
	home := newHealthServer(t, false)
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "home", Command: sleeper, Listen: home.listen},
		manifest.Service{Name: "gateway", Command: sleeper, DependsOn: []string{"home"}},
	)
	sup.SetWaitTimeout(shortWait)

	err := sup.StartAll(t.Context())

	want := "start gateway: home isn't healthy within 300ms: http://" +
		strings.TrimPrefix(home.listen, "tcp://") + "/healthz answered 503 Service Unavailable"
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want %q", err, want)
	}

	assertStates(t, sup, map[string]supervisor.State{
		"home": supervisor.Running, "gateway": supervisor.Stopped,
	})
}

func TestStartAllGivesUpOnADependencyThatExited(t *testing.T) {
	tests := []struct{ name, script, want string }{
		{"with an error", "exit 3", "start gateway: home exited: exit status 3"},
		{"with status 0", "exit 0", "start gateway: home exited"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home := newHealthServer(t, false)
			sup := newSupervisor(t, io.Discard,
				manifest.Service{
					Name: "home", Command: []string{"sh", "-c", test.script}, Listen: home.listen,
				},
				manifest.Service{Name: "gateway", Command: sleeper, DependsOn: []string{"home"}},
			)

			err := sup.StartAll(t.Context())
			if err == nil || err.Error() != test.want {
				t.Errorf("got %v, want %q", err, test.want)
			}

			assertStates(t, sup, map[string]supervisor.State{"gateway": supervisor.Stopped})
		})
	}
}

func TestStartAllGivesUpOnADependencyThatStops(t *testing.T) {
	home := newHealthServer(t, false)
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "home", Command: sleeper, Listen: home.listen},
		manifest.Service{Name: "gateway", Command: sleeper, DependsOn: []string{"home"}},
	)
	result := make(chan error, 1)

	go func() {
		result <- sup.StartAll(t.Context())
	}()

	<-home.checked

	err := sup.Stop("home")
	if err != nil {
		t.Fatal(err)
	}

	err = <-result
	if err == nil || err.Error() != "start gateway: home isn't running" {
		t.Errorf("got %v, want home not running", err)
	}

	assertStates(t, sup, map[string]supervisor.State{"gateway": supervisor.Stopped})
}

func TestStartAllLeavesARunningServiceAlone(t *testing.T) {
	home := newHealthServer(t, false)
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "home", Command: sleeper, Listen: home.listen},
		manifest.Service{Name: "gateway", Command: sleeper, DependsOn: []string{"home"}},
	)

	for _, name := range []string{"home", "gateway"} {
		err := sup.Start(name)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := sup.StartAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(home.checked) > 0 {
		t.Error("got a check of home, want none")
	}
}

func TestStartAndRestartDontWaitForDependencies(t *testing.T) {
	// home never runs, so a wait for it would fail.
	sup := newSupervisor(t, io.Discard,
		manifest.Service{Name: "home", Command: sleeper},
		manifest.Service{Name: "gateway", Command: sleeper, DependsOn: []string{"home"}},
	)

	for name, action := range map[string]func(string) error{
		"Start": sup.Start, "Restart": sup.Restart,
	} {
		err := action("gateway")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		assertStates(t, sup, map[string]supervisor.State{
			"home": supervisor.Stopped, "gateway": supervisor.Running,
		})
	}
}

func TestRunEndsWithoutAnErrorWhenItsContextEndsDuringAWait(t *testing.T) {
	home := newHealthServer(t, false)
	services := []manifest.Service{
		{Name: "home", Command: sleeper, Listen: home.listen},
		{Name: "gateway", Command: sleeper, DependsOn: []string{"home"}},
	}
	ctx, cancel := context.WithCancel(t.Context())
	out := make(lines, bufferedLines)
	result := make(chan error, 1)

	go func() {
		result <- supervisor.Run(ctx, services, out)
	}()

	<-home.checked
	cancel()

	err := <-result
	if err != nil {
		t.Fatal(err)
	}

	got := withoutPIDs(out.rest())
	want := []string{"cockpit | started home (pid N)\n", "cockpit | stopping home\n"}

	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunStopsWhatStartedWhenADependencyExits(t *testing.T) {
	home := newHealthServer(t, false)
	services := []manifest.Service{
		{Name: "a", Command: sleeper},
		{Name: "home", Command: []string{"sh", "-c", "exit 3"}, Listen: home.listen},
		{Name: "gateway", Command: sleeper, DependsOn: []string{"home"}},
	}
	out := make(lines, bufferedLines)

	err := supervisor.Run(t.Context(), services, out)
	if err == nil || err.Error() != "start gateway: home exited: exit status 3" {
		t.Errorf("got %v, want home's exit", err)
	}

	got := withoutPIDs(out.rest())
	want := []string{
		"cockpit | started a (pid N)\n",
		"cockpit | started home (pid N)\n",
		"cockpit | stopping a\n",
	}

	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// shortWait is the wait timeout of a test that waits for it to pass.
const shortWait = 300 * time.Millisecond

func assertStates(t *testing.T, sup *supervisor.Supervisor, want map[string]supervisor.State) {
	t.Helper()

	for name, state := range want {
		if got := statusOf(sup, name); got.State != state {
			t.Errorf("got %+v, want %s %s", got, name, state)
		}
	}
}
