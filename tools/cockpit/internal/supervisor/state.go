package supervisor

import "time"

// State is where a service is in its life.
type State string

const (
	// Stopped is a service that never started, or that the supervisor stopped.
	Stopped State = "stopped"
	// Running is a service whose process runs.
	Running State = "running"
	// Stopping is a service that the supervisor is stopping, and that hasn't exited yet.
	Stopping State = "stopping"
	// Exited is a service that exited on its own.
	Exited State = "exited"
)

// Status is a service's state at one moment.
type Status struct {
	Name  string
	State State
	// PID and Started are the process's ID and start time while it runs or stops, and zero
	// otherwise.
	PID     int
	Started time.Time
	// Err is how the service exited, when it exited on its own; nil for exit status 0.
	Err error
}
