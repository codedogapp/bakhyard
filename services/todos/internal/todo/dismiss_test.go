package todo_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestDismissDismissesATicketOnce(t *testing.T) {
	store, _ := newStore(t)

	for _, want := range []bool{true, false, false} {
		added, err := store.Dismiss(t.Context(), "WEB-9")
		if added != want || err != nil {
			t.Fatalf("got %v (%v), want %v", added, err, want)
		}
	}

	got, err := store.Dismissed(t.Context())
	if err != nil || !slices.Equal(got, []string{"WEB-9"}) {
		t.Errorf("got %q (%v), want WEB-9 once", got, err)
	}
}

func TestDismissedListsTheKeysInByteOrder(t *testing.T) {
	store, pool := newStore(t)
	// ICU's root collation puts punctuation before letters, so A_B-1 would come first. The test
	// database's own collation sorts byte by byte, and couldn't tell.
	_, err := pool.Exec(t.Context(),
		`ALTER TABLE dismissals ALTER COLUMN jira_key TYPE text COLLATE "und-x-icu"`)
	if err != nil {
		t.Fatal(err)
	}

	// The keys are stored in neither order.
	for _, key := range []string{"WEB-9", "AB-9", "A_B-1", "OPS-31", "AB-10"} {
		if _, err := store.Dismiss(t.Context(), key); err != nil {
			t.Fatal(err)
		}
	}

	got, err := store.Dismissed(t.Context())

	want := []string{"AB-10", "AB-9", "A_B-1", "OPS-31", "WEB-9"}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %q (%v), want %q", got, err, want)
	}
}

func TestDismissedListsNothingWithoutDismissals(t *testing.T) {
	store, _ := newStore(t)

	got, err := store.Dismissed(t.Context())
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("got %#v (%v), want an empty list", got, err)
	}
}

func TestDismissRefusesAKeyThatIsntJiras(t *testing.T) {
	store, _ := newStore(t)

	for _, key := range []string{
		"", "web-9", "W-9", "WEB-09", "WEB-0", "WEB9", " WEB-9", "WEB-9\n", "WEB-١", "WEB-9/",
	} {
		added, err := store.Dismiss(t.Context(), key)
		if added || !errors.Is(err, todo.ErrTicket) {
			t.Errorf("%q: got %v (%v), want ErrTicket", key, added, err)
		}
	}

	got, err := store.Dismissed(t.Context())
	if err != nil || len(got) != 0 {
		t.Errorf("got %q (%v), want nothing stored", got, err)
	}
}

func TestTheDismissalsFailWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	added, err := store.Dismiss(t.Context(), "WEB-9")
	if added || err == nil || !strings.HasPrefix(err.Error(), "dismiss the ticket: ") ||
		errors.Is(err, todo.ErrTicket) {
		t.Errorf("got %v (%v), want an error that starts with dismiss the ticket", added, err)
	}

	_, err = store.Dismissed(t.Context())
	if err == nil || !strings.HasPrefix(err.Error(), "read the dismissals: ") {
		t.Errorf("got %v, want an error that starts with read the dismissals", err)
	}
}
