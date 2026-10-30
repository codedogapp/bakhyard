package gitea

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
)

// change is the update that the client asks Gitea for.
type change struct {
	// Content is the file's new bytes, which JSON carries in base64.
	Content []byte `json:"content"`
	// SHA is the ID of the blob that the new bytes replace.
	SHA string `json:"sha"`
}

// Update commits text as the file at name, a path in the repository such as "Notes/Inbox.md",
// on its default branch, in place of blob, the ID of the blob that the file had when it was
// read. It returns ErrChanged when the file has another blob by now, and ErrNotFound when the
// repository no longer has it. The commit is the bot user's, with Gitea's message, such as
// "Update Notes/Inbox.md".
func (c *Client) Update(ctx context.Context, name, blob string, text []byte) error {
	action := "update " + name + " in Gitea"

	// Without a blob, Gitea would make a new file rather than update one.
	if blob == "" {
		return fmt.Errorf("%s: no blob to replace", action)
	}

	target, err := c.target("contents", name)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	body, err := json.Marshal(change{Content: text, SHA: blob})
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	res, err := c.send(ctx, http.MethodPut, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	_ = res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("%s: %w", action, ErrNotFound)
	// Gitea answers a stale blob with a 422, as it does a path that it can't take, which a file
	// that it has can't have.
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("%s: %w", action, ErrChanged)
	default:
		return fmt.Errorf("%s: %s", action, res.Status)
	}
}
