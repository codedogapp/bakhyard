package week_test

import (
	"bytes"
	"context"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/codedogapp/bakhyard/internal/env"
	timev1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1/timev1connect"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
	"github.com/codedogapp/bakhyard/services/home/internal/week"
)

func TestAskAnswersTimesWeek(t *testing.T) {
	tests := map[string]struct {
		answer *timev1.GetWeekResponse
		want   *screen.Week
	}{
		"a week": {
			answer: weekOf(31.5, 40),
			want:   &screen.Week{LoggedHours: 31.5, TargetHours: 40},
		},
		"a week with nothing logged": {
			answer: weekOf(0, 40),
			want:   &screen.Week{LoggedHours: 0, TargetHours: 40},
		},
		"no week": {answer: &timev1.GetWeekResponse{}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			address := serveTime(t, test.answer, nil)
			got, state, logs := ask(t, env.Peer{Name: "time", Address: address})

			if !sameWeek(got, test.want) || state != screen.StateOK || logs != "" {
				t.Errorf("got %+v, %q and logs %q, want %+v, ok and no logs", got, state, logs,
					test.want)
			}
		})
	}
}

func TestAskMarksTheWeekUnavailableAndLogsIt(t *testing.T) {
	// Hours that aren't finite would cut the whole Home's JSON short.
	tests := map[string]string{
		"time fails": serveTime(t, nil,
			connect.NewError(connect.CodeUnavailable, "the week isn't read yet")),
		"time is down":          downAddress(t),
		"NaN hours logged":      serveTime(t, weekOf(math.NaN(), 40), nil),
		"an infinite target":    serveTime(t, weekOf(31.5, math.Inf(1)), nil),
		"minus infinite logged": serveTime(t, weekOf(math.Inf(-1), 40), nil),
	}

	for name, address := range tests {
		t.Run(name, func(t *testing.T) {
			got, state, logs := ask(t, env.Peer{Name: "time", Address: address})

			if got != nil || state != screen.StateUnavailable {
				t.Errorf("got %+v and %q, want no week, unavailable", got, state)
			}

			if !strings.Contains(logs, `level=ERROR msg="get the week" peer=time err=`) ||
				strings.Count(logs, "\n") != 1 {
				t.Errorf("logged %q, want one line about time", logs)
			}
		})
	}
}

func TestAskAsksThePeerNamedTime(t *testing.T) {
	got, state, logs := ask(t,
		env.Peer{Name: "todos", Address: downAddress(t)},
		env.Peer{Name: "time", Address: serveTime(t, weekOf(8, 37.5), nil)},
		env.Peer{Name: "pulls", Address: downAddress(t)},
	)

	want := &screen.Week{LoggedHours: 8, TargetHours: 37.5}
	if !sameWeek(got, want) || state != screen.StateOK || logs != "" {
		t.Errorf("got %+v, %q and logs %q, want %+v, ok and no logs", got, state, logs, want)
	}
}

func TestAskWithoutTimeAsksNothing(t *testing.T) {
	// A peer by another name that would answer a week, and no peers at all.
	tests := map[string][]env.Peer{
		"another peer": {{Name: "todos", Address: serveTime(t, weekOf(8, 40), nil)}},
		"no peers":     nil,
	}

	for name, peers := range tests {
		t.Run(name, func(t *testing.T) {
			got, state, logs := ask(t, peers...)

			if got != nil || state != "" || logs != "" {
				t.Errorf("got %+v, %q and logs %q, want no week, no state, no logs", got, state,
					logs)
			}
		})
	}
}

// ask returns what an Asker of peers answers, and what it logs.
func ask(t *testing.T, peers ...env.Peer) (*screen.Week, screen.State, string) {
	t.Helper()

	var logs bytes.Buffer

	got, state := week.New(peers, slog.New(slog.NewTextHandler(&logs, nil))).Ask(t.Context())

	return got, state, logs.String()
}

func weekOf(logged, target float64) *timev1.GetWeekResponse {
	return timev1.GetWeekResponse_builder{
		Week: timev1.Week_builder{LoggedHours: new(logged), TargetHours: new(target)}.Build(),
	}.Build()
}

func sameWeek(a, b *screen.Week) bool {
	return a == b || a != nil && b != nil && *a == *b
}

// answering answers GetWeek with its answer, or its error.
type answering struct {
	answer *timev1.GetWeekResponse
	err    error
}

func (a answering) GetWeek(
	context.Context, *timev1.GetWeekRequest,
) (*timev1.GetWeekResponse, error) {
	return a.answer, a.err
}

// serveTime serves the time contract on h2c, gRPC only as the services speak it to each other
// (decision 158), answering GetWeek with answer or err, and returns its address.
func serveTime(t *testing.T, answer *timev1.GetWeekResponse, err error) string {
	t.Helper()

	server := connect.NewServer()
	timev1connect.RegisterTimeServiceHandler(server, answering{answer: answer, err: err})

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)

	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	listener := httptest.NewUnstartedServer(grpcOnly(mux))
	listener.Config.Protocols = h2c
	listener.Start()
	t.Cleanup(listener.Close)

	return listener.Listener.Addr().String()
}

// grpcOnly refuses a call that isn't gRPC, though the contract's handlers answer Connect's
// protocol and gRPC-Web too.
func grpcOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/grpc" &&
			!strings.HasPrefix(contentType, "application/grpc+") {
			http.Error(w, "gRPC only", http.StatusUnsupportedMediaType)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// downAddress returns an address where nothing listens any more.
func downAddress(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	return server.Listener.Addr().String()
}
