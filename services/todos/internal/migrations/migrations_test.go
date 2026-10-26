package migrations_test

import (
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/todos/internal/migrations"
)

func TestTheMigrationsMakeTheTodos(t *testing.T) {
	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	for _, test := range []struct {
		what, sql string
		want      []string
	}{
		{
			"migrations",
			"SELECT version_id::text FROM goose_db_version WHERE version_id > 0 ORDER BY id",
			[]string{"1", "2"},
		},
		{
			"columns",
			`SELECT concat_ws(' ', column_name, data_type, is_nullable, column_default,
identity_generation)
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'todos'
ORDER BY ordinal_position`,
			[]string{"id bigint NO ALWAYS", "title text NO", "done boolean NO false"},
		},
		{
			"primary key",
			`SELECT column_name
FROM information_schema.table_constraints
JOIN information_schema.key_column_usage USING (constraint_schema, constraint_name)
WHERE constraint_schema = current_schema() AND table_constraints.table_name = 'todos'
AND constraint_type = 'PRIMARY KEY'`,
			[]string{"id"},
		},
	} {
		got := texts(t, pool, test.sql)
		if !slices.Equal(got, test.want) {
			t.Errorf("got the %s %q, want %q", test.what, got, test.want)
		}
	}
}

// texts returns the one text column of the rows that sql selects in pool's database.
func texts(t *testing.T, pool *pgxpool.Pool, sql string) []string {
	t.Helper()

	rows, err := pool.Query(t.Context(), sql)
	if err != nil {
		t.Fatal(err)
	}

	values, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}

	return values
}
