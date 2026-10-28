-- The devices that get Web Pushes: each browser's push subscription (docs/plan.md decision 162).

-- +goose Up
CREATE TABLE push_subscriptions (
    endpoint text PRIMARY KEY,
    p256dh text NOT NULL,
    auth text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE push_subscriptions;
