package phone_test

import (
	"bufio"
	"net/http"
	"testing"
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
