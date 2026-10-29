package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTheGatewayNeedsASessionForTheAPIOnly(t *testing.T) {
	reached := make(chan string, 10)

	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		reached <- r.Method + " " + r.URL.Path
	}))
	t.Cleanup(web.Close)

	router, _ := gateway(t, web.URL)

	// The app's files load without a session, and the app shows its sign-in screen on the API's
	// 401 (decision 86). An /api path gets the 401 whether a service claims it or not.
	for _, test := range []struct {
		method, target string
		want           int
	}{
		{http.MethodGet, "/", http.StatusOK},
		{http.MethodHead, "/notes", http.StatusOK},
		{http.MethodGet, "/auth/other", http.StatusOK},
		{http.MethodGet, "/api/home", http.StatusUnauthorized},
		{http.MethodPost, "/api/home", http.StatusUnauthorized},
		{http.MethodGet, "/api/notes", http.StatusUnauthorized},
		{http.MethodGet, "/api/notes/files/Inbox.md", http.StatusUnauthorized},
		{http.MethodGet, "/api/pulls/mine", http.StatusUnauthorized},
		{http.MethodGet, "/api/time/report", http.StatusUnauthorized},
		{http.MethodGet, "/api/todos", http.StatusUnauthorized},
		{http.MethodGet, "/api/nudge/state", http.StatusUnauthorized},
		{http.MethodPost, "/api/notify/subscriptions", http.StatusUnauthorized},
		{http.MethodGet, "/api/nothing", http.StatusUnauthorized},
		{http.MethodGet, "/api", http.StatusUnauthorized},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, signedIn(t, test.method, test.target))

		if rec.Code != test.want || rec.Header().Get("Location") != "" {
			t.Errorf("%s %s without a session: got %d to %q, want %d",
				test.method, test.target, rec.Code, rec.Header().Get("Location"), test.want)
		}
	}

	close(reached)

	var got []string
	for request := range reached {
		got = append(got, request)
	}

	if strings.Join(got, ", ") != "GET /, HEAD /notes, GET /auth/other" {
		t.Errorf("web got %q, want only the app's paths", got)
	}
}

func TestTheGatewayKeepsTheSessionFromTheServices(t *testing.T) {
	cookies := make(chan string, 1)
	service := func(name string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			cookies <- name + " got " + r.Header.Get("Cookie")
		}))
		t.Cleanup(srv.Close)

		return srv
	}
	home, pulls, todos, web := service("home"), service("pulls"), service("todos"), service("web")
	timeService := service("time")

	vars, items := gatewayEnv(t, web.URL)
	vars["BAKHYARD_PEERS"] = "home=tcp://" + strings.TrimPrefix(home.URL, "http://") +
		",notes=tcp://127.0.0.1:1,notify=tcp://127.0.0.1:1,nudge=tcp://127.0.0.1:1" +
		",pulls=tcp://" + strings.TrimPrefix(pulls.URL, "http://") +
		",time=tcp://" + strings.TrimPrefix(timeService.URL, "http://") +
		",todos=tcp://" + strings.TrimPrefix(todos.URL, "http://") +
		",web=tcp://" + strings.TrimPrefix(web.URL, "http://")
	router, token := gatewayFrom(t, vars, items)

	// Only the API renews the session; neither the services behind the session lock nor the one
	// behind the Tailscale lock alone sees it.
	for _, test := range []struct {
		target, service string
		renewed         bool
	}{
		{"/api/home", "home", true},
		{"/api/pulls/mine", "pulls", true},
		{"/api/time/report", "time", true},
		{"/api/todos", "todos", true},
		{"/notes", "web", false},
	} {
		req := withSession(signedIn(t, http.MethodGet, test.target), token)
		req.Header.Add("Cookie", "theme=dark")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		renewed := rec.Header().Get("Set-Cookie") != ""
		if rec.Code != http.StatusOK || renewed != test.renewed {
			t.Errorf("GET %s: got %d, renewed %t; want 200, renewed %t",
				test.target, rec.Code, renewed, test.renewed)
		}

		select {
		case got := <-cookies:
			if got != test.service+" got theme=dark" {
				t.Errorf("GET %s: %s, want %s got only theme=dark", test.target, got, test.service)
			}
		default:
			t.Errorf("GET %s: no service got it, want %s", test.target, test.service)
		}
	}
}
