// Package supervisor runs services, each in a process group of its own. It starts, stops,
// kills and restarts each one, keeps track of its state, and marks each line of its output
// with its name.
package supervisor

import (
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/process"
)

const (
	// cockpit marks the supervisor's own lines.
	cockpit = "cockpit"
	// stopTimeout is how long a service has to exit after SIGTERM, before SIGKILL.
	stopTimeout = 10 * time.Second
)

// Supervisor runs the services of a manifest. Its methods are safe to call at the same time.
type Supervisor struct {
	services     []*service // in start order; the slice and the names never change
	out          *output
	changed      chan struct{}
	mu           sync.Mutex // guards shuttingDown, and each service's state, err and run
	shuttingDown bool       // set by Shutdown, after which no service starts
}

type service struct {
	name    string
	command []string
	vars    []string
	listen  string
	state   State
	err     error // how the service exited, when it exited on its own
	run     *run  // the process, while the service runs or stops
}

// run is one run of a service's command.
type run struct {
	proc    *process.Process
	lines   *lineWriter
	started time.Time
	exited  chan struct{} // closed once the supervisor has recorded the exit
}

// New returns a supervisor for services, given in start order, with every service stopped.
// The services' output and the supervisor's own lines go to out.
func New(services []manifest.Service, out io.Writer) *Supervisor {
	vars := environment(services)
	sup := &Supervisor{out: newOutput(out, services), changed: make(chan struct{}, 1)}

	for _, svc := range services {
		sup.services = append(sup.services, &service{
			name:    svc.Name,
			command: svc.Command,
			vars:    vars[svc.Name],
			listen:  svc.Listen,
			state:   Stopped,
		})
	}

	return sup
}

// Services returns the status of every service, in start order.
func (s *Supervisor) Services() []Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	statuses := make([]Status, 0, len(s.services))

	for _, svc := range s.services {
		status := Status{Name: svc.name, Listen: svc.listen, State: svc.state, Err: svc.err}
		if svc.run != nil {
			status.PID, status.Started = svc.run.proc.PID(), svc.run.started
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// Changed returns a channel that receives a value after a service's state changes, for one
// reader. It holds one value at most, so several changes can give one value: read Services
// after each.
func (s *Supervisor) Changed() <-chan struct{} {
	return s.changed
}

func (s *Supervisor) notify() {
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

func (s *Supervisor) find(name string) (*service, error) {
	index := slices.IndexFunc(s.services, func(svc *service) bool { return svc.name == name })
	if index < 0 {
		return nil, fmt.Errorf("no service %q", name)
	}

	return s.services[index], nil
}
