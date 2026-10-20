package tui

import (
	"errors"
	"io"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

const stopping = "Stopping every service…"

func TestQuitStopsEveryServiceThenExits(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a", "b")...)

	err := sup.StartAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	screen, shutDown := update(t, newModel(sup, time.Now()), press('q'))
	if shutDown == nil || screen.status() != stopping {
		t.Fatalf("got %q, want every service stopping", screen.status())
	}

	screen, quit := update(t, screen, shutDown())
	if got := states(sup); got != "stopped stopped" {
		t.Errorf("got %q, want every service stopped", got)
	}

	if quit == nil {
		t.Fatal("the TUI doesn't quit")
	}

	if _, ok := quit().(tea.QuitMsg); !ok || screen.err != nil {
		t.Errorf("got %v, want the TUI to quit with no error", screen.err)
	}
}

func TestCtrlCQuitsToo(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a")...)
	ctrlC := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

	screen, shutDown := update(t, newModel(sup, time.Now()), ctrlC)
	if shutDown == nil || screen.status() != stopping {
		t.Errorf("got %q, want every service stopping", screen.status())
	}
}

func TestWhileQuittingNothingStarts(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a")...)
	screen, _ := update(t, newModel(sup, time.Now()), press('q'))

	for _, msg := range []tea.Msg{press('s'), press('q'), quitMsg{}, doneMsg{errors.New("late")}} {
		var next tea.Cmd

		screen, next = update(t, screen, msg)
		if next != nil || screen.status() != stopping {
			t.Errorf("after a %T: got %q and a command, want neither", msg, screen.status())
		}
	}
}

func TestShutdownsErrorIsRunsResult(t *testing.T) {
	failure := errors.New("stop a: operation not permitted")
	sup := newSupervisor(t, io.Discard, sleeping("a")...)

	screen, _ := update(t, newModel(sup, time.Now()), shutDownMsg{err: failure})
	if !errors.Is(screen.err, failure) {
		t.Errorf("got %v, want %v", screen.err, failure)
	}
}
