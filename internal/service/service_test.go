package service_test

import (
	"errors"
	"net"
	"net/http"
	"syscall"
	"testing"

	"github.com/codedogapp/bakhyard/internal/service"
)

func TestRunNeedsAnAddress(t *testing.T) {
	routes := func(func(string) string) (http.Handler, error) {
		t.Error("built the routes without an address")

		return http.NotFoundHandler(), nil
	}

	err := service.Run(getenvOf(nil), routes)
	if err == nil || err.Error() != "BAKHYARD_LISTEN is not set" {
		t.Errorf("got %v, want BAKHYARD_LISTEN is not set", err)
	}
}

func TestRunReportsRoutesThatFail(t *testing.T) {
	failed := errors.New("no peer")

	var name string

	routes := func(getenv func(string) string) (http.Handler, error) {
		name = getenv("BAKHYARD_NAME")

		return nil, failed
	}

	err := service.Run(getenvOf(map[string]string{
		"BAKHYARD_LISTEN": "tcp://127.0.0.1:8081",
		"BAKHYARD_NAME":   "Ada",
	}), routes)
	if !errors.Is(err, failed) {
		t.Errorf("got %v, want %v", err, failed)
	}

	if name != "Ada" {
		t.Errorf("the routes read BAKHYARD_NAME as %q, want Ada", name)
	}
}

func TestRunStopsOnSIGINTOrSIGTERM(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			routes := func(func(string) string) (http.Handler, error) {
				// Run catches the signal by now, so it stops Run rather than the test.
				err := syscall.Kill(syscall.Getpid(), sig)

				return http.NotFoundHandler(), err
			}

			listen := map[string]string{"BAKHYARD_LISTEN": "tcp://" + freeAddress(t)}

			err := service.Run(getenvOf(listen), routes)
			if err != nil {
				t.Errorf("got %v, want nil", err)
			}
		})
	}
}

func getenvOf(vars map[string]string) func(string) string {
	return func(key string) string {
		return vars[key]
	}
}

// freeAddress returns a loopback address whose port was free a moment ago.
func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	err = listener.Close()
	if err != nil {
		t.Fatal(err)
	}

	return listener.Addr().String()
}
