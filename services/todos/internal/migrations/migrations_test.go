package migrations_test

import (
	"errors"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/todos/internal/migrations"
)

func TestTheMigrationsMakeTheTodos(t *testing.T) {
	pool := open(t, databasetest.NewSchema(t), migrations.FS())

	for _, test := range []struct {
		what, sql string
		want      []string
	}{
		{
			"migrations",
			"SELECT version_id::text FROM goose_db_version WHERE version_id > 0 ORDER BY id",
			[]string{"1", "2", "3"},
		},
		{
			"columns",
			`SELECT concat_ws(' ', column_name, data_type, is_nullable, column_default,
identity_generation)
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'todos'
ORDER BY ordinal_position`,
			[]string{
				"id bigint NO ALWAYS", "title text NO", "done boolean NO false",
				"priority text NO 'normal'::text",
			},
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

func TestThePriorityIsHighNormalOrLow(t *testing.T) {
	pool := open(t, databasetest.NewSchema(t), migrations.FS())

	const insert = "INSERT INTO todos (title, priority) VALUES ('Buy milk', $1)"

	for _, priority := range []string{"high", "normal", "low"} {
		if _, err := pool.Exec(t.Context(), insert, priority); err != nil {
			t.Errorf("%q: got %v, want it stored", priority, err)
		}
	}

	for _, priority := range []string{"High", "urgent", ""} {
		_, err := pool.Exec(t.Context(), insert, priority)
		// 23514 is Postgres's check_violation.
		if refusal, ok := errors.AsType[*pgconn.PgError](err); !ok || refusal.Code != "23514" {
			t.Errorf("%q: got %v, want a check violation", priority, err)
		}
	}
}

func TestTheTodosStoredBeforeThePriorityAreNormal(t *testing.T) {
	schema := databasetest.NewSchema(t)
	before := fstest.MapFS{}

	for _, name := range []string{"00001_todos.sql", "00002_done.sql"} {
		data, err := fs.ReadFile(migrations.FS(), name)
		if err != nil {
			t.Fatal(err)
		}

		before[name] = &fstest.MapFile{Data: data}
	}

	_, err := open(t, schema, before).Exec(t.Context(),
		"INSERT INTO todos (title, done) VALUES ('Buy milk', true)")
	if err != nil {
		t.Fatal(err)
	}

	got := texts(t, open(t, schema, migrations.FS()),
		"SELECT concat_ws(' ', id, title, done::text, priority) FROM todos")
	if want := []string{"1 Buy milk true normal"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// open opens schema and runs the migrations in files that it doesn't have yet.
func open(t *testing.T, schema databasetest.Schema, files fs.FS) *pgxpool.Pool {
	t.Helper()

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, files)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	return pool
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
