package question

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
)

const (
	columns = `id::text, project, text, details, options, asked_at`
	// purgeSQL forgets the questions that are too old to matter: $1 and $2 are the TTLs of
	// pending and answered ones, in seconds.
	purgeSQL = `DELETE FROM questions
WHERE (answered_at IS NULL AND asked_at < now() - make_interval(secs => $1))
   OR answered_at < now() - make_interval(secs => $2)`
	askSQL = `INSERT INTO questions (id, project, text, details, options)
VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING`
	pendingSQL = `SELECT ` + columns + ` FROM questions
WHERE answered_at IS NULL AND asked_at >= now() - make_interval(secs => $1)
ORDER BY asked_at DESC, id`
	getSQL = `SELECT ` + columns + `, answered_at IS NOT NULL, option, reply FROM questions
WHERE id = $1 AND (answered_at IS NOT NULL OR asked_at >= now() - make_interval(secs => $2))`
	answerSQL = `UPDATE questions SET answered_at = now(), option = $2, reply = $3
WHERE id = $1 AND answered_at IS NULL AND asked_at >= now() - make_interval(secs => $4)`
	cancelSQL = `DELETE FROM questions WHERE id = $1`
)

// Store keeps the questions in nudge's database, and publishes to hub whenever they change.
type Store struct {
	pool *pgxpool.Pool
	hub  *change.Hub
}

// NewStore returns the store that keeps the questions in pool's database.
func NewStore(pool *pgxpool.Pool, hub *change.Hub) *Store {
	return &Store{pool: pool, hub: hub}
}

// Ask stores the question, and reports whether it's new: a question with its ID that's there
// already stays as it is. It also forgets the questions that expired. A question that fails its
// rules gets that error, and nothing is stored.
func (s *Store) Ask(ctx context.Context, item Question) (bool, error) {
	if err := item.check(); err != nil {
		return false, err
	}

	if item.Options == nil {
		item.Options = []string{}
	}

	_, err := s.pool.Exec(ctx, purgeSQL, pendingTTL.Seconds(), answeredTTL.Seconds())
	if err != nil {
		return false, fmt.Errorf("forget the old questions: %w", err)
	}

	tag, err := s.pool.Exec(
		ctx,
		askSQL,
		item.ID,
		item.Project,
		item.Text,
		item.Details,
		item.Options,
	)
	if err != nil {
		return false, fmt.Errorf("store the question: %w", err)
	}

	created := tag.RowsAffected() == 1
	if created {
		s.hub.Publish()
	}

	return created, nil
}

// Pending lists the questions that wait for an answer, newest first.
func (s *Store) Pending(ctx context.Context) ([]Question, error) {
	// pgx allows ignoring Query's error: CollectRows returns it.
	rows, _ := s.pool.Query(ctx, pendingSQL, pendingTTL.Seconds())

	questions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Question, error) {
		var item Question

		err := row.Scan(
			&item.ID,
			&item.Project,
			&item.Text,
			&item.Details,
			&item.Options,
			&item.AskedAt,
		)

		return item, err
	})
	if err != nil {
		return nil, fmt.Errorf("read the questions: %w", err)
	}

	return questions, nil
}

// get returns the question with questionID, and its answer when it has one.
func (s *Store) get(ctx context.Context, questionID string) (Question, *Answer, error) {
	var (
		item     Question
		answered bool
		answer   Answer
	)

	err := s.pool.QueryRow(ctx, getSQL, questionID, pendingTTL.Seconds()).
		Scan(&item.ID, &item.Project, &item.Text, &item.Details, &item.Options, &item.AskedAt,
			&answered, &answer.Option, &answer.Text)
	if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
		return Question{}, nil, ErrNotFound
	}

	if err != nil {
		return Question{}, nil, fmt.Errorf("read the question: %w", err)
	}

	if !answered {
		return item, nil, nil
	}

	return item, &answer, nil
}

// Reply records the answer to the question with questionID, which wakes the agent that waits for it.
// It returns ErrNotFound when the question is gone or answered already, and the answer's own
// error when it doesn't fit the question.
func (s *Store) Reply(ctx context.Context, questionID string, answer Answer) error {
	item, answered, err := s.get(ctx, questionID)
	if err != nil {
		return err
	}

	if answered != nil {
		return ErrNotFound
	}

	answer.Text = strings.TrimSpace(answer.Text)
	if err := item.accepts(answer); err != nil {
		return err
	}

	tag, err := s.pool.Exec(
		ctx,
		answerSQL,
		questionID,
		answer.Option,
		answer.Text,
		pendingTTL.Seconds(),
	)
	if err != nil {
		return fmt.Errorf("store the answer: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	s.hub.Publish()

	return nil
}

// Cancel forgets the question with questionID, answered or not; one that isn't there is no error.
func (s *Store) Cancel(ctx context.Context, questionID string) error {
	tag, err := s.pool.Exec(ctx, cancelSQL, questionID)
	if isBadUUID(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("cancel the question: %w", err)
	}

	if tag.RowsAffected() > 0 {
		s.hub.Publish()
	}

	return nil
}
