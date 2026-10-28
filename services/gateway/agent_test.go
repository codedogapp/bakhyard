package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentsReachNudgeWithTheTailscaleLoginAlone(t *testing.T) {
	reached := make(chan string, 4)

	nudge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached <- r.Method + " " + r.URL.Path + " cookie=" + r.Header.Get("Cookie")

		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(nudge.Close)

	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	vars, items := gatewayEnv(t, web.URL)
	vars["BAKHYARD_PEERS"] = strings.Replace(vars["BAKHYARD_PEERS"],
		"nudge=tcp://127.0.0.1:1", "nudge=tcp://"+strings.TrimPrefix(nudge.URL, "http://"), 1)
	router, token := gatewayFrom(t, vars, items)

	// The path reaches nudge as it is, with the agent's login and no session, and a cookie that
	// the browser's session left is not passed on.
	for _, test := range []struct {
		method, target string
		session        bool
		want           string
	}{
		{http.MethodPost, "/agent/questions", false, "POST /agent/questions cookie="},
		{http.MethodDelete, "/agent/questions/abc", false, "DELETE /agent/questions/abc cookie="},
		{http.MethodPost, "/agent/sessions", true, "POST /agent/sessions cookie="},
		{http.MethodPut, "/agent/sessions/abc/note", false, "PUT /agent/sessions/abc/note cookie="},
	} {
		req := signedIn(t, test.method, test.target)
		if test.session {
			req = withSession(req, token)
		}

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("%s %s: got %d, want nudge's 202", test.method, test.target, rec.Code)
		}

		if got := <-reached; got != test.want {
			t.Errorf("nudge got %q, want %q", got, test.want)
		}
	}
}
