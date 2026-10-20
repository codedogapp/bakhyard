package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/probe"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

const (
	checkEvery = 2 * time.Second
	// checkTimeout is how long a service has to answer its health check (decision 60).
	checkTimeout = time.Second
)

// checkedMsg is how each service's health check went, by PID: nil if the service is up, and
// why it is down otherwise.
type checkedMsg map[int]error

// checkLater checks the services' health once checkEvery has passed.
func checkLater(sup *supervisor.Supervisor, prober *probe.Prober) tea.Cmd {
	return tea.Tick(checkEvery, func(time.Time) tea.Msg { return check(sup, prober) })
}

// check checks every service that has a process, all at once.
func check(sup *supervisor.Supervisor, prober *probe.Prober) checkedMsg {
	timedOut := fmt.Errorf("no answer within %s", checkTimeout)

	ctx, cancel := context.WithTimeoutCause(context.Background(), checkTimeout, timedOut)
	defer cancel()

	var (
		group sync.WaitGroup
		mutex sync.Mutex
	)

	checked := checkedMsg{}

	for _, status := range sup.Services() {
		if status.PID == 0 {
			continue
		}

		group.Go(func() {
			err := prober.Check(ctx, status.Listen)

			mutex.Lock()
			defer mutex.Unlock()

			checked[status.PID] = err
		})
	}

	group.Wait()

	return checked
}

// health says whether the process pid passed its last check, if it has had one.
func health(checks map[int]error, pid int) string {
	err, ok := checks[pid]

	switch {
	case !ok:
		return ""
	case err != nil:
		return "down"
	default:
		return "up"
	}
}

// down says why the selected service is down, if it failed its last check. The table has a row
// for each service, in the supervisor's order.
func (m model) down() string {
	statuses := m.sup.Services()

	cursor := m.grid.Cursor()
	if cursor < 0 || cursor >= len(statuses) {
		return ""
	}

	status := statuses[cursor]

	err := m.checks[status.PID]
	if err == nil {
		return ""
	}

	return fmt.Sprintf("%s is down: %v", status.Name, err)
}
