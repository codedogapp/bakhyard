// Package session keeps the agent sessions that are running, each with its last note. A session
// lasts as long as its owner keeps it; nothing about it is stored.
package session

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
)

var (
	// ErrNotFound means the session has ended.
	ErrNotFound = errors.New("no such session")
	// ErrNoNote means the note is empty.
	ErrNoNote = errors.New("the note is empty")
)

// Session is a running agent session. Its note is the agent's last report, empty until it makes
// one.
type Session struct {
	ID      string    `json:"id"`
	Project string    `json:"project"`
	Note    string    `json:"note"`
	NotedAt time.Time `json:"noted_at"`
}

// Registry holds the running sessions, and publishes to hub whenever they change. It is safe for
// use by several goroutines at once.
type Registry struct {
	hub     *change.Hub
	mu      sync.Mutex
	running []*Session // oldest first
}

// NewRegistry returns an empty registry that publishes to hub.
func NewRegistry(hub *change.Hub) *Registry {
	return &Registry{hub: hub}
}

// Start adds a session for project, and returns it with the function that ends it. Ending it twice
// is harmless.
func (r *Registry) Start(project string) (Session, func()) {
	started := &Session{ID: uuid.New().String(), Project: project}

	r.mu.Lock()
	r.running = append(r.running, started)
	r.mu.Unlock()
	r.hub.Publish()

	return *started, func() {
		r.mu.Lock()
		r.running = slices.DeleteFunc(r.running, func(s *Session) bool { return s == started })
		r.mu.Unlock()
		r.hub.Publish()
	}
}

// Report sets the note of the session with sessionID, replacing its last one.
func (r *Registry) Report(sessionID, note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return ErrNoNote
	}

	r.mu.Lock()

	index := slices.IndexFunc(r.running, func(s *Session) bool { return s.ID == sessionID })
	if index < 0 {
		r.mu.Unlock()

		return ErrNotFound
	}

	r.running[index].Note = note
	r.running[index].NotedAt = time.Now()
	r.mu.Unlock()
	r.hub.Publish()

	return nil
}

// List returns the running sessions, newest first.
func (r *Registry) List() []Session {
	r.mu.Lock()
	defer r.mu.Unlock()

	listed := make([]Session, 0, len(r.running))
	for _, s := range slices.Backward(r.running) {
		listed = append(listed, *s)
	}

	return listed
}
