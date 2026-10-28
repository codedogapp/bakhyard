package question_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
	"github.com/codedogapp/bakhyard/services/nudge/internal/migrations"
	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
)

const (
	id1 = "11111111-1111-4111-8111-111111111111"
	id2 = "22222222-2222-4222-8222-222222222222"
)

func newStore(t *testing.T) (*question.Store, *pgxpool.Pool, *change.Hub) {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	hub := &change.Hub{}

	return question.NewStore(pool, hub), pool, hub
}

func ask(t *testing.T, store *question.Store, id, text string) {
	t.Helper()

	askWith(t, store, question.Question{ID: id, Text: text})
}

func askWith(t *testing.T, store *question.Store, asked question.Question) {
	t.Helper()

	asked.Project = "atlas"

	created, err := store.Ask(t.Context(), asked)
	if err != nil || !created {
		t.Fatalf("Ask(%s): %v, created %v", asked.Text, err, created)
	}
}

func TestAskRefusesABadQuestion(t *testing.T) {
	tests := map[string]struct {
		q    question.Question
		want error
	}{
		"no id":    {question.Question{Text: "x"}, question.ErrBadID},
		"a bad id": {question.Question{ID: "nope", Text: "x"}, question.ErrBadID},
		"no text":  {question.Question{ID: id1, Text: " \n"}, question.ErrNoText},
		"an empty option": {
			question.Question{ID: id1, Text: "x", Options: []string{"a", " "}},
			question.ErrBadOptions,
		},
		"a repeated option": {
			question.Question{ID: id1, Text: "x", Options: []string{"a", "a"}},
			question.ErrBadOptions,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store, _, _ := newStore(t)

			created, err := store.Ask(t.Context(), test.q)
			if created || !errors.Is(err, test.want) {
				t.Errorf("got %v, %v, want %v", created, err, test.want)
			}
		})
	}
}

func TestAskStoresAQuestionOnceAndPendingListsNewestFirst(t *testing.T) {
	store, _, hub := newStore(t)
	changes, end := hub.Subscribe()

	defer end()

	askWith(t, store, question.Question{ID: id1, Text: "first", Options: []string{"yes", "no"}})
	<-changes

	created, err := store.Ask(t.Context(), question.Question{ID: id1, Text: "again"})
	if err != nil || created {
		t.Errorf("asking the same id again: got %v, %v, want no new question", created, err)
	}

	select {
	case <-changes:
		t.Error("an ask that stored nothing published a change")
	default:
	}

	ask(t, store, id2, "second")

	got, err := store.Pending(t.Context())
	if err != nil || len(got) != 2 || got[0].ID != id2 || got[1].Text != "first" ||
		!slices.Equal(got[1].Options, []string{"yes", "no"}) || got[0].Options == nil {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestReplyAnswersAWaitingAgentOnce(t *testing.T) {
	store, _, _ := newStore(t)
	askWith(t, store, question.Question{ID: id1, Text: "Merge?", Options: []string{"yes", "no"}})

	type result struct {
		answer question.Answer
		err    error
	}

	waited := make(chan result, 1)

	go func() {
		answer, err := store.Wait(t.Context(), id1)
		waited <- result{answer, err}
	}()

	bad := map[string]question.Answer{
		"empty":          {},
		"unknown option": {Option: "maybe"},
	}
	for name, answer := range bad {
		if err := store.Reply(t.Context(), id1, answer); err == nil {
			t.Errorf("Reply accepted the %s answer", name)
		}
	}

	if err := store.Reply(
		t.Context(),
		id1,
		question.Answer{Option: "yes", Text: "  go \n"},
	); err != nil {
		t.Fatal(err)
	}

	if got := <-waited; got.err != nil ||
		got.answer != (question.Answer{Option: "yes", Text: "go"}) {
		t.Errorf("the agent got %+v", got)
	}

	if err := store.Reply(
		t.Context(),
		id1,
		question.Answer{Text: "late"},
	); !errors.Is(
		err,
		question.ErrNotFound,
	) {
		t.Errorf("answering twice: got %v, want ErrNotFound", err)
	}

	// An agent that lost its connection asks again and gets the answer at once.
	if got, err := store.Wait(t.Context(), id1); err != nil || got.Text != "go" {
		t.Errorf("waiting again: got %+v, %v", got, err)
	}

	if pending, _ := store.Pending(t.Context()); len(pending) != 0 {
		t.Errorf("an answered question is still pending: %+v", pending)
	}
}

func TestCancelEndsTheWaitAndForgetsTheQuestion(t *testing.T) {
	store, _, _ := newStore(t)
	ask(t, store, id1, "Merge?")

	waited := make(chan error, 1)

	go func() {
		_, err := store.Wait(t.Context(), id1)
		waited <- err
	}()

	// Whether the cancel comes before the agent reads the question or after it subscribes, the
	// wait ends.
	if err := store.Cancel(t.Context(), id1); err != nil {
		t.Fatal(err)
	}

	if err := <-waited; !errors.Is(err, question.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}
