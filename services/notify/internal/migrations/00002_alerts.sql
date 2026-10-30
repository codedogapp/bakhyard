-- The alerts that Alertmanager posted, a row for each time one fired, for the page that follows
-- them up (docs/plan.md decision 606).

-- +goose Up
CREATE TABLE alerts (
    fingerprint text NOT NULL,
    started_at timestamptz NOT NULL,
    name text NOT NULL,
    summary text NOT NULL,
    description text NOT NULL,
    resolved_at timestamptz,
    PRIMARY KEY (fingerprint, started_at)
);

-- +goose Down
DROP TABLE alerts;
