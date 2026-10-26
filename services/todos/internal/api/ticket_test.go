package api_test

import (
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

func TestPostTodoTakesNoTicket(t *testing.T) {
	router, _, _ := todos(t)

	rec := post(t, router, `{"title":"Buy milk","ticket":"ABC-1"}`)
	if rec.Code != http.StatusBadRequest ||
		rec.Body.String() != "the body isn't a JSON object with just a title\n" {
		t.Errorf(
			"got %d %q, want the plain 400 of a body that isn't a new todo",
			rec.Code,
			rec.Body,
		)
	}

	if got := get(t, router).Body.String(); got != `{"todos":[]}` {
		t.Errorf("got the list %s, want no todo", got)
	}
}
