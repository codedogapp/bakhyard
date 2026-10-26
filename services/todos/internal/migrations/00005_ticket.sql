-- The Jira ticket that a todo names, by its key, such as ABC-123, or NULL for a todo without one
-- (docs/plan.md §8.2 todos-ticket, decision 125). The todos stored before it name none. The
-- service checks the key, as it checks the title.

-- +goose Up
ALTER TABLE todos ADD COLUMN jira_key text;

-- +goose Down
ALTER TABLE todos DROP COLUMN jira_key;
