package main

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTheGatewayPassesAPlansEventsOnAsTheyCome(t *testing.T) {
	ended := make(chan struct{})
	planner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: status\ndata: {\"working\":true}\n\n")
		_ = http.NewResponseController(w).Flush()

		select {
		case <-ended:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(planner.Close)

	vars, items := gatewayEnv(t, "http://127.0.0.1:1")
	vars["BAKHYARD_PEERS"] = strings.Replace(vars["BAKHYARD_PEERS"],
		"planner=tcp://127.0.0.1:1", "planner=tcp://"+strings.TrimPrefix(planner.URL, "http://"), 1)
	router, token := gatewayFrom(t, vars, items)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	// The stream ends first, so that the servers can close.
	t.Cleanup(func() { close(ended) })

	// The stream stays open, so a gateway that held the event back until it ends would answer
	// nothing in time.
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, srv.URL+"/api/planner/plans/p1/events", nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	req.Host = testHost
	req.Header.Set("Tailscale-User-Login", testLogin)

	resp, err := srv.Client().Do(withSession(req, token))
	if err != nil {
		t.Fatalf("GET a plan's events: %v, want the planner's event at once", err)
	}

	defer func() { _ = resp.Body.Close() }()

	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil || line != "event: status\n" {
		t.Errorf("the stream began with %q, %v; want the planner's event", line, err)
	}
}
