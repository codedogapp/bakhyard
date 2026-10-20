package tui

import (
	"context"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/resources"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

const (
	measureEvery   = 2 * time.Second
	measureTimeout = time.Second
	mebibyte       = 1 << 20
)

// measuredMsg is the CPU and memory of each service's process, by PID, or why measuring failed.
type measuredMsg struct {
	usage map[int]resources.Usage
	err   error
}

// measureLater measures the services' CPU and memory once measureEvery has passed.
func measureLater(sup *supervisor.Supervisor) tea.Cmd {
	return tea.Tick(measureEvery, func(time.Time) tea.Msg { return measure(sup) })
}

// measure measures the CPU and memory of every service that has a process.
func measure(sup *supervisor.Supervisor) measuredMsg {
	ctx, cancel := context.WithTimeout(context.Background(), measureTimeout)
	defer cancel()

	usage, err := resources.Measure(ctx, processIDs(sup.Services()))

	return measuredMsg{usage: usage, err: err}
}

// processIDs returns the PID of each service that has a process.
func processIDs(statuses []supervisor.Status) []int {
	var pids []int

	for _, status := range statuses {
		if status.PID != 0 {
			pids = append(pids, status.PID)
		}
	}

	return pids
}

// cpu is the share of one core that the process pid used, if it has been measured.
func cpu(usage map[int]resources.Usage, pid int) string {
	use, ok := usage[pid]
	if !ok {
		return ""
	}

	return strconv.FormatFloat(use.CPU, 'f', 1, 64) + "%"
}

// memory is the resident memory of the process pid, if it has been measured.
func memory(usage map[int]resources.Usage, pid int) string {
	use, ok := usage[pid]
	if !ok {
		return ""
	}

	return strconv.FormatFloat(float64(use.Memory)/mebibyte, 'f', 1, 64) + " MiB"
}
