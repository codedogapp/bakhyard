package api_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestPatchTodoChangesATodo(t *testing.T) {
	router, _, _ := todos(t)
	post(t, router, `{"title":"Call home"}`)
	post(t, router, `{"title":"Buy milk"}`)

	for _, test := range []struct{ body, want string }{
		{`{"done":true}`, `{"id":2,"title":"Buy milk","done":true}`},
		{`{"title":"  Buy <oat> milk "}`, `{"id":2,"title":"Buy <oat> milk","done":true}`},
		{`{"title":null,"done":null}`, `{"id":2,"title":"Buy <oat> milk","done":true}`},
		{`{}`, `{"id":2,"title":"Buy <oat> milk","done":true}`},
		{`null`, `{"id":2,"title":"Buy <oat> milk","done":true}`},
		{`{"title":"Buy milk","done":false}`, `{"id":2,"title":"Buy milk","done":false}`},
	} {
		rec := patch(t, router, "/todos/2", test.body)
		if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
			rec.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("%s: got %d %s (%s), want 200 %s as JSON", test.body, rec.Code, rec.Body,
				rec.Header().Get("Content-Type"), test.want)
		}

		want := `{"todos":[` + test.want + `,{"id":1,"title":"Call home","done":false}]}`
		if got := get(t, router).Body.String(); got != want {
			t.Fatalf("%s: got the list %s, want %s", test.body, got, want)
		}
	}
}

func TestPatchTodoRefusesABadBody(t *testing.T) {
	const notChange = "the body isn't a JSON object with just a title and a done mark\n"

	tests := []struct{ name, body, want string }{
		{"empty", "", notChange},
		{"invalid JSON", `{"done":`, notChange},
		{"a list", `[true]`, notChange},
		{"a string as done", `{"done":"true"}`, notChange},
		{"a number as done", `{"done":1}`, notChange},
		{"a number as title", `{"title":5}`, notChange},
		{"an unknown member", `{"done":true,"id":1}`, notChange},
		{"a duplicate member", `{"done":true,"done":false}`, notChange},
		{"trailing data", `{"done":true}{}`, notChange},
		{"invalid UTF-8", "{\"title\":\"Buy \xff\"}", notChange},
		{"a blank title", `{"title":" ","done":true}`, "invalid title: empty once trimmed\n"},
		{
			"a long title", `{"title":"` + strings.Repeat("é", 501) + `"}`,
			"invalid title: over 500 characters\n",
		},
		{"an escaped NUL", `{"title":"Buy\u0000milk"}`, "invalid title: has a control character\n"},
	}

	router, _, _ := todos(t)
	post(t, router, `{"title":"Buy milk"}`)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := patch(t, router, "/todos/1", test.body)
			if rec.Code != http.StatusBadRequest || rec.Body.String() != test.want ||
				rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Errorf("got %d %q (%s), want a plain 400 %q",
					rec.Code, rec.Body, rec.Header().Get("Content-Type"), test.want)
			}
		})
	}

	want := `{"todos":[{"id":1,"title":"Buy milk","done":false}]}`
	if got := get(t, router).Body.String(); got != want {
		t.Errorf("got the list %s, want the todo unchanged", got)
	}
}

func TestPatchTodoTakesUpTo64KiB(t *testing.T) {
	const body = `{"done":true}`

	router, _, _ := todos(t)
	post(t, router, `{"title":"Buy milk"}`)

	padded := strings.Repeat(" ", 64<<10-len(body)) + body
	if rec := patch(t, router, "/todos/1", padded); rec.Code != http.StatusOK {
		t.Errorf("64 KiB: got %d %q, want 200", rec.Code, rec.Body)
	}

	rec := patch(t, router, "/todos/1", " "+padded)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("a byte more: got %d %q, want 413", rec.Code, rec.Body)
	}
}

func TestDeleteTodoDeletesATodo(t *testing.T) {
	router, _, _ := todos(t)
	post(t, router, `{"title":"Buy milk"}`)
	post(t, router, `{"title":"Call home"}`)

	rec := remove(t, router, "/todos/1")
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Errorf("got %d %q, want 204 without a body", rec.Code, rec.Body)
	}

	want := `{"todos":[{"id":2,"title":"Call home","done":false}]}`
	if got := get(t, router).Body.String(); got != want {
		t.Errorf("got the list %s, want %s", got, want)
	}
}

func TestATodoThatIsMissingGets404(t *testing.T) {
	router, _, _ := todos(t)
	post(t, router, `{"title":"Buy milk"}`)
	remove(t, router, "/todos/1")

	for _, todoID := range []string{"1", "2", "0", "-1", "x", "1.0", "1e3", "9223372036854775808"} {
		for _, rec := range []*httptest.ResponseRecorder{
			patch(t, router, "/todos/"+todoID, `{"done":true}`), remove(t, router, "/todos/"+todoID),
		} {
			if rec.Code != http.StatusNotFound || rec.Body.String() != "no such todo\n" ||
				rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Errorf("todo %s: got %d %q, want a plain 404", todoID, rec.Code, rec.Body)
			}
		}
	}
}

func TestPatchAndDeleteFailWithoutTheDatabase(t *testing.T) {
	router, pool, logs := todos(t)
	post(t, router, `{"title":"Buy milk"}`)
	pool.Close()

	for _, rec := range []*httptest.ResponseRecorder{
		patch(t, router, "/todos/1", `{"done":true}`), remove(t, router, "/todos/1"),
	} {
		if rec.Code != http.StatusInternalServerError ||
			rec.Body.String() != "Internal Server Error\n" {
			t.Errorf("got %d %q, want a plain 500", rec.Code, rec.Body)
		}
	}

	for _, cause := range []string{
		`msg="change the todo" err="change the todo: `, `msg="delete the todo" err="delete the todo: `,
	} {
		if !strings.Contains(logs.String(), cause) {
			t.Errorf("got the logs %q, want %s", logs, cause)
		}
	}
}

func TestATodoAllowsPatchAndDelete(t *testing.T) {
	router, _, _ := todos(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/todos/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	allow := slices.Sorted(slices.Values(rec.Header().Values("Allow")))
	if rec.Code != http.StatusMethodNotAllowed ||
		!slices.Equal(allow, []string{"DELETE", "PATCH"}) {
		t.Errorf("got %d, Allow %q, want 405 and DELETE, PATCH", rec.Code, allow)
	}
}

// patch returns router's answer to PATCH target with body.
func patch(t *testing.T, router http.Handler, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, target,
		strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

// remove returns router's answer to DELETE target.
func remove(t *testing.T, router http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
