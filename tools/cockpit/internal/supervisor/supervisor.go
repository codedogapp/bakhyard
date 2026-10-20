// Package supervisor runs services in the foreground: it starts them in order, marks each line
// of their output with the service's name, and stops them in reverse order.
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
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

// service is a service that has started.
type service struct {
	name  string
	proc  *process.Process
	lines *lineWriter
}

type supervisor struct {
	out     *output
	started []*service
	exits   chan *service // each started service, once it has exited
}

// Run starts the services in the order given, and runs them until ctx ends or one of them
// exits. Then it stops the others in reverse order, each with SIGTERM, and SIGKILL after 10 s.
// It returns nil when ctx ended, and an error when a service couldn't start or exited.
func Run(ctx context.Context, services []manifest.Service, out io.Writer) error {
	sup := &supervisor{
		out:   newOutput(out, services),
		exits: make(chan *service, len(services)),
	}

	err := sup.startAll(services)
	if err == nil {
		err = sup.wait(ctx)
	}

	return errors.Join(err, sup.stopAll(), sup.out.failure())
}

// startAll starts the services in order, up to the first that can't start.
func (s *supervisor) startAll(services []manifest.Service) error {
	for _, svc := range services {
		lines := &lineWriter{out: s.out, name: svc.Name}

		proc, err := process.Start(svc.Command, lines)
		if err != nil {
			return fmt.Errorf("start %s: %w", svc.Name, err)
		}

		running := &service{name: svc.Name, proc: proc, lines: lines}
		s.started = append(s.started, running)
		s.out.writeLine(cockpit, fmt.Sprintf("started %s (pid %d)", svc.Name, proc.PID()))

		go func() {
			<-proc.Done()

			s.exits <- running
		}()
	}

	return nil
}

// wait returns when ctx ends, or when a service exits, with the exit as an error.
func (s *supervisor) wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case svc := <-s.exits:
		return svc.exitError()
	}
}

// stopAll stops the services that are still running, in reverse order.
func (s *supervisor) stopAll() error {
	var errs []error

	for _, svc := range slices.Backward(s.started) {
		errs = append(errs, s.stop(svc))
	}

	return errors.Join(errs...)
}

func (s *supervisor) stop(svc *service) error {
	select {
	case <-svc.proc.Done():
	default:
		s.out.writeLine(cockpit, "stopping "+svc.name)

		err := svc.proc.Stop(stopTimeout)
		if err != nil {
			return fmt.Errorf("stop %s: %w", svc.name, err)
		}
	}

	// The service has exited, so nothing writes to its lines any more.
	svc.lines.flush()

	return nil
}

func (svc *service) exitError() error {
	err := svc.proc.Err()
	if err == nil {
		return fmt.Errorf("%s exited", svc.name)
	}

	return fmt.Errorf("%s exited: %w", svc.name, err)
}
