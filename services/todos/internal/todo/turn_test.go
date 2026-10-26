package todo_test

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestChangeMovesTwoTodosAtOnce(t *testing.T) {
	store, pool := newStore(t)
	for _, title := range []string{"Buy milk", "Answer mail", "Call home", "Pay rent"} {
		add(t, pool, title)
	}

	// Each round moves two todos above the top one at once, then two below it.
	for round := range 10 {
		ids := order(t, store)
		atOnce(t, store, ids[1:3], todo.Change{Before: new(ids[0])})

		got := order(t, store)
		if !together(got[:2], ids[1:3]) || !slices.Equal(got[2:], []int64{ids[0], ids[3]}) {
			t.Fatalf("round %d: got the order %v from %v, want two above %d",
				round, got, ids, ids[0])
		}

		atOnce(t, store, got[2:], todo.Change{After: new(got[0])})

		again := order(t, store)
		if again[0] != got[0] || !together(again[1:3], got[2:]) || again[3] != got[1] {
			t.Fatalf("round %d: got the order %v from %v, want two below %d",
				round, again, got, got[0])
		}
	}
}

// atOnce makes change to each of todoIDs, all at once, and fails t unless every change passes.
func atOnce(t *testing.T, store *todo.Store, todoIDs []int64, change todo.Change) {
	t.Helper()

	start := make(chan struct{})
	errs := make([]error, len(todoIDs))

	var changes sync.WaitGroup
	for i, todoID := range todoIDs {
		changes.Go(func() {
			<-start

			_, errs[i] = store.Change(t.Context(), todoID, change)
		})
	}

	close(start)
	changes.Wait()

	if err := errors.Join(errs...); err != nil {
		t.Fatalf("got %v, want every change", err)
	}
}

// together reports whether got holds the IDs of want, in any order.
func together(got, want []int64) bool {
	return slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want)))
}
