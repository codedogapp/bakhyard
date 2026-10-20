package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
)

// Run starts the services in the order given, each once the services it depends on are
// healthy, and runs them until ctx ends or one of them exits. Then it stops the others in
// reverse order, each with SIGTERM, and SIGKILL after 10 s. It returns nil when ctx ended, and
// an error when a service couldn't start or exited.
func Run(ctx context.Context, services []manifest.Service, out io.Writer) error {
	sup := New(services, out)

	err := sup.StartAll(ctx)
	if err == nil {
		err = sup.wait(ctx)
	}

	// ctx ended while a service waited to start: a stop, not a failure.
	if errors.Is(err, context.Canceled) {
		err = nil
	}

	return errors.Join(err, sup.StopAll(), sup.out.failure())
}

// wait returns when ctx ends, or when a service exits on its own, with the exit as an error.
func (s *Supervisor) wait(ctx context.Context) error {
	for {
		err := s.exitError()
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-s.changed:
		}
	}
}

// exitError returns the first service, in start order, that exited on its own, as an error.
func (s *Supervisor) exitError() error {
	for _, status := range s.Services() {
		if status.State == Exited {
			return exited(status.Name, status.Err)
		}
	}

	return nil
}

// exited says that the service called name exited on its own, and how: err, or nil for exit
// status 0.
func exited(name string, err error) error {
	if err == nil {
		return fmt.Errorf("%s exited", name)
	}

	return fmt.Errorf("%s exited: %w", name, err)
}
