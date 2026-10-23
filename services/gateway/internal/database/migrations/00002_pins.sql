-- The Google account pinned to each allowed email (docs/plan.md §3 Auth, decision 87): the first
-- account that signs in with the email, by its sub, which Google never changes or reuses.

-- +goose Up
CREATE TABLE pins (
    email text PRIMARY KEY,
    sub text NOT NULL,
    pinned_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE pins;
