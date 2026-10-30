package gitea

import (
	"context"
	"fmt"
	"net/http"
)

// Create commits text as a new file at name, a path in the repository such as "Notes/Inbox.md",
// on its default branch. It returns ErrTaken when the path is taken, and ErrRefused when Gitea
// refuses the commit. The commit is the bot user's, with Gitea's message, such as
// "Add Notes/Inbox.md".
func (c *Client) Create(ctx context.Context, name string, text []byte) error {
	action := "create " + name + " in Gitea"

	status, err := c.commit(ctx, http.MethodPost, name, change{Content: text})
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	switch status {
	case http.StatusCreated:
		return nil
	// Gitea answers a file that's there already with a 422, as it does a file where one of the
	// path's folders would be, or a folder at the path.
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("%s: %w", action, ErrTaken)
	// As for an update: a commit that another one beat to the branch, or a bot that can't write.
	case http.StatusForbidden:
		return fmt.Errorf("%s: %w", action, ErrRefused)
	default:
		return fmt.Errorf("%s: %d %s", action, status, http.StatusText(status))
	}
}
