package probe_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/probe"
)

func TestAServiceThatAnswers200IsUp(t *testing.T) {
	requests := make(chan string, 1)
	listen := serve(t, func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.RequestURI()

		w.WriteHeader(http.StatusOK)
	})

	err := probe.New().Check(t.Context(), listen)
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}

	if got := <-requests; got != "GET /healthz" {
		t.Errorf("the service got %q, want GET /healthz", got)
	}
}

func TestAnyOtherAnswerIsDown(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{
			name:    "an error",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) },
			want:    "/healthz answered 503 Service Unavailable",
		},
		{
			name:    "another success",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
			want:    "/healthz answered 204 No Content",
		},
		{
			name: "a redirect to a page that answers 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/healthz" {
					http.Redirect(w, r, "/elsewhere", http.StatusFound)
				}
			},
			want: "/healthz answered 302 Found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := probe.New().Check(t.Context(), serve(t, test.handler))
			if err == nil || !strings.HasSuffix(err.Error(), test.want) {
				t.Errorf("got %v, want an error ending in %q", err, test.want)
			}
		})
	}
}

func TestAServiceThatTakesNoConnectionIsDown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	listen := "tcp://" + listener.Addr().String()

	err = listener.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = probe.New().Check(t.Context(), listen)
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("got %v, want the connection refused", err)
	}
}

func TestACheckEndsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	listen := serve(t, func(_ http.ResponseWriter, r *http.Request) {
		// The service has the request, and never answers: the client gives up.
		cancel()
		<-r.Context().Done()
	})

	err := probe.New().Check(ctx, listen)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want the check cancelled", err)
	}
}

func TestACheckGivesUpAfterASecond(t *testing.T) {
	listen := serve(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})

	err := probe.New().Check(t.Context(), listen)
	if err == nil || !strings.HasSuffix(err.Error(), "/healthz\": no answer within 1s") {
		t.Errorf("got %v, want no answer within 1s", err)
	}
}

func TestABadListenAddressIsAnError(t *testing.T) {
	for _, listen := range []string{"unix:///tmp/a.sock", "tcp://127.0.0.1", "tcp://a host:80"} {
		t.Run(listen, func(t *testing.T) {
			err := probe.New().Check(t.Context(), listen)
			if err == nil {
				t.Error("got nil, want an error")
			}
		})
	}
}

func TestEachCheckOpensANewConnection(t *testing.T) {
	var connections atomic.Int32

	srv := httptest.NewUnstartedServer(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	)
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	prober := probe.New()
	listen := "tcp://" + srv.Listener.Addr().String()

	for range 2 {
		err := prober.Check(t.Context(), listen)
		if err != nil {
			t.Fatal(err)
		}
	}

	if got := connections.Load(); got != 2 {
		t.Errorf("got %d connections for 2 checks, want 2", got)
	}
}

// serve starts a service that answers with handler, and returns its listen address.
func serve(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return "tcp://" + srv.Listener.Addr().String()
}
