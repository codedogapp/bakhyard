package supervisor

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/process"
)

// Start starts a service that is stopped or has exited. A running service is left alone, and
// one that is stopping can't start until it has stopped.
func (s *Supervisor) Start(name string) error {
	svc, err := s.find(name)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch svc.state {
	case Running:
		return nil
	case Stopping:
		return fmt.Errorf("start %s: it is still stopping", name)
	case Stopped, Exited:
	}

	lines := &lineWriter{out: s.out, name: name}

	proc, err := process.Start(svc.command, svc.vars, lines)
	if err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}

	current := &run{proc: proc, lines: lines, started: time.Now(), exited: make(chan struct{})}
	svc.state, svc.err, svc.run = Running, nil, current

	s.out.writeLine(cockpit, fmt.Sprintf("started %s (pid %d)", name, proc.PID()))
	s.notify()

	go s.watch(svc, current)

	return nil
}

// Stop stops a running service: SIGTERM to its process group, then SIGKILL if it hasn't exited
// after 10 s. It returns once the service has exited, or with an error if a signal can't be
// sent. A service that is stopping already is waited for, and one that isn't running is left
// alone.
func (s *Supervisor) Stop(name string) error {
	current, was, err := s.halt(name)
	if current == nil || err != nil {
		return err
	}

	if was == Running {
		s.out.writeLine(cockpit, "stopping "+name)

		err = current.proc.Stop(stopTimeout)
		if err != nil {
			return fmt.Errorf("stop %s: %w", name, err)
		}
	}

	<-current.exited

	return nil
}

// Kill sends SIGKILL at once to the process group of a service that is running or stopping,
// and returns once the service has exited, or with an error if the signal can't be sent. A
// service that isn't running is left alone.
func (s *Supervisor) Kill(name string) error {
	current, _, err := s.halt(name)
	if current == nil || err != nil {
		return err
	}

	s.out.writeLine(cockpit, "killing "+name)

	err = current.proc.Kill()
	if err != nil {
		return fmt.Errorf("kill %s: %w", name, err)
	}

	<-current.exited

	return nil
}

// Restart stops a service if it runs, then starts it.
func (s *Supervisor) Restart(name string) error {
	err := s.Stop(name)
	if err != nil {
		return err
	}

	return s.Start(name)
}

// StartAll starts the services in start order, up to the first that can't start.
func (s *Supervisor) StartAll() error {
	for _, svc := range s.services {
		err := s.Start(svc.name)
		if err != nil {
			return err
		}
	}

	return nil
}

// StopAll stops the services in reverse start order.
func (s *Supervisor) StopAll() error {
	var errs []error

	for _, svc := range slices.Backward(s.services) {
		errs = append(errs, s.Stop(svc.name))
	}

	return errors.Join(errs...)
}

// halt marks a running service as stopping. It returns the state the service was in, and its
// run if it is running or stopping.
func (s *Supervisor) halt(name string) (*run, State, error) {
	svc, err := s.find(name)
	if err != nil {
		return nil, "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	was := svc.state
	if was == Running {
		svc.state = Stopping

		s.notify()
	}

	return svc.run, was, nil
}

// watch waits for a run to exit, then records it: a service that was stopping has stopped, and
// any other has exited on its own.
func (s *Supervisor) watch(svc *service, current *run) {
	<-current.proc.Done()
	// The process has exited, so nothing writes to its lines any more.
	current.lines.flush()
	s.record(svc, current)
	close(current.exited)
}

func (s *Supervisor) record(svc *service, current *run) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if svc.state == Stopping {
		svc.state = Stopped
	} else {
		svc.state, svc.err = Exited, current.proc.Err()
	}

	svc.run = nil

	s.notify()
}
