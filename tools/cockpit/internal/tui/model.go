package tui

import (
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/probe"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/resources"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

// model is the TUI's state.
type model struct {
	sup        *supervisor.Supervisor
	prober     *probe.Prober
	grid       table.Model
	now        time.Time               // the last tick, which the uptimes are measured at
	usage      map[int]resources.Usage // the last CPU and memory measured, by PID
	unmeasured error                   // why the last measuring failed, until one works
	checks     map[int]error           // how the last health checks went, by PID
	failure    error                   // the last action's error, until the next action ends
	killing    string                  // the service to kill once y confirms it
	quitting   bool                    // Shutdown is stopping every service
	err        error                   // Shutdown's error, once it has returned
}

func newModel(sup *supervisor.Supervisor, now time.Time) model {
	start := model{sup: sup, prober: probe.New(), grid: newGrid(len(sup.Services())), now: now}

	return start.refresh()
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		perform(m.startAll),
		waitForChange(m.sup),
		tick(),
		measureLater(m.sup),
		checkLater(m.sup, m.prober),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.press(msg)
	case changedMsg:
		return m.refresh(), waitForChange(m.sup)
	case tickMsg:
		m.now = time.Time(msg)

		return m.refresh(), tick()
	case measuredMsg:
		m.usage, m.unmeasured = msg.usage, msg.err

		// The next measuring waits for this one, so they never overlap.
		return m.refresh(), measureLater(m.sup)
	case checkedMsg:
		m.checks = msg

		// The same goes for the checks.
		return m.refresh(), checkLater(m.sup, m.prober)
	case doneMsg:
		m.failure = msg.err

		return m, nil
	case quitMsg:
		return m.quit()
	case shutDownMsg:
		m.err = msg.err

		return m, tea.Quit
	}

	return m, nil
}

// refresh puts the services' current state in the table.
func (m model) refresh() model {
	m.grid.SetRows(rows(m.sup.Services(), m.now, m.usage, m.checks))

	return m
}
