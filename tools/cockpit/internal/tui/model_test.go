package tui

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

func TestKeysControlTheSelectedService(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a", "b")...)
	screen := newModel(sup, time.Now())

	for _, step := range []struct {
		key  rune
		want string // the states of a and b
	}{
		{'s', "running stopped"},
		{'j', "running stopped"}, // selects b
		{'a', "running running"},
		{'x', "running stopped"},
		{'s', "running running"},
		{'Z', "stopped stopped"},
	} {
		screen = act(t, screen, press(step.key))

		if got := states(sup); got != step.want {
			t.Fatalf("after %c: got %q, want %q", step.key, got, step.want)
		}
	}
}

func TestRestartRestartsTheSelectedService(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a", "b")...)
	screen := newModel(sup, time.Now())

	err := sup.StartAll()
	if err != nil {
		t.Fatal(err)
	}

	before := sup.Services()

	act(t, act(t, screen, press('j')), press('r'))

	after := sup.Services()
	if after[0] != before[0] {
		t.Errorf("got a %+v, want it untouched", after[0])
	}

	if after[1].State != supervisor.Running || !after[1].Started.After(before[1].Started) {
		t.Errorf("got b %+v, want it running again since %v", after[1], before[1].Started)
	}
}

func TestKillAsksFirst(t *testing.T) {
	var out bytes.Buffer

	sup := newSupervisor(t, &out, sleeping("a")...)
	screen := act(t, newModel(sup, time.Now()), press('s'))

	screen = act(t, screen, press('K'))
	if got := screen.status(); got != "Kill a? y/n" {
		t.Errorf("got %q, want the question", got)
	}

	// Any key but y is a no, so x drops the kill and doesn't stop a either.
	screen = act(t, screen, press('x'))
	if screen.status() != "" || states(sup) != "running" {
		t.Errorf("got %q and a %s, want no question and a running", screen.status(), states(sup))
	}

	screen = act(t, act(t, screen, press('K')), press('y'))
	if screen.status() != "" || states(sup) != "stopped" {
		t.Errorf("got %q and a %s, want no question and a stopped", screen.status(), states(sup))
	}

	if !strings.Contains(out.String(), "killing a") || strings.Contains(out.String(), "stopping") {
		t.Errorf("got\n%s\nwant a killed, not stopped", out.String())
	}
}

func TestAnActionsErrorShowsUntilTheNextActionEnds(t *testing.T) {
	bad := manifest.Service{Name: "bad", Command: []string{"/nonexistent"}}
	sup := newSupervisor(t, io.Discard, bad)
	screen := act(t, newModel(sup, time.Now()), press('s'))

	if got := screen.status(); !strings.HasPrefix(got, "start bad: ") {
		t.Errorf("got %q, want why bad didn't start", got)
	}

	if !strings.Contains(screen.View().Content, screen.status()) {
		t.Errorf("the view doesn't show the error:\n%s", screen.View().Content)
	}

	screen = act(t, screen, press('x'))
	if got := screen.status(); got != "" {
		t.Errorf("got %q after an action that worked, want nothing", got)
	}
}

func TestChangesAndTicksRefreshTheTable(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a")...)
	screen := newModel(sup, time.Now())

	err := sup.Start("a")
	if err != nil {
		t.Fatal(err)
	}

	screen, next := update(t, screen, changedMsg{})
	if row := screen.grid.Rows()[0]; row[1] != "running" || next == nil {
		t.Errorf("got %q, want a running, and to wait for the next change", row)
	}

	upFor := 90 * time.Second
	screen, next = update(t, screen, tickMsg(sup.Services()[0].Started.Add(upFor)))

	if row := screen.grid.Rows()[0]; row[4] != "1m30s" || next == nil {
		t.Errorf("got %q, want a up for 1m30s, and to wait for the next tick", row)
	}
}

func TestWithNoServicesTheKeysDoNothingHarmful(t *testing.T) {
	sup := newSupervisor(t, io.Discard)
	screen := act(t, newModel(sup, time.Now()), press('s'))

	if got := screen.status(); got != `no service ""` {
		t.Errorf("got %q, want that there is no service", got)
	}
}

// press returns the message for pressing key.
func press(key rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: key, Text: string(key)}
}

// update gives msg to screen, and returns the screen and the command it gives back.
func update(t *testing.T, screen model, msg tea.Msg) (model, tea.Cmd) {
	t.Helper()

	next, cmd := screen.Update(msg)

	updated, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned a %T, want a model", next)
	}

	return updated, cmd
}

// act gives key to screen, then runs the action the key starts, if any, and reports how it went.
func act(t *testing.T, screen model, key tea.KeyPressMsg) model {
	t.Helper()

	screen, cmd := update(t, screen, key)
	if cmd != nil {
		screen, _ = update(t, screen, cmd())
	}

	return screen
}
