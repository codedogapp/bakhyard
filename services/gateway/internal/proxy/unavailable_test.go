package proxy_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestUpstreamDown(t *testing.T) {
	for _, test := range downUpstreams {
		t.Run(test.path, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.path, nil)
			rec := forward(req, closedUpstream(t), closedUpstream(t), slog.New(slog.DiscardHandler))

			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want %q", got, "application/json")
			}

			want := `{"code":"unavailable","message":"` + test.upstream + ` is unavailable"}`
			if got := rec.Body.String(); got != want {
				t.Errorf("body = %q, want %q", got, want)
			}
		})
	}
}

func TestLogsUnavailableUpstream(t *testing.T) {
	for _, test := range downUpstreams {
		t.Run(test.path, func(t *testing.T) {
			var logs strings.Builder

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.path, nil)
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			forward(req, closedUpstream(t), closedUpstream(t), logger)

			for _, want := range []string{"level=WARN", "upstream=" + test.upstream, "err="} {
				if !strings.Contains(logs.String(), want) {
					t.Errorf("logs = %q, want them to contain %q", logs.String(), want)
				}
			}
		})
	}
}

// downUpstreams pairs a path with the upstream it goes to, for the tests where both are down.
var downUpstreams = []struct{ path, upstream string }{
	{"/api/home", "home"},
	{"/notes", "web"},
}

// closedUpstream returns the URL of a server that has shut down, so dialing it fails.
func closedUpstream(t *testing.T) *url.URL {
	t.Helper()

	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	return parse(t, srv.URL)
}
