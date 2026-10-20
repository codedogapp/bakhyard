package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestRunReportsAnAddressInUse(t *testing.T) {
	taken := listen(t)

	err := Run(t.Context(), taken.Addr().String(), http.NotFoundHandler())
	if !errors.Is(err, syscall.EADDRINUSE) {
		t.Errorf("got %v, want the address in use", err)
	}
}

func TestRunReturnsWhenTheContextHasEnded(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := Run(ctx, "127.0.0.1:0", http.NotFoundHandler())
	if err != nil {
		t.Errorf("got %v, want nil", err)
	}
}

func TestServeFinishesTheRequestsInFlight(t *testing.T) {
	listener := &closeNotifier{Listener: listen(t), closed: make(chan struct{})}
	listener.notify = sync.OnceFunc(func() { close(listener.closed) })
	arrived := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(arrived)
		// The shutdown has begun once the server closes its listener.
		<-listener.closed

		_, _ = io.WriteString(w, "done")
	})
	ctx, cancel := context.WithCancel(t.Context())
	served := make(chan error, 1)

	go func() {
		served <- serve(ctx, listener, handler, time.Minute)
	}()

	answer := get(t.Context(), listener.Addr())

	<-arrived
	cancel()

	got := <-answer
	if got.err != nil || got.body != "done" {
		t.Errorf("got %q, %v, want the whole answer", got.body, got.err)
	}

	err := <-served
	if err != nil {
		t.Errorf("got %v, want nil", err)
	}
}

func TestServeStopsWaitingAfterTheTimeout(t *testing.T) {
	listener := listen(t)
	arrived := make(chan struct{})
	release := make(chan struct{})
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(arrived)
		<-release
	})
	ctx, cancel := context.WithCancel(t.Context())
	served := make(chan error, 1)

	go func() {
		served <- serve(ctx, listener, handler, 0)
	}()

	answer := get(t.Context(), listener.Addr())

	<-arrived
	cancel()

	err := <-served
	if !errors.Is(err, context.DeadlineExceeded) ||
		err.Error() != "shut down: context deadline exceeded" {
		t.Errorf("got %v, want the shutdown's deadline", err)
	}

	close(release)
	<-answer
}

func TestServeReportsAFailedAccept(t *testing.T) {
	listener := listen(t)

	err := listener.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = serve(t.Context(), listener, http.NotFoundHandler(), time.Minute)
	if !errors.Is(err, net.ErrClosed) {
		t.Errorf("got %v, want the closed listener", err)
	}
}

func listen(t *testing.T) net.Listener {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	return listener
}

// closeNotifier closes closed when the server closes it.
type closeNotifier struct {
	net.Listener

	closed chan struct{}
	notify func()
}

func (l *closeNotifier) Close() error {
	l.notify()

	return l.Listener.Close()
}

type answer struct {
	body string
	err  error
}

// get sends a GET request to addr, and returns a channel that receives the answer.
func get(ctx context.Context, addr net.Addr) <-chan answer {
	result := make(chan answer, 1)

	go func() {
		body, err := fetch(ctx, "http://"+addr.String()+"/")
		result <- answer{body: body, err: err}
	}()

	return result
}

func fetch(ctx context.Context, url string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}

	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)

	return string(body), err
}
