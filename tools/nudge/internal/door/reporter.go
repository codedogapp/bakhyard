package door

import (
	"context"
	"errors"
	"sync"
)

// Reporter keeps one session on the service for an agent, and starts it again when the service
// has forgotten it.
type Reporter struct {
	ctx     context.Context
	client  *Client
	project string

	mu      sync.Mutex
	session Session
	live    bool
	end     context.CancelFunc
}

// NewReporter returns a reporter for project. Its sessions end when ctx ends.
func NewReporter(ctx context.Context, client *Client, project string) *Reporter {
	return &Reporter{ctx: ctx, client: client, project: project}
}

// Register starts the session, in place of the one it has.
func (r *Reporter) Register() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.register()
}

// Report sets the session's note. When the session isn't there, because the service hasn't had it
// yet or has restarted, it starts a new one and reports on that.
func (r *Reporter) Report(ctx context.Context, note string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.live {
		err := r.session.Report(ctx, note)
		if !errors.Is(err, ErrSessionGone) {
			return err
		}
	}

	if err := r.register(); err != nil {
		return err
	}

	return r.session.Report(ctx, note)
}

func (r *Reporter) register() error {
	if r.live {
		r.end()
		r.live = false
	}

	ctx, end := context.WithCancel(r.ctx)

	session, err := r.client.Register(ctx, r.project)
	if err != nil {
		end()

		return err
	}

	r.session, r.end, r.live = session, end, true

	return nil
}
