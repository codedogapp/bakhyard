package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTheGatewayForwardsTheNotes(t *testing.T) {
	reached := make(chan string, 1)

	notes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached <- r.Method + " " + r.URL.Path

		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(notes.Close)

	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	vars, items := gatewayEnv(t, web.URL)
	vars["BAKHYARD_PEERS"] = "home=tcp://127.0.0.1:1" +
		",notes=tcp://" + strings.TrimPrefix(notes.URL, "http://") +
		",web=tcp://" + strings.TrimPrefix(web.URL, "http://")
	router, token := gatewayFrom(t, vars, items)

	// Whatever the method, notes answers for itself (decision 41).
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, withSession(signedIn(t, method, "/api/notes"), token))

		if rec.Code != http.StatusAccepted {
			t.Errorf("%s /api/notes: got %d, want notes' 202", method, rec.Code)
		}

		select {
		case got := <-reached:
			if got != method+" /notes" {
				t.Errorf("%s /api/notes: notes got %s, want %s /notes", method, got, method)
			}
		default:
			t.Errorf("%s /api/notes: notes got nothing", method)
		}
	}
}

func TestTheGatewayNeedsItsPeers(t *testing.T) {
	for peers, want := range map[string]string{
		"home=tcp://127.0.0.1:1,web=tcp://127.0.0.1:2": `BAKHYARD_PEERS has no peer "notes"`,
		"": `BAKHYARD_PEERS has no peer "home"` + "\n" +
			`BAKHYARD_PEERS has no peer "notes"` + "\n" + `BAKHYARD_PEERS has no peer "web"`,
	} {
		vars, items := gatewayEnv(t, "http://127.0.0.1:2")
		vars["BAKHYARD_PEERS"] = peers

		_, err := routes(func(key string) string { return vars[key] }, keychainWith(items))
		if err == nil || err.Error() != want {
			t.Errorf("with the peers %q: got %v, want the error %q", peers, err, want)
		}
	}
}
