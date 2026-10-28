package agent_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"strings"
	"testing"
)

// startSession starts a session, and returns its ID and the function that ends it.
func startSession(t *testing.T, gate *door, project string) (string, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gate.server.URL+"/agent/sessions",
		strings.NewReader(`{"project":"`+project+`"}`))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := gate.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = resp.Body.Close() })

	var started struct {
		ID string `json:"id"`
	}
	if err := json.UnmarshalDecode(jsontext.NewDecoder(resp.Body), &started); err != nil {
		t.Fatal(err)
	}

	return started.ID, cancel
}

func TestASessionLastsAsLongAsItsRequestAndKeepsItsNote(t *testing.T) {
	gate := newDoor(t)
	changes, end := gate.hub.Subscribe()

	defer end()

	sessionID, leave := startSession(t, gate, "atlas")

	if got := gate.call(
		t,
		http.MethodPut,
		"/agent/sessions/"+sessionID+"/note",
		`{"note":"reading the code"}`,
	); got.status != http.StatusNoContent {
		t.Fatalf("report: reply %d %s", got.status, got.body)
	}

	if listed := gate.sessions.List(); len(listed) != 1 || listed[0].Project != "atlas" ||
		listed[0].Note != "reading the code" {
		t.Errorf("sessions: %+v", listed)
	}

	<-changes
	leave()

	// The session ends once its request ends; the hub tells when.
	for len(gate.sessions.List()) != 0 {
		<-changes
	}
}

func TestReportingGetsARefusalWhenItCannotBeKept(t *testing.T) {
	gate := newDoor(t)
	sessionID, _ := startSession(t, gate, "atlas")

	tests := []struct {
		name, path, body string
		want             int
	}{
		{"an unknown session", "/agent/sessions/nope/note", `{"note":"x"}`, http.StatusNotFound},
		{
			"an empty note",
			"/agent/sessions/" + sessionID + "/note",
			`{"note":" "}`,
			http.StatusBadRequest,
		},
		{"a bad body", "/agent/sessions/" + sessionID + "/note", `nope`, http.StatusBadRequest},
	}

	for _, test := range tests {
		if got := gate.call(t, http.MethodPut, test.path, test.body); got.status != test.want {
			t.Errorf("%s: reply %d %s, want %d", test.name, got.status, got.body, test.want)
		}
	}
}

func TestASessionEndsWhenTheServiceStops(t *testing.T) {
	gate := newDoor(t)
	changes, end := gate.hub.Subscribe()

	defer end()

	startSession(t, gate, "atlas")
	gate.stop()

	for len(gate.sessions.List()) != 0 {
		<-changes
	}
}

func TestABadStartGetsA400(t *testing.T) {
	gate := newDoor(t)

	if got := gate.call(
		t,
		http.MethodPost,
		"/agent/sessions",
		`{"extra":1}`,
	); got.status != http.StatusBadRequest {
		t.Errorf("reply %d %s, want 400", got.status, got.body)
	}
}
