package requestid_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"uuid"

	"github.com/codedogapp/bakhyard/internal/requestid"
)

const canonical = "0193f1a2-7c4e-7b8d-9e0f-1a2b3c4d5e6f"

func TestMiddlewareKeepsAUUID(t *testing.T) {
	for _, sent := range []string{
		canonical,
		"0193F1A2-7C4E-7B8D-9E0F-1A2B3C4D5E6F",
		"{0193f1a2-7c4e-7b8d-9e0f-1a2b3c4d5e6f}",
		"urn:uuid:0193f1a2-7c4e-7b8d-9e0f-1a2b3c4d5e6f",
		"0193f1a27c4e7b8d9e0f1a2b3c4d5e6f",
	} {
		t.Run(sent, func(t *testing.T) {
			got, answered := serve(t, sent)
			if got != canonical || !slices.Equal(answered, []string{canonical}) {
				t.Errorf("got %q and answered %q, want %q", got, answered, canonical)
			}
		})
	}
}

func TestMiddlewareMakesAUUIDv7Otherwise(t *testing.T) {
	for _, sent := range []string{"", "abc", canonical + "0", `"); DROP TABLE logs; --`} {
		t.Run(sent, func(t *testing.T) {
			got, answered := serve(t, sent)

			id, err := uuid.Parse(got)
			if err != nil || id.String() != got {
				t.Fatalf("got %q, want a UUID in its canonical form", got)
			}

			if version := id[6] >> 4; version != 7 {
				t.Errorf("got a UUID of version %d, want 7", version)
			}

			if !slices.Equal(answered, []string{got}) {
				t.Errorf("answered %q, want %q", answered, got)
			}
		})
	}
}

func TestMiddlewareGivesEachRequestItsOwnID(t *testing.T) {
	first, _ := serve(t, "")
	second, _ := serve(t, "")

	if first == second {
		t.Errorf("two requests got the same ID, %q", first)
	}
}

func TestPassSendsTheRequestsIDOn(t *testing.T) {
	header := http.Header{requestid.Header: {"the client's"}}
	handler := requestid.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		requestid.Pass(r.Context(), header)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set(requestid.Header, canonical)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got := header.Values(requestid.Header); !slices.Equal(got, []string{canonical}) {
		t.Errorf("got %q, want %q", got, canonical)
	}
}

func TestPassDeletesTheIDOutsideARequest(t *testing.T) {
	header := http.Header{requestid.Header: {canonical}}
	requestid.Pass(t.Context(), header)

	if got := header.Values(requestid.Header); len(got) != 0 {
		t.Errorf("got %q, want none", got)
	}
}

func TestFromContextOutsideARequest(t *testing.T) {
	if id, ok := requestid.FromContext(t.Context()); ok {
		t.Errorf("got %q, want none", id)
	}
}

// serve sends a request whose X-Request-Id is sent, or that has none when sent is "", through the
// middleware. It returns the ID that the handler found and the IDs that the answer carries.
func serve(t *testing.T, sent string) (got string, answered []string) {
	t.Helper()

	handler := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := requestid.FromContext(r.Context())
		if !ok {
			t.Error("the handler's context has no ID")
		}

		got = id

		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	if sent != "" {
		req.Header.Set(requestid.Header, sent)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return got, rec.Result().Header.Values(requestid.Header)
}
