package door

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
)

// ErrSessionGone means the service has no such session: it has restarted since the session began.
var ErrSessionGone = errors.New("the session is gone")

// Session is a session that an agent has started on the service. It lasts until the context that
// started it ends.
type Session struct {
	id     string
	client *Client
}

// Register starts a session for project. The session lasts until ctx ends or the service stops.
func (c *Client) Register(ctx context.Context, project string) (Session, error) {
	body, err := json.Marshal(struct {
		Project string `json:"project"`
	}{project})
	if err != nil {
		return Session{}, fmt.Errorf("write the start: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/agent/sessions", body)
	if err != nil {
		return Session{}, err
	}

	// The body stays open: it keeps the session alive.
	stop := context.AfterFunc(ctx, func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		stop()

		defer func() { _ = resp.Body.Close() }()

		return Session{}, refusal(resp)
	}

	// Decode one value only: the body goes on, open, for as long as the session lasts.
	var started struct {
		ID string `json:"id"`
	}
	if err := json.UnmarshalDecode(jsontext.NewDecoder(resp.Body), &started); err != nil {
		_ = resp.Body.Close()

		return Session{}, fmt.Errorf("read the session: %w", err)
	}

	return Session{id: started.ID, client: c}, nil
}

// Report sets the session's note.
func (s Session) Report(ctx context.Context, note string) error {
	body, err := json.Marshal(struct {
		Note string `json:"note"`
	}{note})
	if err != nil {
		return fmt.Errorf("write the report: %w", err)
	}

	resp, err := s.client.do(ctx, http.MethodPut, "/agent/sessions/"+s.id+"/note", body)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return ErrSessionGone
	default:
		return refusal(resp)
	}
}
