// Package tui shows the cockpit's services in a table and controls them from the keyboard.
package tui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

type (
	changedMsg  struct{}            // a service's state changed
	tickMsg     time.Time           // a second has passed
	doneMsg     struct{ err error } // an action has ended
	quitMsg     struct{}            // Run's ctx has ended
	shutDownMsg struct{ err error } // Shutdown has returned
)

// Run shows the services of sup and controls them until the user quits or ctx ends. It starts
// every service when it opens, and stops them all before it returns. opts go to the Bubble Tea
// program, so tests can give it their own input and output.
func Run(ctx context.Context, sup *supervisor.Supervisor, opts ...tea.ProgramOption) error {
	opts = append(opts, tea.WithoutSignalHandler())
	program := tea.NewProgram(newModel(sup, time.Now()), opts...)

	stop := context.AfterFunc(ctx, func() { program.Send(quitMsg{}) })
	defer stop()

	final, err := program.Run()
	if quit, ok := final.(model); ok && err == nil {
		return quit.err
	}

	// The program ended before its quit could stop the services.
	return errors.Join(err, sup.Shutdown())
}

// perform runs action in the background, then reports how it went.
func perform(action func() error) tea.Cmd {
	return func() tea.Msg { return doneMsg{err: action()} }
}

// waitForChange waits for a service's state to change. The TUI is Changed's only reader.
func waitForChange(sup *supervisor.Supervisor) tea.Cmd {
	return func() tea.Msg {
		<-sup.Changed()

		return changedMsg{}
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(at time.Time) tea.Msg { return tickMsg(at) })
}
