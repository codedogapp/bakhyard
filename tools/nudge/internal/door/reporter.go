package door

import (
	"context"
	"errors"
	"sync"
)

// ErrClosed means the reporter is closed, so it starts no session.
var ErrClosed = errors.New("the reporter is closed")

// Reporter keeps one session on the service for an agent, and starts it again when the service
// has forgotten it. Its session lasts until Close.
type Reporter struct {
	client  *Client
	project string
	// closed closes with Close, and ends the session with it.
	closed  chan struct{}
	closing sync.Once

	mu      sync.Mutex
	session Session
	live    bool
	end     context.CancelFunc
}

// NewReporter returns a reporter for project.
func NewReporter(client *Client, project string) *Reporter {
	return &Reporter{client: client, project: project, closed: make(chan struct{})}
}

// Close ends the session, and keeps the reporter from starting another. It doesn't wait for the
// service.
func (r *Reporter) Close() {
	r.closing.Do(func() { close(r.closed) })
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

	select {
	case <-r.closed:
		return ErrClosed
	default:
	}

	// The session outlives the call that starts it.
	ctx, end := context.WithCancel(context.Background())
	go r.endOnClose(ctx, end)

	session, err := r.client.Register(ctx, r.project)
	if err != nil {
		end()

		return err
	}

	r.session, r.end, r.live = session, end, true

	return nil
}

// endOnClose ends the session that ctx holds once the reporter closes.
func (r *Reporter) endOnClose(ctx context.Context, end context.CancelFunc) {
	select {
	case <-r.closed:
		end()
	case <-ctx.Done():
	}
}
