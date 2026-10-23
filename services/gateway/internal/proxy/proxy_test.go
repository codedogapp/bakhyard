package proxy_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
)

const upstreamBody = `{"items":[]}`

func TestForwardsHome(t *testing.T) {
	rec, upstreams := route(t, http.MethodGet, "/api/home?week=1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	upstreams.expect(t, "GET /home?week=1", "")

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
		t.Run(method+" home", func(t *testing.T) {
			rec, upstreams := route(t, method, "/api/home")

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			upstreams.expect(t, method+" /home", "")
		})

		t.Run(method+" web", func(t *testing.T) {
			rec, upstreams := route(t, method, "/notes")

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			upstreams.expect(t, "", method+" /notes")
		})
	}
}

func TestRejectsNonStandardMethods(t *testing.T) {
	for _, path := range []string{"/api/home", "/"} {
		t.Run(path, func(t *testing.T) {
			rec, upstreams := route(t, "PURGE", path)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}

			upstreams.expect(t, "", "")
		})
	}
}

func TestDropsClientForwardingHeaders(t *testing.T) {
	names := []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"}

	for _, path := range []string{"/api/home", "/"} {
		t.Run(path, func(t *testing.T) {
			target, headers := headerSink(t)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)

			for _, name := range names {
				req.Header.Set(name, "spoofed")
			}

			rec := forward(req, target, target, slog.New(slog.DiscardHandler))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			got := <-headers
			for _, name := range names {
				if values := got.Values(name); len(values) != 0 {
					t.Errorf("upstream got %s: %q, want none", name, values)
				}
			}
		})
	}
}

// received holds the requests that the home and web stand-ins got.
type received struct{ home, web <-chan string }

// expect fails t unless home and web got exactly the requests given; "" means none.
func (upstreams received) expect(t *testing.T, home, web string) {
	t.Helper()

	expectRequest(t, "home", upstreams.home, home)
	expectRequest(t, "web", upstreams.web, web)
}

func expectRequest(t *testing.T, name string, seen <-chan string, want string) {
	t.Helper()

	// The stand-in reports a request before it answers, so the report is already here.
	select {
	case req := <-seen:
		if req != want {
			t.Errorf("%s got %q, want %q", name, req, want)
		}
	default:
		if want != "" {
			t.Errorf("%s got no request, want %q", name, want)
		}
	}
}

// route sends a request through a gateway with stand-ins for home and web.
func route(t *testing.T, method, path string) (*httptest.ResponseRecorder, received) {
	t.Helper()

	home, homeSeen := upstream(t)
	web, webSeen := upstream(t)
	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	rec := forward(req, home, web, slog.New(slog.DiscardHandler))

	return rec, received{home: homeSeen, web: webSeen}
}

// forward sends req through a gateway that has home and web registered at their targets.
func forward(
	req *http.Request,
	home, web *url.URL,
	logger *slog.Logger,
) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	proxy.Register(router, "home", home, logger)
	proxy.RegisterUnclaimed(router)
	proxy.RegisterFallback(router, "web", web, logger)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

// upstream starts a stand-in service that reports each request as "<method> <request URI>".
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

// headerSink starts a stand-in service that reports the headers of each request.
func headerSink(t *testing.T) (*url.URL, <-chan http.Header) {
	t.Helper()

	headers := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
	}))
	t.Cleanup(srv.Close)

	return parse(t, srv.URL), headers
}

func parse(t *testing.T, raw string) *url.URL {
	t.Helper()

	target, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}

	return target
}
