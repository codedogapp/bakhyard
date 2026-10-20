package supervisor

import (
	"context"
	"fmt"
	"time"
)

const (
	// waitTimeout is how long StartAll waits for a service's dependencies to be healthy, before
	// it gives up on the service (decision 61).
	waitTimeout = 10 * time.Second
	// waitEvery is how long StartAll waits after a dependency fails a check, before the next.
	waitEvery = 100 * time.Millisecond
)

// awaitDependencies returns once every service that svc depends on is healthy, or with why
// not: a dependency isn't running, isn't healthy within the wait timeout, or ctx has ended.
func (s *Supervisor) awaitDependencies(ctx context.Context, svc *service) error {
	deadline := time.After(s.waitTimeout)

	for _, name := range svc.dependsOn {
		dependency, err := s.find(name)
		if err == nil {
			err = s.awaitHealthy(ctx, dependency, deadline)
		}

		if err != nil {
			return fmt.Errorf("start %s: %w", svc.name, err)
		}
	}

	return nil
}

// awaitHealthy checks svc until it is healthy. It gives up once svc isn't running, deadline has
// passed or ctx has ended.
func (s *Supervisor) awaitHealthy(
	ctx context.Context, svc *service, deadline <-chan time.Time,
) error {
	for {
		err := s.running(svc)
		if err != nil {
			return err
		}

		err = s.prober.Check(ctx, svc.listen)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("%s isn't healthy within %s: %w", svc.name, s.waitTimeout, err)
		case <-time.After(waitEvery):
		}
	}
}

// running returns nil if svc runs, and why not otherwise.
func (s *Supervisor) running(svc *service) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch svc.state {
	case Running:
		return nil
	case Exited:
		return exited(svc.name, svc.err)
	case Stopped, Stopping:
	}

	return fmt.Errorf("%s isn't running", svc.name)
}
