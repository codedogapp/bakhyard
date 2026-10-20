package supervisor_test

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

// stubborn ignores SIGTERM, and so does the sleep it becomes.
var stubborn = []string{"sh", "-c", `trap "" TERM; echo ready; exec sleep 60`}

func TestKillEndsAStopThatHangs(t *testing.T) {
	out := make(lines, bufferedLines)
	sup := newSupervisor(t, out, manifest.Service{Name: "a", Command: stubborn})

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	out.readUntil("a", "ready")

	began := time.Now()
	stopped := make(chan error, 1)

	go func() {
		stopped <- sup.Stop("a")
	}()

	waitFor(sup, "a", supervisor.Stopping)

	err = sup.Start("a")
	if err == nil || err.Error() != "start a: it is still stopping" {
		t.Errorf("got %v, want no start while a stops", err)
	}

	err = errors.Join(sup.Kill("a"), <-stopped)
	if err != nil {
		t.Fatal(err)
	}

	// The stop alone would have waited 10 s before SIGKILL.
	if took := time.Since(began); took > 5*time.Second {
		t.Errorf("the stop took %v, want the kill to end it at once", took)
	}

	if a := statusOf(sup, "a"); a.State != supervisor.Stopped {
		t.Errorf("got %+v, want a stopped", a)
	}
}

func TestStopsAtTheSameTimeAllReturn(t *testing.T) {
	sup := newSupervisor(t, io.Discard, manifest.Service{Name: "a", Command: sleeper})

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	var group sync.WaitGroup

	errs := make(chan error, 3)
	for range cap(errs) {
		group.Go(func() { errs <- sup.Stop("a") })
	}

	group.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}

	if a := statusOf(sup, "a"); a.State != supervisor.Stopped {
		t.Errorf("got %+v, want a stopped", a)
	}
}
