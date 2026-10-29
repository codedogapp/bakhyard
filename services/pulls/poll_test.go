package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPullsListsMyPullsInEachWorkspace(t *testing.T) {
	address, _ := bitbucketServer(t)

	rec := getPulls(t.Context(), t, address, "/pulls/mine")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"repo":"acme/web"`) ||
		!strings.Contains(rec.Body.String(), `"repo":"labs/web"`) {
		t.Errorf("got %d %q, want a pull request in acme and in labs", rec.Code, rec.Body)
	}
}

func TestPullsListsThePullsThatWaitForMyReview(t *testing.T) {
	address, _ := bitbucketServer(t)

	rec := getPulls(t.Context(), t, address, "/pulls/reviews")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"repo":"acme/api"`) ||
		!strings.Contains(rec.Body.String(), `"repo":"labs/app"`) {
		t.Errorf("got %d %q, want a pull request in acme and in labs", rec.Code, rec.Body)
	}
}

func TestPullsStopsListingWhenItStops(t *testing.T) {
	stopped, stop := context.WithCancel(t.Context())
	stop()

	// Nothing listens on port 1, so a listing that went ahead would fail.
	for _, target := range []string{"/pulls/mine", "/pulls/reviews"} {
		rec := getPulls(stopped, t, "http://127.0.0.1:1/2.0", target)
		if rec.Code != http.StatusOK || rec.Body.String() != "{}" {
			t.Errorf("%s: got %d %s, want 200 {}: no listing, and no failure", target, rec.Code,
				rec.Body)
		}
	}
}

// getPulls returns the answer to a GET of target of pulls with Bitbucket's API at address, whose
// routes run until ctx ends.
func getPulls(
	ctx context.Context, t *testing.T, address, target string,
) *httptest.ResponseRecorder {
	t.Helper()

	router, err := routes(ctx, envOf(bitbucketSettings(address)),
		secretsWith(map[string]string{"bitbucket-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
