package agent

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/codedogapp/bakhyard/services/nudge/internal/session"
)

// start is the body of POST /agent/sessions.
type start struct {
	Project string `json:"project"`
}

// started is the reply to a start: the new session's ID.
type started struct {
	ID string `json:"id"`
}

// report is the body of PUT /agent/sessions/{id}/note.
type report struct {
	Note string `json:"note"`
}

// startSession starts a session and tells the agent its ID. It then holds the request open: the
// session lasts until the agent leaves or the service stops.
func (d Door) startSession(stopping context.Context, w http.ResponseWriter, r *http.Request) {
	var body start
	if !readBody(w, r, &body, "start") {
		return
	}

	live, end := d.Sessions.Start(body.Project)
	defer end()

	w.Header().Set("Content-Type", "application/json")

	if err := json.MarshalWrite(w, started{ID: live.ID}); err != nil {
		return
	}

	if err := http.NewResponseController(w).Flush(); err != nil {
		return
	}

	select {
	case <-r.Context().Done():
	case <-stopping.Done():
	}
}

// report sets the note of the session in the path, and answers 204. A session that has ended
// gets a 404, and an empty note a 400.
func (d Door) report(w http.ResponseWriter, r *http.Request) {
	var body report
	if !readBody(w, r, &body, "report") {
		return
	}

	switch err := d.Sessions.Report(r.PathValue("id"), body.Note); {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, session.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
