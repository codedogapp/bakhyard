-- A todo's done mark (docs/plan.md §8.2 todos-edit, decision 110). The todos stored before it
-- aren't done.

-- +goose Up
ALTER TABLE todos ADD COLUMN done boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE todos DROP COLUMN done;
