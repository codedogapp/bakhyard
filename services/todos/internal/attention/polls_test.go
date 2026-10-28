package attention_test

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/todos/internal/attention"
	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestListItemsListsTheHighTodosBeforeAPollSucceeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tickets := started(t, fakeJira{
			search: func(context.Context) ([]jira.Ticket, error) {
				return nil, errors.New("jira is down")
			},
			done: []jira.Ticket{webNine},
		})

		got := list(t, todosOf(someTodos...), tickets)

		want := []string{callTheBank, shipTheExport, deployHigh}
		if !slices.Equal(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestListItemsKeepsTheDoneTicketsOfTheLastPollToSucceed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		polls := 0
		tickets := started(t, fakeJira{
			search: func(context.Context) ([]jira.Ticket, error) {
				polls++
				if polls > 1 {
					return nil, errors.New("jira is down")
				}

				return nil, nil
			},
			done: []jira.Ticket{webNine},
		})

		list(t, todosOf(someTodos...), tickets)
		time.Sleep(5 * time.Minute)
		synctest.Wait()

		got := list(t, todosOf(someTodos...), tickets)

		want := []string{callTheBank, fixTheLogin, shipTheExport, deployTheFix}
		if polls != 2 || !slices.Equal(got, want) {
			t.Errorf("after %d polls, got %q, want %q", polls, got, want)
		}
	})
}

func TestListItemsWaitsForTheFirstPollBeforeReadingTheTodos(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		tickets := started(t, fakeJira{
			search: func(context.Context) ([]jira.Ticket, error) {
				<-release

				return nil, nil
			},
			done: []jira.Ticket{webNine},
		})

		var read atomic.Bool

		type answer struct {
			res *attentionv1.ListItemsResponse
			err error
		}

		listed := make(chan answer)

		go func() {
			res, err := attention.NewLister(noted(&read), tickets).ListItems(t.Context(),
				&attentionv1.ListItemsRequest{})
			listed <- answer{res, err}
		}()

		synctest.Wait()

		early := read.Load()

		close(release)

		got := <-listed

		want := []string{callTheBank, fixTheLogin, shipTheExport, deployTheFix}
		if early || got.err != nil || !slices.Equal(lines(got.res), want) {
			t.Errorf("read the todos before the first poll ended: %t; got %q and %v, want %q",
				early, lines(got.res), got.err, want)
		}
	})
}

func TestListItemsStopsWaitingWhenItsContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tickets := started(t, fakeJira{
			search: func(ctx context.Context) ([]jira.Ticket, error) {
				<-ctx.Done()

				return nil, ctx.Err()
			},
		})

		var read atomic.Bool

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		res, err := attention.NewLister(noted(&read), tickets).ListItems(ctx,
			&attentionv1.ListItemsRequest{})
		if !errors.Is(err, context.DeadlineExceeded) || res != nil || read.Load() {
			t.Errorf("got %v and %v, read the todos: %t; want no answer, the context's error "+
				"and no read", res, err, read.Load())
		}
	})
}

// noted returns a reader of someTodos that notes in read that it has read them.
func noted(read *atomic.Bool) func(context.Context) ([]todo.Todo, error) {
	return func(ctx context.Context) ([]todo.Todo, error) {
		read.Store(true)

		return todosOf(someTodos...)(ctx)
	}
}
