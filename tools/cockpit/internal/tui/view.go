package tui

import (
	"fmt"
	"strconv"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/resources"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

const (
	help = "s start · x stop · K kill · r restart · a start all · Z stop all · q quit"

	nameWidth   = 12
	stateWidth  = 9
	pidWidth    = 7
	uptimeWidth = 9
	cpuWidth    = 7
	memoryWidth = 10
	exitWidth   = 30
	cellPadding = 1 // on each side of a cell
)

func (m model) View() tea.View {
	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		m.grid.View(),
		"",
		m.status(),
		lipgloss.NewStyle().Faint(true).Render(help),
	))
	view.AltScreen = true

	return view
}

// status is the line under the table: what the TUI is doing or asking, or the last error.
func (m model) status() string {
	switch {
	case m.quitting:
		return "Stopping every service…"
	case m.killing != "":
		return fmt.Sprintf("Kill %s? y/n", m.killing)
	case m.failure != nil:
		return m.failure.Error()
	case m.unmeasured != nil:
		return m.unmeasured.Error()
	default:
		return ""
	}
}

// newGrid returns an empty table with room for every service, and no colours (§7).
func newGrid(services int) table.Model {
	columns := []table.Column{
		{Title: "NAME", Width: nameWidth},
		{Title: "STATE", Width: stateWidth},
		{Title: "PID", Width: pidWidth},
		{Title: "UPTIME", Width: uptimeWidth},
		{Title: "CPU", Width: cpuWidth},
		{Title: "MEM", Width: memoryWidth},
		{Title: "EXIT", Width: exitWidth},
	}

	width := 0
	for _, column := range columns {
		width += column.Width + 2*cellPadding
	}

	return table.New(
		table.WithColumns(columns),
		table.WithWidth(width),
		table.WithHeight(1+services), // the header, then a line per service
		table.WithFocused(true),
		table.WithStyles(table.Styles{
			Header:   lipgloss.NewStyle().Bold(true).Padding(0, cellPadding),
			Cell:     lipgloss.NewStyle().Padding(0, cellPadding),
			Selected: lipgloss.NewStyle().Reverse(true),
		}),
	)
}

// rows returns a row per service, with its uptime at now and its last measured usage.
func rows(
	statuses []supervisor.Status, now time.Time, usage map[int]resources.Usage,
) []table.Row {
	all := make([]table.Row, 0, len(statuses))

	for _, status := range statuses {
		all = append(all, table.Row{
			status.Name, string(status.State), pid(status), uptime(status, now),
			cpu(usage, status.PID), memory(usage, status.PID), exit(status),
		})
	}

	return all
}

func pid(status supervisor.Status) string {
	if status.PID == 0 {
		return ""
	}

	return strconv.Itoa(status.PID)
}

func uptime(status supervisor.Status, now time.Time) string {
	if status.Started.IsZero() {
		return ""
	}

	return max(now.Sub(status.Started), 0).Round(time.Second).String()
}

// exit says how a service that exited on its own ended.
func exit(status supervisor.Status) string {
	switch {
	case status.State != supervisor.Exited:
		return ""
	case status.Err == nil:
		return "exit status 0"
	default:
		return status.Err.Error()
	}
}
