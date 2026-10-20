package tui

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/resources"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

func TestMeasuringFillsTheColumns(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a")...)
	screen := newModel(sup, time.Now())

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	usage := map[int]resources.Usage{sup.Services()[0].PID: {CPU: 1.5, Memory: 3 * mebibyte}}

	screen, next := update(t, screen, measuredMsg{usage: usage})
	if row := screen.grid.Rows()[0]; row[5] != "1.5%" || row[6] != "3.0 MiB" || next == nil {
		t.Errorf("got %q, want a's usage, and to measure again", row)
	}
}

func TestAFailedMeasuringShowsUntilOneWorks(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a")...)
	screen := newModel(sup, time.Now())

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	usage := map[int]resources.Usage{sup.Services()[0].PID: {CPU: 1.5, Memory: 3 * mebibyte}}
	screen, _ = update(t, screen, measuredMsg{usage: usage})

	screen, next := update(t, screen, measuredMsg{err: errors.New("measure resources: no ps")})
	if row := screen.grid.Rows()[0]; row[5] != "" || row[6] != "" || next == nil {
		t.Errorf("got %q, want no usage, and to measure again", row)
	}

	if got := screen.status(); got != "measure resources: no ps" {
		t.Errorf("got %q, want why measuring failed", got)
	}

	screen, _ = update(t, screen, measuredMsg{usage: usage})
	if got := screen.status(); got != "" {
		t.Errorf("got %q once measuring worked, want nothing", got)
	}
}

func TestTheKillQuestionAndAnActionsErrorComeBeforeAMeasuringError(t *testing.T) {
	bad := manifest.Service{Name: "bad", Command: []string{"/nonexistent"}}
	sup := newSupervisor(t, io.Discard, bad)
	unmeasured := measuredMsg{err: errors.New("measure resources: no ps")}
	screen, _ := update(t, newModel(sup, time.Now()), unmeasured)

	screen = act(t, screen, press('K'))
	if got := screen.status(); got != "Kill bad? y/n" {
		t.Errorf("got %q, want the question", got)
	}

	screen = act(t, act(t, screen, press('n')), press('s'))
	if got := screen.status(); !strings.HasPrefix(got, "start bad: ") {
		t.Errorf("got %q, want why bad didn't start", got)
	}

	screen = act(t, screen, press('x'))
	if got := screen.status(); got != unmeasured.err.Error() {
		t.Errorf("got %q once the action's error went, want the measuring error", got)
	}
}

func TestMeasureMeasuresTheServicesWithAProcess(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a", "b")...)

	err := sup.Start("b")
	if err != nil {
		t.Fatal(err)
	}

	measured := measure(sup)

	use, ok := measured.usage[sup.Services()[1].PID]
	if measured.err != nil || len(measured.usage) != 1 || !ok || use.Memory <= 0 {
		t.Errorf("got %+v, want b's usage alone", measured)
	}
}

func TestProcessIDsLeavesOutTheServicesWithNoProcess(t *testing.T) {
	statuses := []supervisor.Status{
		{Name: "a", State: supervisor.Stopped},
		{Name: "b", State: supervisor.Running, PID: 41},
		{Name: "c", State: supervisor.Exited},
		{Name: "d", State: supervisor.Stopping, PID: 7},
	}

	if got := processIDs(statuses); !slices.Equal(got, []int{41, 7}) {
		t.Errorf("got %v, want 41 and 7", got)
	}
}
