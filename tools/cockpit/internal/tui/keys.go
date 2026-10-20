package tui

import tea "charm.land/bubbletea/v2"

// press handles a key: an action on the selected service, the answer to a kill's question, or
// a move in the table.
func (m model) press(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.quitting:
		return m, nil
	case m.killing != "":
		return m.confirm(msg.String())
	}

	name := m.selected()

	switch msg.String() {
	case "q", "ctrl+c":
		return m.quit()
	case "K":
		m.killing = name

		return m, nil
	}

	action, ok := m.actions()[msg.String()]
	if ok {
		return m, perform(func() error { return action(name) })
	}

	var cmd tea.Cmd

	m.grid, cmd = m.grid.Update(msg)

	return m, cmd
}

// actions maps a key to what it does, given the selected service.
func (m model) actions() map[string]func(name string) error {
	return map[string]func(string) error{
		"s": m.sup.Start,
		"x": m.sup.Stop,
		"r": m.sup.Restart,
		"a": func(string) error { return m.sup.StartAll() },
		"Z": func(string) error { return m.sup.StopAll() },
	}
}

// confirm kills the service that K picked if the answer is y, and drops the kill otherwise.
func (m model) confirm(answer string) (tea.Model, tea.Cmd) {
	name := m.killing
	m.killing = ""

	if answer != "y" {
		return m, nil
	}

	return m, perform(func() error { return m.sup.Kill(name) })
}

// quit stops every service in the background. The TUI exits once they have all stopped.
func (m model) quit() (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, nil
	}

	m.quitting = true

	return m, func() tea.Msg { return shutDownMsg{err: m.sup.Shutdown()} }
}

func (m model) selected() string {
	row := m.grid.SelectedRow()
	if row == nil {
		return ""
	}

	return row[0]
}
