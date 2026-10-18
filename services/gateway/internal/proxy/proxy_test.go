package proxy_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
)

const upstreamBody = `{"items":[]}`

func TestForwardsHome(t *testing.T) {
	target, seen := upstream(t)

	rec := serve(t, target, http.MethodGet, "/api/home?week=1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := <-seen; got != "GET /home?week=1" {
		t.Errorf("upstream got %q, want %q", got, "GET /home?week=1")
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	if got := rec.Body.String(); got != upstreamBody {
		t.Errorf("body = %q, want %q", got, upstreamBody)
	}
}

func TestForwardsEveryMethod(t *testing.T) {
	methods := []string{
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			target, seen := upstream(t)

			rec := serve(t, target, method, "/api/home")

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			if got, want := <-seen, method+" /home"; got != want {
				t.Errorf("upstream got %q, want %q", got, want)
			}
		})
	}
}

func TestRejectsOtherPaths(t *testing.T) {
	paths := []string{
		"/",
		"/home",
		"/healthz",
		"/api",
		"/api/",
		"/api/healthz",
		"/api/home/",
		"/api/home/x",
		"/api/homes",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			target, seen := upstream(t)

			rec := serve(t, target, http.MethodGet, path)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}

			if len(seen) != 0 {
				t.Errorf("upstream got %q, want no request", <-seen)
			}
		})
	}
}

func TestDropsClientForwardingHeaders(t *testing.T) {
	headers := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
	}))
	t.Cleanup(srv.Close)

	names := []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/home", nil)

	for _, name := range names {
		req.Header.Set(name, "spoofed")
	}

	rec := forward(parse(t, srv.URL), req, slog.New(slog.DiscardHandler))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	got := <-headers
	for _, name := range names {
		if values := got.Values(name); len(values) != 0 {
			t.Errorf("upstream got %s: %q, want none", name, values)
		}
	}
}

func TestLogsUnavailableUpstream(t *testing.T) {
	var logs strings.Builder

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/home", nil)
	forward(closedUpstream(t), req, slog.New(slog.NewTextHandler(&logs, nil)))

	for _, want := range []string{"level=WARN", "upstream=home", "err="} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs = %q, want them to contain %q", logs.String(), want)
		}
	}
}

func TestUpstreamDown(t *testing.T) {
	rec := serve(t, closedUpstream(t), http.MethodGet, "/api/home")

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	want := `{"code":"unavailable","message":"home is unavailable"}`
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// upstream starts a stand-in for home that reports each request as "<method> <request URI>".
func upstream(t *testing.T) (*url.URL, <-chan string) {
	t.Helper()

	seen := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Method + " " + r.URL.RequestURI()

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, upstreamBody)
	}))
	t.Cleanup(srv.Close)

	return parse(t, srv.URL), seen
}

// closedUpstream returns the URL of a server that has shut down, so dialing it fails.
func closedUpstream(t *testing.T) *url.URL {
	t.Helper()

	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	return parse(t, srv.URL)
}

func serve(t *testing.T, target *url.URL, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)

	return forward(target, req, slog.New(slog.DiscardHandler))
}

// forward sends req through a gateway that has home registered at target.
func forward(target *url.URL, req *http.Request, logger *slog.Logger) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	proxy.Register(mux, "home", target, logger)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

func parse(t *testing.T, raw string) *url.URL {
	t.Helper()

	target, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}

	return target
}
