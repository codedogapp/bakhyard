-- The user's sessions (docs/plan.md §3 Auth). Each row holds the SHA-256 of a session's token,
-- never the token.

-- +goose Up
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE sessions;
