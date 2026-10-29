package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	timev1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1/timev1connect"
)

func TestHomeAnswersTheWeekFromTime(t *testing.T) {
	settings := map[string]string{"BAKHYARD_PEERS": "time=tcp://" + serveTime(t)}

	router, err := routes(t.Context(), func(name string) string { return settings[name] })
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/home", nil))

	var got map[string]jsontext.Value
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("got the Home %s (%v), want JSON", rec.Body, err)
	}

	week, state := string(got["week"]), string(got["weekState"])
	if want := `{"loggedHours":31.5,"targetHours":40}`; week != want || state != `"ok"` {
		t.Errorf("week = %s and weekState = %s, want %s and \"ok\"", week, state, want)
	}
}

// timeStandIn answers GetWeek with 31.5 of 40 hours.
type timeStandIn struct{}

func (timeStandIn) GetWeek(
	context.Context, *timev1.GetWeekRequest,
) (*timev1.GetWeekResponse, error) {
	return timev1.GetWeekResponse_builder{
		Week: timev1.Week_builder{LoggedHours: new(31.5), TargetHours: new(40.0)}.Build(),
	}.Build(), nil
}

// serveTime serves the stand-in over the time contract on h2c, as the services serve it, and
// returns its address.
func serveTime(t *testing.T) string {
	t.Helper()

	server := connect.NewServer()
	timev1connect.RegisterTimeServiceHandler(server, timeStandIn{})

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)

	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	listener := httptest.NewUnstartedServer(mux)
	listener.Config.Protocols = h2c
	listener.Start()
	t.Cleanup(listener.Close)

	return listener.Listener.Addr().String()
}
