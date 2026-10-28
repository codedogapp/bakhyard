-- The questions that agents ask the user, and the answers: they live in the database so that a
-- question survives a restart (docs/plan.md decision 163).

-- +goose Up
CREATE TABLE questions (
    id uuid PRIMARY KEY,
    project text NOT NULL,
    text text NOT NULL,
    details text NOT NULL,
    options text[] NOT NULL,
    asked_at timestamptz NOT NULL DEFAULT now(),
    answered_at timestamptz,
    option text NOT NULL DEFAULT '',
    reply text NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE questions;
