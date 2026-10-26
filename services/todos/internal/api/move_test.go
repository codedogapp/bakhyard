package api_test

import (
	"encoding/json/v2"
	"net/http"
	"slices"
	"testing"
)

func TestPatchTodoMovesATodo(t *testing.T) {
	router, _, _ := todos(t)
	for _, body := range []string{`{"title":"Buy milk"}`, `{"title":"Call home"}`, `{"title":"Pay"}`} {
		post(t, router, body)
	}

	for _, test := range []struct {
		target, body, want string
		order              []int64
	}{
		{
			"/todos/1", `{"before":3}`,
			`{"id":1,"title":"Buy milk","done":false,"priority":"normal"}`,
			[]int64{1, 3, 2},
		},
		{
			"/todos/1", `{"after":2}`,
			`{"id":1,"title":"Buy milk","done":false,"priority":"normal"}`,
			[]int64{3, 2, 1},
		},
		{
			"/todos/2", `{"before":null,"after":null}`,
			`{"id":2,"title":"Call home","done":false,"priority":"normal"}`,
			[]int64{3, 2, 1},
		},
		{
			"/todos/3", `{"priority":"high","after":1}`,
			`{"id":3,"title":"Pay","done":false,"priority":"high"}`,
			[]int64{2, 1, 3},
		},
	} {
		rec := patch(t, router, test.target, test.body)
		if rec.Code != http.StatusOK || rec.Body.String() != test.want {
			t.Fatalf("%s: got %d %s, want 200 %s", test.body, rec.Code, rec.Body, test.want)
		}

		if got := order(t, router); !slices.Equal(got, test.order) {
			t.Fatalf("%s: got the order %v, want %v", test.body, got, test.order)
		}
	}
}

func TestPatchTodoRefusesABadMove(t *testing.T) {
	router, _, _ := todos(t)
	for _, body := range []string{`{"title":"Buy milk"}`, `{"title":"Call home"}`, `{"title":"Pay"}`} {
		post(t, router, body)
	}

	list := get(t, router).Body.String()

	for _, test := range []struct {
		target, body string
		code         int
		want         string
	}{
		{"/todos/1", `{"before":3,"after":2}`, 400, "invalid move: before and after together\n"},
		{"/todos/1", `{"title":"Buy oat milk","before":4}`, 400, "invalid before: no such todo\n"},
		{"/todos/3", `{"after":4,"priority":"low"}`, 400, "invalid after: no such todo\n"},
		{"/todos/4", `{"before":5}`, 404, "no such todo\n"},
		{"/todos/1", `{"before":"3"}`, 400, notChange},
		{"/todos/1", `{"after":1.5}`, 400, notChange},
		{"/todos/1", `{"before":true}`, 400, notChange},
		{"/todos/1", `{"after":[3]}`, 400, notChange},
		{"/todos/1", `{"before":9223372036854775808}`, 400, notChange},
		{"/todos/1", `{"After":3}`, 400, notChange},
	} {
		rec := patch(t, router, test.target, test.body)
		if rec.Code != test.code || rec.Body.String() != test.want {
			t.Errorf(
				"%s: got %d %q, want %d %q",
				test.body,
				rec.Code,
				rec.Body,
				test.code,
				test.want,
			)
		}

		if got := get(t, router).Body.String(); got != list {
			t.Errorf("%s: got the list %s, want %s", test.body, got, list)
		}
	}
}

// order returns the IDs of router's todos in the list's order.
func order(t *testing.T, router http.Handler) []int64 {
	t.Helper()

	var list struct {
		Todos []struct {
			ID int64 `json:"id"`
		} `json:"todos"`
	}
	if err := json.Unmarshal(get(t, router).Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}

	var ids []int64
	for _, item := range list.Todos {
		ids = append(ids, item.ID)
	}

	return ids
}
