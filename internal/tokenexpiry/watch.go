package tokenexpiry

import (
	"context"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

// interval is the time between two asks of when a token expires (decision 503).
const interval = 5 * time.Minute

// expiration is what an ask of when a token expires finds.
type expiration struct {
	// Expires is when the token expires, zero for never.
	Expires time.Time
}

// Watch asks ask when a token expires, at once and then every five minutes, in the background
// until ctx ends, and logs each ask that fails to logger, as "poll <source>", such as "poll GitHub
// token". The expiry that it returns waits for the first ask to end, then tells what the latest
// ask to succeed found: the zero time until one has (decision 503).
func Watch(ctx context.Context, source string, ask Expiry, logger *slog.Logger) Expiry {
	look := func(ctx context.Context) (expiration, error) {
		expires, err := ask(ctx)

		return expiration{Expires: expires}, err
	}

	poller := poll.New(source, interval, look, logger)
	go poller.Run(ctx)

	return func(ctx context.Context) (time.Time, error) {
		latest, err := poller.Latest(ctx)

		return latest.Found.Expires, err
	}
}
