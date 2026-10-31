package requestid_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/codedogapp/bakhyard/internal/requestid"
)

func TestTransportSendsTheRequestsIDOn(t *testing.T) {
	var got []string

	handler := requestid.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = ask(t, r.Context())
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set(requestid.Header, canonical)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !slices.Equal(got, []string{canonical}) {
		t.Errorf("the peer got %q, want %q", got, canonical)
	}
}

func TestTransportDropsTheIDOutsideARequest(t *testing.T) {
	if got := ask(t, t.Context()); len(got) != 0 {
		t.Errorf("the peer got %q, want none", got)
	}
}

// ask asks a peer through Transport, with ctx, in a request whose X-Request-Id the caller set.
// It returns the IDs that the peer got, and fails the test if the request itself changed.
func ask(t *testing.T, ctx context.Context) []string {
	t.Helper()

	received := make(chan []string, 1)
	peer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		received <- r.Header.Values(requestid.Header)
	}))
	t.Cleanup(peer.Close)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, peer.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set(requestid.Header, "the caller's")

	client := &http.Client{Transport: requestid.Transport(peer.Client().Transport)}

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	_ = res.Body.Close()

	if sent := req.Header.Values(requestid.Header); !slices.Equal(sent, []string{"the caller's"}) {
		t.Errorf("the request's X-Request-Id became %q, want it unchanged", sent)
	}

	return <-received
}
