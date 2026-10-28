package question_test

import (
	"errors"
	"testing"

	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
)

func TestExpiredQuestionsAreGone(t *testing.T) {
	store, pool, _ := newStore(t)
	ask(t, store, id1, "old pending")
	ask(t, store, id2, "old answered")

	if err := store.Reply(t.Context(), id2, question.Answer{Text: "ok"}); err != nil {
		t.Fatal(err)
	}

	_, err := pool.Exec(t.Context(),
		"UPDATE questions SET asked_at = now() - interval '25 hours' WHERE id = $1", id1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Wait(t.Context(), id1); !errors.Is(err, question.ErrNotFound) {
		t.Errorf("waiting for an expired question: got %v, want ErrNotFound", err)
	}

	if err := store.Reply(
		t.Context(),
		id1,
		question.Answer{Text: "late"},
	); !errors.Is(
		err,
		question.ErrNotFound,
	) {
		t.Errorf("answering an expired question: got %v", err)
	}

	_, err = pool.Exec(t.Context(),
		"UPDATE questions SET answered_at = now() - interval '8 days' WHERE id = $1", id2)
	if err != nil {
		t.Fatal(err)
	}

	ask(t, store, "33333333-3333-4333-8333-333333333333", "new")

	var count int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM questions").
		Scan(&count); err != nil ||
		count != 1 {
		t.Errorf("%d questions left (%v), want only the new one", count, err)
	}
}

func TestAnIdThatIsNotAUUIDIsNotFound(t *testing.T) {
	store, _, _ := newStore(t)

	if _, err := store.Wait(t.Context(), "nope"); !errors.Is(err, question.ErrNotFound) {
		t.Errorf("Wait: got %v", err)
	}

	if err := store.Reply(
		t.Context(),
		"nope",
		question.Answer{Text: "x"},
	); !errors.Is(
		err,
		question.ErrNotFound,
	) {
		t.Errorf("Reply: got %v", err)
	}

	if err := store.Cancel(t.Context(), "nope"); err != nil {
		t.Errorf("Cancel: got %v", err)
	}
}

func TestTheStoreFailsWithoutTheDatabase(t *testing.T) {
	store, pool, _ := newStore(t)
	pool.Close()

	ctx := t.Context()
	q := question.Question{ID: id1, Text: "x"}

	if _, err := store.Ask(ctx, q); err == nil {
		t.Error("Ask succeeded")
	}

	if _, err := store.Pending(ctx); err == nil {
		t.Error("Pending succeeded")
	}

	if _, err := store.Wait(ctx, id1); err == nil || errors.Is(err, question.ErrNotFound) {
		t.Errorf("Wait: got %v", err)
	}

	if err := store.Reply(ctx, id1, question.Answer{Text: "x"}); err == nil {
		t.Error("Reply succeeded")
	}

	if err := store.Cancel(ctx, id1); err == nil {
		t.Error("Cancel succeeded")
	}
}
