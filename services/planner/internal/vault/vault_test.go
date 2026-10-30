package vault_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/planner/internal/vault"
)

// request is what the fake notes got.
type request struct {
	method string
	uri    string
	path   string
	body   string
}

// saveTime is the time of the tests' saves: 4 May in UTC, but 5 May in its own zone.
func saveTime() time.Time {
	zone := time.FixedZone("CEST", int((2 * time.Hour).Seconds()))

	return time.Date(2026, time.May, 4, 23, 30, 0, 0, time.UTC).In(zone)
}

// clientOf returns a client of the notes that server is, and closes server at the test's end.
func clientOf(t *testing.T, server *httptest.Server) vault.Client {
	t.Helper()
	t.Cleanup(server.Close)

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return vault.Client{Notes: address, HTTP: server.Client(), Now: saveTime}
}

// makingNotes returns a client of a fake notes that makes a note at the path that it's asked for,
// and sends what it got to the channel.
func makingNotes(t *testing.T) (vault.Client, <-chan request) {
	t.Helper()

	got := make(chan request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		name := strings.TrimPrefix(r.URL.Path, "/notes/files/")

		got <- request{method: r.Method, uri: r.RequestURI, path: name, body: string(body)}

		w.WriteHeader(http.StatusCreated)
		_ = json.MarshalWrite(w, map[string]string{"path": name})
	}))

	return clientOf(t, server), got
}

// answering returns a client of a fake notes that gives every request the same answer.
func answering(t *testing.T, status int, body string) vault.Client {
	t.Helper()

	return clientOf(t, httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, body)
		},
	)))
}

func TestSavePostsTheNoteToNotesAndReturnsItsPath(t *testing.T) {
	client, got := makingNotes(t)

	path, err := client.Save(t.Context(), "# Add dark mode\n\n1. Add a toggle.\n")
	if err != nil {
		t.Fatal(err)
	}

	sent := <-got
	want := "Plans/2026-05-05-add-dark-mode.md"

	if path != want || sent.method != http.MethodPost || sent.path != want {
		t.Errorf("got %q from %s %q, want %q from POST", path, sent.method, sent.path, want)
	}
}

func TestSaveEscapesTheNotesPath(t *testing.T) {
	client, got := makingNotes(t)

	if _, err := client.Save(t.Context(), "# Kávé és tea"); err != nil {
		t.Fatal(err)
	}

	if uri, want := (<-got).uri, "/notes/files/Plans/2026-05-05-k%C3%A1v%C3%A9-%C3%A9s-tea.md"; uri != want {
		t.Errorf("got %q, want %q", uri, want)
	}
}

func TestSaveReturnsThePathThatNotesGave(t *testing.T) {
	client := answering(t, http.StatusCreated, `{"path":"Plans/2026-05-05-add-dark-mode 1.md"}`)

	path, err := client.Save(t.Context(), "# Add dark mode")
	if want := "Plans/2026-05-05-add-dark-mode 1.md"; err != nil || path != want {
		t.Errorf("got %q (%v), want %q", path, err, want)
	}
}

func TestSaveFailsWhenNotesDoesNotMakeTheNote(t *testing.T) {
	for _, status := range []int{
		http.StatusOK, http.StatusBadRequest, http.StatusNotFound, http.StatusMethodNotAllowed,
		http.StatusConflict, http.StatusRequestEntityTooLarge, http.StatusBadGateway,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := answering(t, status, `{"path":"Plans/2026-05-05-add-dark-mode.md"}`)

			_, err := client.Save(t.Context(), "# Add dark mode")
			if want := strconv.Itoa(status); err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("got %v, want an error that names %s", err, want)
			}
		})
	}
}

func TestSaveFailsWhenNotesAnswerIsNotJSON(t *testing.T) {
	for _, body := range []string{"", "made it", `{"path":"a.md"} and more`} {
		client := answering(t, http.StatusCreated, body)

		if path, err := client.Save(t.Context(), "# Add dark mode"); err == nil {
			t.Errorf("got %q for the answer %q, want an error", path, body)
		}
	}
}

func TestSaveFailsWhenNotesIsGone(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	client := clientOf(t, server)

	server.Close()

	if path, err := client.Save(t.Context(), "# Add dark mode"); err == nil {
		t.Errorf("got %q from a notes that is gone, want an error", path)
	}
}

func TestSaveStopsWithItsContext(t *testing.T) {
	client, _ := makingNotes(t)
	ctx, cancel := context.WithCancel(t.Context())

	cancel()

	if _, err := client.Save(ctx, "# Add dark mode"); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}
}
