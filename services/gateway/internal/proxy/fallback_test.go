package proxy_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
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
		"/api/homes",
		"/api//home",
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

func TestLeavesTheRoutersOwnPathsToIt(t *testing.T) {
	web, webSeen := upstream(t)
	router := chi.NewRouter()
	router.Get(
		"/own",
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
	)
	proxy.RegisterFallback(router, "web", web, slog.New(slog.DiscardHandler))

	tests := []struct {
		method string
		code   int
		allow  string
	}{
		{method: http.MethodGet, code: http.StatusOK},
		{method: http.MethodPost, code: http.StatusMethodNotAllowed, allow: http.MethodGet},
		{method: http.MethodDelete, code: http.StatusMethodNotAllowed, allow: http.MethodGet},
	}

	for _, test := range tests {
		t.Run(test.method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(
				rec,
				httptest.NewRequestWithContext(t.Context(), test.method, "/own", nil),
			)

			if rec.Code != test.code || rec.Header().Get("Allow") != test.allow {
				t.Errorf("got %d, Allow %q; want %d, Allow %q",
					rec.Code, rec.Header().Get("Allow"), test.code, test.allow)
			}

			expectRequest(t, "web", webSeen, "")
		})
	}
}
