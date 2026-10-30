package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestPlannerSavesPlansThroughNotes(t *testing.T) {
	asked := make(chan string, 1)
	notes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked <- r.Method + " " + r.URL.Path

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"path":"Plans/2026-10-30-add-dark-mode.md"}`))
	}))
	t.Cleanup(notes.Close)

	peers := "todos=tcp://127.0.0.1:8084,notes=tcp://" + notes.Listener.Addr().String()

	client, err := vaultClient(envOf(map[string]string{"BAKHYARD_PEERS": peers}))
	if err != nil {
		t.Fatal(err)
	}

	path, err := client.Save(t.Context(), "# Add dark mode\n")
	if want := "Plans/2026-10-30-add-dark-mode.md"; err != nil || path != want {
		t.Errorf("got %q, %v, want %q", path, err, want)
	}

	// Today's date is in the name, between the folder and the title.
	got := <-asked
	if !strings.HasPrefix(got, "POST /notes/files/Plans/") ||
		!strings.HasSuffix(got, "-add-dark-mode.md") {
		t.Errorf("notes got %s, want POST /notes/files/Plans/<date>-add-dark-mode.md", got)
	}
}

func TestPlannerSavesNoPlansWithoutNotes(t *testing.T) {
	for _, peers := range []string{"", "todos=tcp://127.0.0.1:8084"} {
		client, err := vaultClient(envOf(map[string]string{"BAKHYARD_PEERS": peers}))
		if client != nil || err != nil {
			t.Errorf("%q: got %+v, %v, want no vault and no error", peers, client, err)
		}
	}
}

func TestPlannerStopsOnABadPeersSetting(t *testing.T) {
	var stopping sync.WaitGroup

	// Copilot is set up but has no home: the peers are checked before the CLI starts.
	settings := envOf(map[string]string{hostVar: "example.ghe.com", "BAKHYARD_PEERS": "notes"})

	_, err := routes(t.Context(), settings, secretsWith(nil), &stopping)

	want := `BAKHYARD_PEERS: "notes" is not name=tcp://host:port`
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want the error %s", err, want)
	}
}
