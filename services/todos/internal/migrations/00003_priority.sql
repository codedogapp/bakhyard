-- A todo's priority, High, Normal or Low (docs/plan.md §8.2 todos-priority, decision 115). The todos
-- stored before it are Normal, as a new todo is (decision 102).

-- +goose Up
ALTER TABLE todos ADD COLUMN priority text NOT NULL DEFAULT 'normal'
    CHECK (priority IN ('high', 'normal', 'low'));

-- +goose Down
ALTER TABLE todos DROP COLUMN priority;
