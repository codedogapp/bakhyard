// Package vault saves plans in the Obsidian vault, as notes that the notes service makes.
package vault

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxAnswer bounds what Save reads of notes' answer, a path in JSON.
const maxAnswer = 64 << 10

// Client asks notes to make plans notes of the vault.
type Client struct {
	// Notes is the base URL of the notes service.
	Notes *url.URL
	HTTP  *http.Client
	// Now tells the time, whose day the note's name and frontmatter carry.
	Now func() time.Time
}

// made is notes' answer to a new note.
type made struct {
	Path string `json:"path"`
}

// Save asks notes to make plan a new note, named with today's date and the plan's title, and
// returns the path that the note got: notes adds " 1", " 2" and so on to a name that's taken.
func (c Client) Save(ctx context.Context, plan string) (string, error) {
	draft := newNote(plan, c.Now())
	endpoint := c.Notes.JoinPath("notes", "files", draft.path).String()

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, strings.NewReader(draft.text),
	)
	if err != nil {
		return "", fmt.Errorf("ask notes to make the note: %w", err)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("ask notes to make the note: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("ask notes to make the note: notes answered %s", resp.Status)
	}

	var answer made
	if err := json.UnmarshalRead(io.LimitReader(resp.Body, maxAnswer), &answer); err != nil {
		return "", fmt.Errorf("read notes' answer: %w", err)
	}

	return answer.Path, nil
}
