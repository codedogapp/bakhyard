package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTheGatewayAnswersItsHealthCheckItself(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	router := gateway(t, web.URL)

	// No login and the loopback's Host, as process-compose's probe sends them.
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		req := httptest.NewRequestWithContext(t.Context(), method, "/healthz", nil)
		req.Host = "127.0.0.1:8080"

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
			t.Errorf("%s /healthz: got %d %q, want the health check", method, rec.Code, rec.Body)
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/healthz", nil),
	)

	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodGet {
		t.Errorf("POST /healthz: got %d, Allow %q; want 405, Allow GET",
			rec.Code, rec.Header().Get("Allow"))
	}
}

func TestTheGatewayForwardsHEADAsHEAD(t *testing.T) {
	got := make(chan string, 1)

	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got <- r.Method + " " + r.URL.Path
	}))
	t.Cleanup(web.Close)

	rec := httptest.NewRecorder()
	request := signedIn(t, http.MethodHead, "/notes")
	gateway(t, web.URL).ServeHTTP(rec, request)

	select {
	case request := <-got:
		if request != "HEAD /notes" {
			t.Errorf("web got %q, want HEAD /notes", request)
		}
	default:
		t.Error("web got nothing, want HEAD /notes")
	}
}

func TestTheGatewayLetsTheUserIn(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	router := gateway(t, web.URL)

	// Nothing listens where the test gateway's home is, so a 503 means the request reached home's
	// forwarder. An /api path that no service claims keeps its 404 behind the lock.
	for target, want := range map[string]int{
		"/api/home":    http.StatusServiceUnavailable,
		"/api/nothing": http.StatusNotFound,
		"/api":         http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, signedIn(t, http.MethodGet, target))

		if rec.Code != want {
			t.Errorf("GET %s: got %d, want %d", target, rec.Code, want)
		}
	}
}

func TestTheGatewayRefusesEveryoneElse(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	router := gateway(t, web.URL)

	for _, target := range []string{"/", "/notes", "/api/home", "/api/nothing"} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, nil))

			if rec.Code != http.StatusForbidden {
				t.Errorf("%s %s without a login: got %d, want 403", method, target, rec.Code)
			}
		}
	}
}

func TestTheGatewayNeedsTheLoginAndTheHost(t *testing.T) {
	for _, name := range []string{"BAKHYARD_TAILSCALE_LOGIN", "BAKHYARD_HOST"} {
		vars := gatewayEnv("http://127.0.0.1:2")
		delete(vars, name)

		_, err := routes(func(key string) string { return vars[key] })
		if err == nil || err.Error() != name+" is not set" {
			t.Errorf("without %s: got %v, want the error %s is not set", name, err, name)
		}
	}
}

// The login and host that the test gateway lets in.
const (
	testLogin = "user@example.com"
	testHost  = "machine.tailnet.ts.net"
)

// signedIn returns a request as tailscale serve sends it for the test gateway's user.
func signedIn(t *testing.T, method, target string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	req.Host = testHost
	req.Header.Set("Tailscale-User-Login", testLogin)

	return req
}

// gateway returns the gateway's routes, with home on a port where nothing listens and web at
// webURL.
func gateway(t *testing.T, webURL string) http.Handler {
	t.Helper()

	vars := gatewayEnv(webURL)

	router, err := routes(func(key string) string { return vars[key] })
	if err != nil {
		t.Fatal(err)
	}

	return router
}

// gatewayEnv returns the test gateway's environment, with home on a port where nothing listens
// and web at webURL.
func gatewayEnv(webURL string) map[string]string {
	return map[string]string{
		"BAKHYARD_PEERS": "home=tcp://127.0.0.1:1,web=tcp://" +
			strings.TrimPrefix(webURL, "http://"),
		"BAKHYARD_TAILSCALE_LOGIN": testLogin,
		"BAKHYARD_HOST":            testHost,
	}
}
