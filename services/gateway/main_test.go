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

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/healthz", nil))

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
	request := httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/notes", nil)
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

// gateway returns the gateway's routes, with home on a port where nothing listens and web at
// webURL.
func gateway(t *testing.T, webURL string) http.Handler {
	t.Helper()

	peers := "home=tcp://127.0.0.1:1,web=tcp://" + strings.TrimPrefix(webURL, "http://")
	getenv := func(name string) string {
		if name == "BAKHYARD_PEERS" {
			return peers
		}

		return ""
	}

	router, err := routes(getenv)
	if err != nil {
		t.Fatal(err)
	}

	return router
}
