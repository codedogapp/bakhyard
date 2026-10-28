package question

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// invalidUUID is Postgres's error code for text that isn't a UUID.
const invalidUUID = "22P02"

// isBadUUID reports whether err says that an ID wasn't a UUID.
func isBadUUID(err error) bool {
	if failed, ok := errors.AsType[*pgconn.PgError](err); ok {
		return failed.Code == invalidUUID
	}

	return false
}

// Wait waits until the question with questionID has an answer and returns it, or until ctx ends. It
// returns ErrNotFound when the question is gone: cancelled or expired. An answered question
// returns its answer at once, so an agent that lost its connection gets it by asking again.
func (s *Store) Wait(ctx context.Context, questionID string) (Answer, error) {
	ctx, stop := context.WithTimeoutCause(ctx, pendingTTL, ErrNotFound)
	defer stop()

	changes, end := s.hub.Subscribe()
	defer end()

	for {
		_, answer, err := s.get(ctx, questionID)
		if err != nil {
			return Answer{}, err
		}

		if answer != nil {
			return *answer, nil
		}

		select {
		case <-changes:
		case <-ctx.Done():
			return Answer{}, fmt.Errorf("wait for the answer: %w", context.Cause(ctx))
		}
	}
}
