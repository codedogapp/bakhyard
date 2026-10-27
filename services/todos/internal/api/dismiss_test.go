package api_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestPutDismissalDismissesATicketOnce(t *testing.T) {
	router, _, _ := todos(t)

	for _, want := range []int{http.StatusCreated, http.StatusNoContent, http.StatusNoContent} {
		rec := do(t, router, http.MethodPut, "/todos/dismissals/WEB-9")
		if rec.Code != want || rec.Body.Len() != 0 {
			t.Fatalf("got %d %q, want %d and no body", rec.Code, rec.Body, want)
		}
	}

	rec := do(t, router, http.MethodGet, "/todos/dismissals")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"dismissals":["WEB-9"]}` ||
		rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("got %d %s (%s), want 200 and WEB-9 once as JSON",
			rec.Code, rec.Body, rec.Header().Get("Content-Type"))
	}
}

func TestPutDismissalIgnoresItsBody(t *testing.T) {
	router, _, _ := todos(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/todos/dismissals/WEB-9",
		strings.NewReader(`{"key":"OPS-31"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	got := do(t, router, http.MethodGet, "/todos/dismissals").Body.String()
	if rec.Code != http.StatusCreated || got != `{"dismissals":["WEB-9"]}` {
		t.Errorf("got %d and the list %s, want 201 and WEB-9 alone", rec.Code, got)
	}
}

func TestGetDismissalsListsTheKeysInByteOrder(t *testing.T) {
	router, _, _ := todos(t)

	rec := do(t, router, http.MethodGet, "/todos/dismissals")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"dismissals":[]}` {
		t.Errorf("without dismissals: got %d %s, want 200 and an empty list", rec.Code, rec.Body)
	}

	for _, key := range []string{"WEB-9", "AB-9", "A_B-1", "OPS-31", "AB-10"} {
		do(t, router, http.MethodPut, "/todos/dismissals/"+key)
	}

	want := `{"dismissals":["AB-10","AB-9","A_B-1","OPS-31","WEB-9"]}`
	if got := do(t, router, http.MethodGet, "/todos/dismissals").Body.String(); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestPutDismissalRefusesAKeyThatIsntJiras(t *testing.T) {
	const notTicket = "invalid ticket: not a Jira key such as ABC-123\n"

	router, _, _ := todos(t)

	for _, key := range []string{
		"web-9", "W-9", "WEB-09", "WEB9", "%20WEB-9", "WEB-9%0A", "WEB-%D9%A1", "WEB%2D9",
		"WEB-9%2F",
	} {
		rec := do(t, router, http.MethodPut, "/todos/dismissals/"+key)
		if rec.Code != http.StatusBadRequest || rec.Body.String() != notTicket ||
			rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Errorf("%s: got %d %q (%s), want a plain 400 %q", key, rec.Code, rec.Body,
				rec.Header().Get("Content-Type"), notTicket)
		}
	}

	if got := do(t, router, http.MethodGet, "/todos/dismissals").Body.String(); got !=
		`{"dismissals":[]}` {
		t.Errorf("got the list %s, want nothing stored", got)
	}
}

func TestTheDismissalsFailWithoutTheDatabase(t *testing.T) {
	router, pool, logs := todos(t)
	pool.Close()

	for _, test := range []struct{ method, target, log string }{
		{http.MethodPut, "/todos/dismissals/WEB-9", `msg="dismiss the ticket" err="dismiss the ticket: `},
		{http.MethodGet, "/todos/dismissals", `msg="list the dismissals" err="read the dismissals: `},
	} {
		rec := do(t, router, test.method, test.target)
		if rec.Code != http.StatusInternalServerError ||
			rec.Body.String() != "Internal Server Error\n" {
			t.Errorf("%s %s: got %d %q, want a plain 500", test.method, test.target, rec.Code,
				rec.Body)
		}

		if !strings.Contains(logs.String(), test.log) {
			t.Errorf("%s %s: got the logs %q, want the cause", test.method, test.target, logs)
		}
	}
}

func TestADismissalAllowsPut(t *testing.T) {
	router, _, _ := todos(t)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		rec := do(t, router, method, "/todos/dismissals/WEB-9")

		allow := rec.Header().Values("Allow")
		if rec.Code != http.StatusMethodNotAllowed || !slices.Equal(allow, []string{"PUT"}) {
			t.Errorf("%s: got %d, Allow %q, want 405 and PUT", method, rec.Code, allow)
		}
	}
}

// do returns router's answer to method target, without a body.
func do(t *testing.T, router http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
