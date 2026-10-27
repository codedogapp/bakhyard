-- The Jira tickets dismissed from the Todos page's inbox, each by its key, such as ABC-123, once
-- and for good (docs/plan.md §8.2 jira-dismiss, decisions 128 and 130). The service checks the
-- key, as it checks a todo's ticket.

-- +goose Up
CREATE TABLE dismissals (
    jira_key text PRIMARY KEY
);

-- +goose Down
DROP TABLE dismissals;
