package tui

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

// backlog is how many lines the supervisor can write before a test reads them.
const backlog = 16

func TestRunStartsEveryServiceAndStopsThemOnQuit(t *testing.T) {
	input, typing := io.Pipe()
	sup, result := runTUI(t, t.Context(), tea.WithInput(input))

	_, err := typing.Write([]byte("q"))
	if err != nil {
		t.Fatal(err)
	}

	// At the end of the input, Bubble Tea's reader stops at once when the program ends.
	err = typing.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = <-result
	if err != nil {
		t.Fatal(err)
	}

	if got := states(sup); got != "stopped stopped" {
		t.Errorf("got %q, want every service stopped", got)
	}
}

func TestRunStopsEveryServiceWhenItsContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	sup, result := runTUI(t, ctx, tea.WithInput(strings.NewReader("")))

	cancel()

	err := <-result
	if err != nil {
		t.Fatal(err)
	}

	if got := states(sup); got != "stopped stopped" {
		t.Errorf("got %q, want every service stopped", got)
	}
}

func TestRunStopsEveryServiceWhenTheProgramFails(t *testing.T) {
	programCtx, kill := context.WithCancel(t.Context())
	sup, result := runTUI(t, t.Context(),
		tea.WithContext(programCtx), tea.WithInput(strings.NewReader("")))

	kill()

	err := <-result
	if !errors.Is(err, tea.ErrProgramKilled) {
		t.Errorf("got %v, want the program killed", err)
	}

	if got := states(sup); got != "stopped stopped" {
		t.Errorf("got %q, want every service stopped", got)
	}
}

// lines gets what the supervisor writes, a line per write.
type lines chan string

func (l lines) Write(data []byte) (int, error) {
	l <- string(data)

	return len(data), nil
}

// runTUI runs Run in the background with two services, a and b, and opts. It returns once both
// have started, with the supervisor and a channel that gets Run's result.
func runTUI(
	t *testing.T, ctx context.Context, opts ...tea.ProgramOption,
) (*supervisor.Supervisor, <-chan error) {
	t.Helper()

	out := make(lines, backlog)
	sup := newSupervisor(t, out, sleeping("a", "b")...)
	result := make(chan error, 1)

	go func() {
		result <- Run(ctx, sup, append(opts, tea.WithOutput(io.Discard))...)
	}()

	for started := 0; started < len(sup.Services()); {
		if strings.Contains(<-out, " | started ") {
			started++
		}
	}

	return sup, result
}
