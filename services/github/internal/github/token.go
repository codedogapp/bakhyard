package github

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

const (
	// expirationHeader is the header in which GitHub tells when the token of a request expires,
	// absent when it never does.
	expirationHeader = "GitHub-Authentication-Token-Expiration"
	// expirationUTC is the layout of expirationHeader's time in UTC, such as
	// 2027-10-27 22:00:00 UTC, as a classic token's answers carry it.
	expirationUTC = time.DateTime + " UTC"
	// expirationOffset is its layout with an offset, such as 2023-04-26 23:23:18 +0200, as some
	// fine-grained tokens' answers carry it (google/go-github#2649).
	expirationOffset = time.DateTime + " -0700"
)

// Expiry returns when the user's token expires, which GitHub tells in its answer about the user,
// or the zero time for a token that never does. Its errors start with "ask GitHub when the token
// expires", or with "read GitHub's answer" when the time is no use.
func (c *Client) Expiry(ctx context.Context) (time.Time, error) {
	res, err := c.get(ctx, c.settings.API.JoinPath("user"), "ask GitHub when the token expires")
	if err != nil {
		return time.Time{}, err
	}

	_ = res.Body.Close()

	value := res.Header.Get(expirationHeader)
	if value == "" {
		return time.Time{}, nil
	}

	return parseExpiration(value)
}

// parseExpiration returns the time that value, expirationHeader's, holds, in UTC.
func parseExpiration(value string) (time.Time, error) {
	for _, layout := range []string{expirationUTC, expirationOffset} {
		expires, err := time.Parse(layout, value)
		if err == nil {
			return expires.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("read GitHub's answer: %s %q is not a time",
		expirationHeader, value)
}

// Expiration is what an ask of when the user's token expires finds.
type Expiration struct {
	// Expires is when the token expires, zero for never.
	Expires time.Time
}

// ExpiryPoller asks GitHub in the background when the user's token expires, and keeps the latest
// answer.
type ExpiryPoller = poll.Poller[Expiration]

// NewExpiryPoller returns a poller that asks with expiry, such as a Client's Expiry, at once and
// then every five minutes once Run runs, and logs the asks that fail to logger, as "poll GitHub
// token".
func NewExpiryPoller(
	expiry func(context.Context) (time.Time, error), logger *slog.Logger,
) *ExpiryPoller {
	look := func(ctx context.Context) (Expiration, error) {
		expires, err := expiry(ctx)

		return Expiration{Expires: expires}, err
	}

	return poll.New("GitHub token", interval, look, logger)
}
