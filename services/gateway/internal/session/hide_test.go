package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
)

func TestHideKeepsTheSessionCookieFromWhatComesNext(t *testing.T) {
	req := request(t, http.MethodGet, "/notes", liveToken)
	req.Header.Add("Cookie", "theme=dark")

	got := passedOn(t, req)

	if cookies := got.Header.Values("Cookie"); len(cookies) != 1 || cookies[0] != "theme=dark" {
		t.Errorf("the request passed on has the cookies %q, want only theme=dark", cookies)
	}

	if got.Method != http.MethodGet || got.URL.Path != "/notes" {
		t.Errorf("the request passed on is %s %s, want GET /notes", got.Method, got.URL.Path)
	}

	if _, err := req.Cookie("__Host-session"); err != nil {
		t.Errorf("the request Hide got lost its session cookie: %v", err)
	}
}

func TestHideLeavesNoCookieHeaderWithoutOtherCookies(t *testing.T) {
	for _, req := range []*http.Request{
		request(t, http.MethodGet, "/", ""),
		request(t, http.MethodGet, "/", liveToken),
	} {
		if cookies := passedOn(t, req).Header.Get("Cookie"); cookies != "" {
			t.Errorf("the request passed on has the cookies %q, want none", cookies)
		}
	}
}

// passedOn returns the request that Hide passes on for req.
func passedOn(t *testing.T, req *http.Request) *http.Request {
	t.Helper()

	var got *http.Request

	session.Hide(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r
	})).ServeHTTP(httptest.NewRecorder(), req)

	if got == nil {
		t.Fatal("Hide passed nothing on")
	}

	return got
}
