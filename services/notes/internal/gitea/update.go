package gitea

import (
	"context"
	"fmt"
	"net/http"
)

// Update commits text as the file at name, a path in the repository such as "Notes/Inbox.md",
// on its default branch, in place of blob, the ID of the blob that the file had when it was
// read. It returns ErrChanged when the file has another blob by now, ErrNotFound when the
// repository no longer has it, and ErrRefused when Gitea refuses the commit. The commit is the bot
// user's, with Gitea's message, such as "Update Notes/Inbox.md".
func (c *Client) Update(ctx context.Context, name, blob string, text []byte) error {
	action := "update " + name + " in Gitea"

	// Without a blob, Gitea would make a new file rather than update one.
	if blob == "" {
		return fmt.Errorf("%s: no blob to replace", action)
	}

	status, err := c.commit(ctx, http.MethodPut, name, change{Content: text, SHA: blob})
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	switch status {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("%s: %w", action, ErrNotFound)
	// Gitea answers a stale blob with a 422, as it does a path that it can't take, which a file
	// that it has can't have.
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("%s: %w", action, ErrChanged)
	// Gitea answers a commit that another one beat to the branch with a 403, "cannot lock ref", as
	// it does a bot that can't write.
	case http.StatusForbidden:
		return fmt.Errorf("%s: %w", action, ErrRefused)
	default:
		return fmt.Errorf("%s: %d %s", action, status, http.StatusText(status))
	}
}
