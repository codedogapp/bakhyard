package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestTheGatewayForwardsEachAPIToItsService(t *testing.T) {
	reached := make(chan string, 1)
	service := func(name string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reached <- name + " got " + r.Method + " " + r.URL.Path

			w.WriteHeader(http.StatusAccepted)
		}))
		t.Cleanup(srv.Close)

		return srv
	}
	home, notes := service("home"), service("notes")
	pulls, todos := service("pulls"), service("todos")
	notify, nudge := service("notify"), service("nudge")
	github, timeService := service("github"), service("time")

	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	vars, items := gatewayEnv(t, web.URL)
	vars["BAKHYARD_PEERS"] = "github=tcp://" + strings.TrimPrefix(github.URL, "http://") +
		",home=tcp://" + strings.TrimPrefix(home.URL, "http://") +
		",notes=tcp://" + strings.TrimPrefix(notes.URL, "http://") +
		",notify=tcp://" + strings.TrimPrefix(notify.URL, "http://") +
		",nudge=tcp://" + strings.TrimPrefix(nudge.URL, "http://") +
		",pulls=tcp://" + strings.TrimPrefix(pulls.URL, "http://") +
		",time=tcp://" + strings.TrimPrefix(timeService.URL, "http://") +
		",todos=tcp://" + strings.TrimPrefix(todos.URL, "http://") +
		",web=tcp://" + strings.TrimPrefix(web.URL, "http://")
	router, token := gatewayFrom(t, vars, items)

	// Whatever the method or the path under /api/<service>, the service answers for itself
	// (decisions 41 and 92).
	for _, test := range []struct{ method, target, want string }{
		{http.MethodGet, "/api/github/mine", "github got GET /github/mine"},
		{http.MethodGet, "/api/home", "home got GET /home"},
		{http.MethodGet, "/api/notes", "notes got GET /notes"},
		{http.MethodPost, "/api/notes", "notes got POST /notes"},
		{
			http.MethodGet, "/api/notes/files/Daily/To%20do.md",
			"notes got GET /notes/files/Daily/To do.md",
		},
		{http.MethodGet, "/api/pulls/mine", "pulls got GET /pulls/mine"},
		{http.MethodPost, "/api/pulls", "pulls got POST /pulls"},
		{http.MethodGet, "/api/time/report", "time got GET /time/report"},
		{http.MethodGet, "/api/todos", "todos got GET /todos"},
		{http.MethodPost, "/api/todos", "todos got POST /todos"},
		{http.MethodGet, "/api/notify/key", "notify got GET /notify/key"},
		{http.MethodGet, "/api/nudge/state", "nudge got GET /nudge/state"},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, withSession(signedIn(t, test.method, test.target), token))

		if rec.Code != http.StatusAccepted {
			t.Errorf("%s %s: got %d, want the service's 202", test.method, test.target, rec.Code)
		}

		select {
		case got := <-reached:
			if got != test.want {
				t.Errorf("%s %s: %s, want %s", test.method, test.target, got, test.want)
			}
		default:
			t.Errorf("%s %s: no service got it, want %s", test.method, test.target, test.want)
		}
	}
}

// peerNames returns the names of the gateway's peers, each of which BAKHYARD_PEERS must name.
func peerNames() []string {
	return []string{"github", "home", "notes", "notify", "nudge", "pulls", "time", "todos", "web"}
}

// withPeers returns BAKHYARD_PEERS with every peer but those in lacking.
func withPeers(lacking ...string) string {
	var peers []string

	for _, name := range peerNames() {
		if !slices.Contains(lacking, name) {
			peers = append(peers, name+"=tcp://127.0.0.1:1")
		}
	}

	return strings.Join(peers, ",")
}

func TestTheGatewayNeedsItsPeers(t *testing.T) {
	// The error names every peer that BAKHYARD_PEERS lacks, one per line.
	for peers, missing := range map[string][]string{
		withPeers("notes"):           {"notes"},
		withPeers("pulls"):           {"pulls"},
		withPeers("notify", "nudge"): {"notify", "nudge"},
		withPeers("pulls", "todos"):  {"pulls", "todos"},
		withPeers("time"):            {"time"},
		withPeers("github"):          {"github"},
		"":                           peerNames(),
	} {
		vars, items := gatewayEnv(t, "http://127.0.0.1:2")
		vars["BAKHYARD_PEERS"] = peers

		var lines []string
		for _, name := range missing {
			lines = append(lines, `BAKHYARD_PEERS has no peer "`+name+`"`)
		}

		_, err := routes(func(key string) string { return vars[key] }, secretsWith(items))
		if want := strings.Join(lines, "\n"); err == nil || err.Error() != want {
			t.Errorf("with the peers %q: got %v, want the error %q", peers, err, want)
		}
	}
}
