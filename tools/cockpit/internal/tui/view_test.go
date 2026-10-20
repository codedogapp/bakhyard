package tui

import (
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

// terminalWidth is the width of a classic terminal, which the TUI must fit.
const terminalWidth = 80

func TestRowsShowEachServicesState(t *testing.T) {
	now := time.Now()

	got := rows([]supervisor.Status{
		{Name: "a", State: supervisor.Stopped},
		{Name: "b", State: supervisor.Running, PID: 41, Started: now.Add(-90 * time.Second)},
		// A start time after now, as a clock change can give, is no time up.
		{Name: "c", State: supervisor.Stopping, PID: 42, Started: now.Add(time.Second)},
		{Name: "d", State: supervisor.Exited},
		{Name: "e", State: supervisor.Exited, Err: errors.New("signal: killed")},
	}, now)

	want := []table.Row{
		{"a", "stopped", "", "", ""},
		{"b", "running", "41", "1m30s", ""},
		{"c", "stopping", "42", "0s", ""},
		{"d", "exited", "", "", "exit status 0"},
		{"e", "exited", "", "", "signal: killed"},
	}
	if !slices.EqualFunc(got, want, slices.Equal[table.Row]) {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
}

func TestTheViewFitsAClassicTerminal(t *testing.T) {
	sup := newSupervisor(t, io.Discard, sleeping("a")...)

	content := newModel(sup, time.Now()).View().Content
	if width := lipgloss.Width(content); width > terminalWidth {
		t.Errorf("got %d columns, want %d at most:\n%s", width, terminalWidth, content)
	}
}
