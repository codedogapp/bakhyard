package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTheGatewaySignsTheUserOutOnThisDeviceOnly(t *testing.T) {
	vars, items := gatewayEnv(t, "http://127.0.0.1:2")
	router, token := gatewayFrom(t, vars, items)
	// Another device's session, in the same database.
	_, other := gatewayFrom(t, vars, items)

	// The app's form is same-origin.
	req := withSession(signedIn(t, http.MethodPost, "/auth/logout"), token)
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" ||
		!strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("got %d to %q with the cookie %q; want 303 to / that drops the cookie",
			rec.Code, rec.Header().Get("Location"), rec.Header().Get("Set-Cookie"))
	}

	if code := apiStatus(t, router, token); code != http.StatusUnauthorized {
		t.Errorf("this device's session: got %d on the API, want 401", code)
	}

	if code := apiStatus(t, router, other); code != http.StatusServiceUnavailable {
		t.Errorf("the other device's session: got %d on the API, want it to live on", code)
	}
}

func TestTheSignOutRouteServesPOSTOnly(t *testing.T) {
	router, token := gateway(t, "http://127.0.0.1:2")

	// No handler runs for these: chi answers 405 before the Tailscale lock, which comes first
	// for POST, the route's method.
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, withSession(signedIn(t, method, "/auth/logout"), token))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /auth/logout: got %d, want 405", method, rec.Code)
		}
	}

	for _, req := range []*http.Request{
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", nil),
		withSession(
			httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", nil), token),
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("POST /auth/logout without a login: got %d, want 403", rec.Code)
		}
	}

	if code := apiStatus(t, router, token); code != http.StatusServiceUnavailable {
		t.Errorf("got %d on the API, want the session to live on", code)
	}
}

func TestTheGatewayRefusesWhatOtherSitesSend(t *testing.T) {
	reached := make(chan string, 10)

	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		reached <- r.Method + " " + r.URL.Path
	}))
	t.Cleanup(web.Close)

	router, token := gateway(t, web.URL)

	// Another site's page can make the browser send its request here, with the session cookie.
	// The tailnet's other machines count as the same site, so only the same origin passes.
	for _, test := range []struct {
		method, target, header, value string
		want                          int
	}{
		{http.MethodPost, "/auth/logout", "Sec-Fetch-Site", "cross-site", http.StatusForbidden},
		{http.MethodPost, "/auth/logout", "Sec-Fetch-Site", "same-site", http.StatusForbidden},
		{http.MethodPost, "/auth/logout", "Origin", "https://elsewhere.example", http.StatusForbidden},
		{http.MethodPost, "/api/home", "Sec-Fetch-Site", "cross-site", http.StatusForbidden},
		{http.MethodDelete, "/notes", "Sec-Fetch-Site", "same-site", http.StatusForbidden},
		{http.MethodGet, "/notes", "Sec-Fetch-Site", "cross-site", http.StatusOK},
		{http.MethodHead, "/notes", "Origin", "https://elsewhere.example", http.StatusOK},
		{http.MethodOptions, "/notes", "Sec-Fetch-Site", "cross-site", http.StatusOK},
	} {
		req := withSession(signedIn(t, test.method, test.target), token)
		req.Header.Set(test.header, test.value)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != test.want {
			t.Errorf("%s %s with %s %s: got %d, want %d",
				test.method, test.target, test.header, test.value, rec.Code, test.want)
		}
	}

	close(reached)

	var got []string
	for request := range reached {
		got = append(got, request)
	}

	if strings.Join(got, ", ") != "GET /notes, HEAD /notes, OPTIONS /notes" {
		t.Errorf("web got %q, want only the safe requests", got)
	}

	if code := apiStatus(t, router, token); code != http.StatusServiceUnavailable {
		t.Errorf("got %d on the API, want the session to live on", code)
	}
}

func TestTheGatewayLetsSameOriginSignOutsThrough(t *testing.T) {
	router, _ := gateway(t, "http://127.0.0.1:2")

	// The browser sends Sec-Fetch-Site, or else an Origin; curl sends neither.
	for header, value := range map[string]string{
		"Sec-Fetch-Site": "same-origin",
		"Origin":         "https://" + testHost,
		"":               "",
	} {
		req := signedIn(t, http.MethodPost, "/auth/logout")
		if header != "" {
			req.Header.Set(header, value)
		}

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Errorf("POST /auth/logout with %q %q: got %d, want 303", header, value, rec.Code)
		}
	}
}

// apiStatus returns the status of GET /api/home with token's session: 401 without a live
// session, and 503 with one, as nothing listens where the test gateway's home is.
func apiStatus(t *testing.T, router http.Handler, token string) int {
	t.Helper()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, withSession(signedIn(t, http.MethodGet, "/api/home"), token))

	return rec.Code
}
