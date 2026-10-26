package todo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// moveSQL puts todo $1 $3 places below todo $2 in the list, 0 for right above it and 1 for right
// below it, and moves every todo at or below that place one down, so the others keep their order.
// It changes no todo when todo $2 doesn't exist.
const moveSQL = `UPDATE todos t
SET position = CASE WHEN t.id = $1 THEN x.position - $3 ELSE t.position - 1 END
FROM todos x WHERE x.id = $2 AND (t.id = $1 OR t.position <= x.position - $3)`

// turnSQL waits until no other move runs and keeps the others waiting until the transaction ends.
// A move's change holds its own todo's row while moveSQL waits for the rows below the new place,
// so two moves at once would each wait for the other's row. The key is the moves' own: nothing
// else in todos' database takes an advisory lock.
const turnSQL = `SELECT pg_advisory_xact_lock(1)`

var (
	// ErrMove is the error of a change that names both a todo to go before and one to go after.
	ErrMove = errors.New("invalid move: before and after together")

	// ErrNeighbour is the error of a todo to go before or after that doesn't exist.
	ErrNeighbour = errors.New("no such todo")
)

// wait makes txn wait for its turn when change moves its todo, so it must come first in txn.
func (change Change) wait(ctx context.Context, txn pgx.Tx) error {
	if change.Before == nil && change.After == nil {
		return nil
	}

	_, err := txn.Exec(ctx, turnSQL)

	return err
}

// move puts the todo todoID where change says, with txn: right before or right after another
// todo in the list (decision 118). A todo named next to itself keeps its place, as only the todos
// below it move, and one that doesn't exist gets an error that wraps ErrNeighbour.
func (change Change) move(ctx context.Context, txn pgx.Tx, todoID int64) error {
	member, neighbour, below := "before", change.Before, 0
	if change.After != nil {
		member, neighbour, below = "after", change.After, 1
	}

	if neighbour == nil {
		return nil
	}

	tag, err := txn.Exec(ctx, moveSQL, todoID, *neighbour, below)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invalid %s: %w", member, ErrNeighbour)
	}

	return nil
}
