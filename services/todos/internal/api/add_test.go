package api_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// notAddition is the refusal of a body that isn't a todo to add.
const notAddition = "the body isn't a JSON object with just a title and a ticket\n"

func TestPostTodosAddsATodo(t *testing.T) {
	router, _, _ := todos(t)

	rec := post(t, router, `{"title":"  Buy <milk> & \"eggs\"  "}`)

	want := `{"id":1,"title":"Buy <milk> & \"eggs\"","done":false,"priority":"normal"}`
	if rec.Code != http.StatusCreated || rec.Body.String() != want ||
		rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("got %d %s (%s), want 201 %s as JSON",
			rec.Code, rec.Body, rec.Header().Get("Content-Type"), want)
	}

	if got := get(t, router).Body.String(); got != `{"todos":[`+want+`]}` {
		t.Errorf("got the list %s, want the new todo in it", got)
	}
}

func TestPostTodosRefusesABadBody(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"empty", "", notAddition},
		{"invalid JSON", `{"title":`, notAddition},
		{"a string", `"Buy milk"`, notAddition},
		{"a list", `["Buy milk"]`, notAddition},
		{"a number as title", `{"title":5}`, notAddition},
		{"an unknown member", `{"title":"Buy milk","done":true}`, notAddition},
		{"a priority", `{"title":"Buy milk","priority":"high"}`, notAddition},
		{"a duplicate member", `{"title":"Buy milk","title":"Call home"}`, notAddition},
		{"trailing data", `{"title":"Buy milk"}{}`, notAddition},
		{"invalid UTF-8", "{\"title\":\"Buy \xff\"}", notAddition},
		{"a lone surrogate", `{"title":"Buy \ud800"}`, notAddition},
		{"null", `null`, "invalid title: empty once trimmed\n"},
		{"a null title", `{"title":null}`, "invalid title: empty once trimmed\n"},
		{"no title", `{}`, "invalid title: empty once trimmed\n"},
		{"a blank title", `{"title":" \t "}`, "invalid title: empty once trimmed\n"},
		{
			"a long title", `{"title":"` + strings.Repeat("é", 501) + `"}`,
			"invalid title: over 500 characters\n",
		},
		{"an escaped NUL", `{"title":"Buy\u0000milk"}`, "invalid title: has a control character\n"},
	}

	router, _, _ := todos(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := post(t, router, test.body)
			if rec.Code != http.StatusBadRequest || rec.Body.String() != test.want ||
				rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Errorf("got %d %q (%s), want a plain 400 %q",
					rec.Code, rec.Body, rec.Header().Get("Content-Type"), test.want)
			}
		})
	}

	if got := get(t, router).Body.String(); got != `{"todos":[]}` {
		t.Errorf("got the list %s, want nothing stored", got)
	}
}

func TestPostTodosTakesUpTo64KiB(t *testing.T) {
	const body = `{"title":"Buy milk"}`

	router, _, _ := todos(t)

	padded := strings.Repeat(" ", 64<<10-len(body)) + body
	if rec := post(t, router, padded); rec.Code != http.StatusCreated {
		t.Errorf("64 KiB: got %d %q, want 201", rec.Code, rec.Body)
	}

	for _, large := range []string{" " + padded, `{"title":"` + strings.Repeat("a", 64<<10) + `"}`} {
		rec := post(t, router, large)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("%d bytes: got %d %q, want 413", len(large), rec.Code, rec.Body)
		}
	}
}

func TestPostTodosFailsWithoutTheDatabase(t *testing.T) {
	router, pool, logs := todos(t)
	pool.Close()

	rec := post(t, router, `{"title":"Buy milk"}`)
	if rec.Code != http.StatusInternalServerError ||
		rec.Body.String() != "Internal Server Error\n" {
		t.Errorf("got %d %q, want a plain 500", rec.Code, rec.Body)
	}

	if !strings.Contains(logs.String(), `msg="add the todo" err="add the todo: `) {
		t.Errorf("got the logs %q, want the cause", logs)
	}
}

func TestTodosAllowsGetAndPost(t *testing.T) {
	router, _, _ := todos(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/todos", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	allow := slices.Sorted(slices.Values(rec.Header().Values("Allow")))
	if rec.Code != http.StatusMethodNotAllowed || !slices.Equal(allow, []string{"GET", "POST"}) {
		t.Errorf("got %d, Allow %q, want 405 and GET, POST", rec.Code, allow)
	}
}

// post returns router's answer to POST /todos with body.
func post(t *testing.T, router http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/todos",
		strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
