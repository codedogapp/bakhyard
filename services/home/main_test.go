package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	timev1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1/timev1connect"
	"github.com/codedogapp/bakhyard/internal/requestid"
)

func TestHomeAnswersTheWeekFromTime(t *testing.T) {
	settings := map[string]string{"BAKHYARD_PEERS": "time=tcp://" + serveTime(t, new(timeStandIn))}

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

func TestHomePassesTheRequestsIDToItsPeers(t *testing.T) {
	const gatewayID = "0193f1a2-7c4e-7b8d-9e0f-1a2b3c4d5e6f"

	standIn := new(timeStandIn)
	settings := map[string]string{"BAKHYARD_PEERS": "time=tcp://" + serveTime(t, standIn)}

	router, err := routes(t.Context(), func(name string) string { return settings[name] })
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/home", nil)
	req.Header.Set(requestid.Header, gatewayID)
	requestid.Middleware(router).ServeHTTP(httptest.NewRecorder(), req)

	want := []string{"GetWeek " + gatewayID, "ListItems " + gatewayID}
	if got := standIn.calls(); !slices.Equal(got, want) {
		t.Errorf("time got %q, want %q", got, want)
	}
}

// timeStandIn answers GetWeek with 31.5 of 40 hours, and ListItems with no items. It notes the
// ID of each call's request, as its middleware took it.
type timeStandIn struct {
	mu  sync.Mutex
	ids []string
}

func (s *timeStandIn) GetWeek(
	ctx context.Context, _ *timev1.GetWeekRequest,
) (*timev1.GetWeekResponse, error) {
	s.note(ctx, "GetWeek")

	return timev1.GetWeekResponse_builder{
		Week: timev1.Week_builder{LoggedHours: new(31.5), TargetHours: new(40.0)}.Build(),
	}.Build(), nil
}

func (s *timeStandIn) ListItems(
	ctx context.Context, _ *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	s.note(ctx, "ListItems")

	return attentionv1.ListItemsResponse_builder{}.Build(), nil
}

// note notes the procedure that was called, with the ID of the request that ctx belongs to.
func (s *timeStandIn) note(ctx context.Context, procedure string) {
	id, _ := requestid.FromContext(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ids = append(s.ids, procedure+" "+id)
}

// calls returns the calls that note noted, sorted.
func (s *timeStandIn) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Sorted(slices.Values(s.ids))
}

// serveTime serves standIn over the time and attention contracts on h2c, behind the request IDs'
// middleware, as the services serve them, and returns its address.
func serveTime(t *testing.T, standIn *timeStandIn) string {
	t.Helper()

	server := connect.NewServer()
	timev1connect.RegisterTimeServiceHandler(server, standIn)
	attentionv1connect.RegisterAttentionServiceHandler(server, standIn)

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)

	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	listener := httptest.NewUnstartedServer(requestid.Middleware(mux))
	listener.Config.Protocols = h2c
	listener.Start()
	t.Cleanup(listener.Close)

	return listener.Listener.Addr().String()
}
