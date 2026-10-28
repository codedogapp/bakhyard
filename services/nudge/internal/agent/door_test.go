package agent_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/nudge/internal/agent"
	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
	"github.com/codedogapp/bakhyard/services/nudge/internal/migrations"
	"github.com/codedogapp/bakhyard/services/nudge/internal/notifier"
	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
	"github.com/codedogapp/bakhyard/services/nudge/internal/session"
)

const (
	id1   = "11111111-1111-4111-8111-111111111111"
	merge = `{"id":"` + id1 + `","project":"atlas","text":"Merge?","details":"CI is green.",` +
		`"options":["Yes","No"]}`
)

// pusher records the pushes it is asked for.
type pusher struct {
	pushes chan notifier.Message
	err    error
}

func (p *pusher) Send(_ context.Context, message notifier.Message) error {
	p.pushes <- message

	return p.err
}

// door is a running agent door, over a database of its own.
type door struct {
	server    *httptest.Server
	questions *question.Store
	sessions  *session.Registry
	hub       *change.Hub
	pushes    chan notifier.Message
	pusher    *pusher
	logs      *bytes.Buffer
	stop      context.CancelFunc
}

func newDoor(t *testing.T) *door {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	hub := &change.Hub{}
	pushes := make(chan notifier.Message, 10)
	stopping, stop := context.WithCancel(t.Context())
	gate := &door{
		questions: question.NewStore(pool, hub),
		sessions:  session.NewRegistry(hub),
		hub:       hub,
		pushes:    pushes,
		pusher:    &pusher{pushes: pushes},
		logs:      &bytes.Buffer{},
		stop:      stop,
	}

	router := chi.NewRouter()
	agent.Door{
		Questions: gate.questions,
		Sessions:  gate.sessions,
		Pusher:    gate.pusher,
		Logger:    slog.New(slog.NewTextHandler(gate.logs, nil)),
		Stopping:  stopping,
	}.Register(router)

	gate.server = httptest.NewServer(router)
	t.Cleanup(gate.server.Close)

	return gate
}

type reply struct {
	status int
	body   string
}

// send makes a request in the background, and returns where its reply will come.
func (gate *door) send(ctx context.Context, method, path, body string) <-chan reply {
	done := make(chan reply, 1)

	go func() {
		req, err := http.NewRequestWithContext(
			ctx,
			method,
			gate.server.URL+path,
			strings.NewReader(body),
		)
		if err != nil {
			done <- reply{body: err.Error()}

			return
		}

		resp, err := gate.server.Client().Do(req)
		if err != nil {
			done <- reply{body: err.Error()}

			return
		}

		defer func() { _ = resp.Body.Close() }()

		raw, _ := io.ReadAll(resp.Body)
		done <- reply{resp.StatusCode, string(raw)}
	}()

	return done
}

// call makes a request and waits for its reply.
func (gate *door) call(t *testing.T, method, path, body string) reply {
	t.Helper()

	return <-gate.send(t.Context(), method, path, body)
}

// pusherFails makes the pushes fail with err.
func (gate *door) pusherFails(err error) { gate.pusher.err = err }

func (gate *door) noMorePushes(t *testing.T) {
	t.Helper()

	select {
	case extra := <-gate.pushes:
		t.Errorf("a second push: %+v", extra)
	default:
	}
}
