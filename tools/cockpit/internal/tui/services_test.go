package tui

import (
	"io"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

// sleeping returns services named names that sleep for a minute unless they are stopped.
func sleeping(names ...string) []manifest.Service {
	services := make([]manifest.Service, 0, len(names))
	for _, name := range names {
		services = append(services, manifest.Service{Name: name, Command: []string{"sleep", "60"}})
	}

	return services
}

// newSupervisor returns a supervisor of services that writes to out, and shuts it down when the
// test ends.
func newSupervisor(
	t *testing.T, out io.Writer, services ...manifest.Service,
) *supervisor.Supervisor {
	t.Helper()

	sup := supervisor.New(services, out)

	t.Cleanup(func() {
		err := sup.Shutdown()
		if err != nil {
			t.Error(err)
		}
	})

	return sup
}

// states returns the state of every service of sup, in start order, split by spaces.
func states(sup *supervisor.Supervisor) string {
	statuses := sup.Services()

	all := make([]string, 0, len(statuses))
	for _, status := range statuses {
		all = append(all, string(status.State))
	}

	return strings.Join(all, " ")
}
