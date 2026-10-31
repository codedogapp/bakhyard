package proxy_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/requestid"
	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
)

const clientID = "0193f1a2-7c4e-7b8d-9e0f-1a2b3c4d5e6f"

func TestPassesTheRequestsIDOn(t *testing.T) {
	for _, test := range []struct {
		name, sent string
		keeps      bool
	}{
		{name: "a UUID", sent: clientID, keeps: true},
		{name: "another value", sent: "not a UUID"},
		{name: "none"},
	} {
		t.Run(test.name, func(t *testing.T) {
			// The gateway and the stand-in service each run behind requestid's middleware, as
			// every service does (internal/service).
			target, seen := idSink(t)
			router := chi.NewRouter()
			proxy.Register(router, "home", target, slog.New(slog.DiscardHandler))

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/home", nil)
			if test.sent != "" {
				req.Header.Set(requestid.Header, test.sent)
			}

			rec := httptest.NewRecorder()
			requestid.Middleware(router).ServeHTTP(rec, req)

			answered := rec.Result().Header.Values(requestid.Header)
			if len(answered) != 1 {
				t.Fatalf("the answer carries %q, want one ID", answered)
			}

			expectRequest(t, "home", seen, answered[0])

			if (answered[0] == clientID) != test.keeps {
				t.Errorf("the gateway answered %q for the client's %q", answered[0], test.sent)
			}
		})
	}
}

// idSink starts a stand-in service behind requestid's middleware that reports each request's ID.
func idSink(t *testing.T) (*url.URL, <-chan string) {
	t.Helper()

	ids := make(chan string, 1)
	srv := httptest.NewServer(requestid.Middleware(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			id, _ := requestid.FromContext(r.Context())
			ids <- id
		},
	)))
	t.Cleanup(srv.Close)

	return parse(t, srv.URL), ids
}
