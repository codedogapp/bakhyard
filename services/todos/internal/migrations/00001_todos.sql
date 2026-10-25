-- The user's todos (docs/plan.md §8.2 todos-jira, decision 102). The done mark, the priority, the
-- order and the Jira key arrive with the steps that need them.

-- +goose Up
CREATE TABLE todos (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title text NOT NULL
);

-- +goose Down
DROP TABLE todos;
