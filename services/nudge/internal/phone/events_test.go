package phone_test

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"testing"
	"time"
)

func TestEventsTellThePageWhatChanged(t *testing.T) {
	site := newPage(t)

	req, _ := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		site.server.URL+"/nudge/events",
		nil,
	)

	resp, err := site.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q", resp.Header.Get("Content-Type"))
	}

	lines := bufio.NewScanner(resp.Body)
	seen := map[string]bool{}

	// The heartbeat shows the stream is open; a question that comes in must show as a change.
	site.askMerge(t)

	for lines.Scan() {
		seen[lines.Text()] = true
		if seen[": open"] && seen[": still here"] && seen["data: {}"] {
			break
		}
	}

	if !seen["event: changed"] || !seen["data: {}"] {
		t.Errorf("lines seen: %v", seen)
	}

	site.stop()

	for lines.Scan() { // the stream ends when the service stops
	}
}

func TestEventsByHeadAnswerTheHeadersAlone(t *testing.T) {
	site := newPage(t)

	// A HEAD that streamed would hold its connection, so the next request on it would wait until
	// the service stops.
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	var reused bool

	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(got httptrace.GotConnInfo) { reused = got.Reused },
	})

	status, header := site.request(ctx, t, http.MethodHead, "/nudge/events")
	if status != http.StatusOK || header.Get("Content-Type") != "text/event-stream" ||
		header.Get("Cache-Control") != "no-store" {
		t.Errorf("HEAD got %d %v, want 200 with the stream's headers", status, header)
	}

	status, _ = site.request(ctx, t, http.MethodGet, "/nudge/state")
	if status != http.StatusOK || !reused {
		t.Errorf("the next request got %d, on the HEAD's connection: %t; want 200 on it",
			status, reused)
	}
}

// request sends a request with no body, with ctx, and answers the response's status and headers
// once it has read the body.
func (site *page) request(
	ctx context.Context, t *testing.T, method, path string,
) (int, http.Header) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, method, site.server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := site.server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}

	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, resp.Header
}
