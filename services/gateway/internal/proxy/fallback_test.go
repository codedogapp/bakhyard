package proxy_test

import (
	"net/http"
	"testing"
)

func TestForwardsOtherPathsToWeb(t *testing.T) {
	paths := []string{
		"/",
		"/home",
		"/healthz",
		"/notes/today?tab=links",
		"/assets/index-D4x9f2Kq.js",
		"/API/home",
		"/apix",
		"/api.json",
		"//api/home",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec, upstreams := route(t, http.MethodGet, path)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			upstreams.expect(t, "", "GET "+path)

			if got := rec.Body.String(); got != upstreamBody {
				t.Errorf("body = %q, want web's body %q", got, upstreamBody)
			}
		})
	}
}

func TestRejectsUnknownAPIPaths(t *testing.T) {
	paths := []string{
		"/api",
		"/api/",
		"/api/healthz",
		"/api/web",
		"/api/home/",
		"/api/home/x",
		"/api/homes",
		"/api//home",
		"/api/home/../healthz",
		"/api/ho%6De",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec, upstreams := route(t, http.MethodGet, path)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}

			upstreams.expect(t, "", "")
		})
	}
}
