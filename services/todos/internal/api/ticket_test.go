package api_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestPatchTodoLinksAndUnlinksATicket(t *testing.T) {
	router, _, _ := todos(t)
	post(t, router, `{"title":"Call home"}`)
	post(t, router, `{"title":"Buy milk"}`)

	const (
		home     = `{"id":1,"title":"Call home","done":false,"priority":"normal"}`
		linked   = `{"id":2,"title":"Buy milk","done":false,"priority":"normal","ticket":"ABC-123"}`
		ticked   = `{"id":2,"title":"Buy milk","done":true,"priority":"normal","ticket":"ABC-123"}`
		relinked = `{"id":2,"title":"Buy oat milk","done":true,"priority":"high","ticket":"OPS-7"}`
		unlinked = `{"id":2,"title":"Buy oat milk","done":true,"priority":"high"}`
	)

	for _, test := range []struct{ body, want, list string }{
		{`{"ticket":"ABC-123"}`, linked, linked + "," + home},
		{`{"ticket":null}`, linked, linked + "," + home},
		{`{}`, linked, linked + "," + home},
		{`{"done":true}`, ticked, ticked + "," + home},
		{
			`{"title":"Buy oat milk","priority":"high","ticket":"OPS-7","after":1}`,
			relinked, home + "," + relinked,
		},
		{`{"ticket":""}`, unlinked, home + "," + unlinked},
		{`{"ticket":""}`, unlinked, home + "," + unlinked},
	} {
		rec := patch(t, router, "/todos/2", test.body)
		if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
			rec.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("%s: got %d %s (%s), want 200 %s as JSON", test.body, rec.Code, rec.Body,
				rec.Header().Get("Content-Type"), test.want)
		}

		want := `{"todos":[` + test.list + `]}`
		if got := get(t, router).Body.String(); got != want {
			t.Fatalf("%s: got the list %s, want %s", test.body, got, want)
		}
	}
}

func TestPatchTodoRefusesABadTicket(t *testing.T) {
	const notTicket = "invalid ticket: not a Jira key such as ABC-123\n"

	router, _, _ := todos(t)
	post(t, router, `{"title":"Buy milk"}`)
	patch(t, router, "/todos/1", `{"ticket":"ABC-1"}`)

	for _, test := range []struct{ name, target, body, want string }{
		{"lower case", "/todos/1", `{"ticket":"abc-1"}`, notTicket},
		{"white space", "/todos/1", `{"ticket":" ABC-1"}`, notTicket},
		{"an escaped line break", "/todos/1", `{"ticket":"ABC-1\n"}`, notTicket},
		{
			"with every other member", "/todos/1",
			`{"title":"Buy oat milk","done":true,"priority":"high","ticket":"ABC 1","after":1}`,
			notTicket,
		},
		{"a number", "/todos/1", `{"ticket":123}`, notChange},
		{"a list", "/todos/1", `{"ticket":["ABC-1"]}`, notChange},
		{"an object", "/todos/1", `{"ticket":{"key":"ABC-1"}}`, notChange},
		{"a boolean", "/todos/1", `{"ticket":false}`, notChange},
		{"a capital member", "/todos/1", `{"Ticket":"ABC-1"}`, notChange},
		{"a duplicate member", "/todos/1", `{"ticket":"ABC-2","ticket":""}`, notChange},
		{
			"and a bad priority", "/todos/1", `{"priority":"urgent","ticket":"abc-1"}`,
			"invalid priority: not high, normal or low\n",
		},
		{"and a bad move", "/todos/1", `{"ticket":"abc-1","before":1,"after":1}`, notTicket},
		{"and a missing neighbour", "/todos/1", `{"ticket":"abc-1","before":9}`, notTicket},
		{"for a missing todo", "/todos/9", `{"ticket":"abc-1"}`, notTicket},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := patch(t, router, test.target, test.body)
			if rec.Code != http.StatusBadRequest || rec.Body.String() != test.want ||
				rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Errorf("got %d %q (%s), want a plain 400 %q",
					rec.Code, rec.Body, rec.Header().Get("Content-Type"), test.want)
			}
		})
	}

	want := `{"todos":[{"id":1,"title":"Buy milk","done":false,"priority":"normal","ticket":"ABC-1"}]}`
	if got := get(t, router).Body.String(); got != want {
		t.Errorf("got the list %s, want the todo unchanged", got)
	}
}

func TestPostTodosLinksATicket(t *testing.T) {
	const linked = `{"id":1,"title":"Fix the date picker","done":false,"priority":"normal",` +
		`"ticket":"WEB-9"}`

	router, _, _ := todos(t)

	rec := post(t, router, `{"title":" Fix the date picker","ticket":"WEB-9"}`)
	if rec.Code != http.StatusCreated || rec.Body.String() != linked ||
		rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("got %d %s (%s), want 201 %s as JSON",
			rec.Code, rec.Body, rec.Header().Get("Content-Type"), linked)
	}

	list := linked

	for i, body := range []string{
		`{"title":"Buy milk","ticket":""}`,
		`{"title":"Buy milk","ticket":null}`,
		`{"title":"Buy milk"}`,
	} {
		want := fmt.Sprintf(`{"id":%d,"title":"Buy milk","done":false,"priority":"normal"}`, i+2)

		rec := post(t, router, body)
		if rec.Code != http.StatusCreated || rec.Body.String() != want {
			t.Fatalf("%s: got %d %s, want 201 %s, without a ticket", body, rec.Code, rec.Body, want)
		}

		list = want + "," + list
	}

	if got := get(t, router).Body.String(); got != `{"todos":[`+list+`]}` {
		t.Errorf("got the list %s, want the four todos, the first with its ticket", got)
	}
}

func TestPostTodosRefusesABadTicket(t *testing.T) {
	const notTicket = "invalid ticket: not a Jira key such as ABC-123\n"

	router, _, _ := todos(t)

	for _, test := range []struct{ name, body, want string }{
		{"lower case", `{"title":"Buy milk","ticket":"abc-1"}`, notTicket},
		{"white space", `{"title":"Buy milk","ticket":" ABC-1"}`, notTicket},
		{"an escaped line break", `{"title":"Buy milk","ticket":"ABC-1\n"}`, notTicket},
		{"and a bad title", `{"title":" ","ticket":"abc-1"}`, "invalid title: empty once trimmed\n"},
		{"a number", `{"title":"Buy milk","ticket":123}`, notAddition},
		{"a list", `{"title":"Buy milk","ticket":["ABC-1"]}`, notAddition},
		{"an object", `{"title":"Buy milk","ticket":{"key":"ABC-1"}}`, notAddition},
		{"a boolean", `{"title":"Buy milk","ticket":false}`, notAddition},
		{"a capital member", `{"title":"Buy milk","Ticket":"ABC-1"}`, notAddition},
		{"a duplicate member", `{"title":"Buy milk","ticket":"ABC-1","ticket":""}`, notAddition},
	} {
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
