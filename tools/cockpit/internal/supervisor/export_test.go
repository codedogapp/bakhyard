package supervisor

import "time"

// SetWaitTimeout sets how long StartAll waits for a service's dependencies, so a test needn't
// wait 10 s for a timeout.
func (s *Supervisor) SetWaitTimeout(timeout time.Duration) {
	s.waitTimeout = timeout
}
