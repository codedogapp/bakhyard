package migrations_test

import (
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/gateway/internal/migrations"
)

func TestTheMigrationsMakeTheSessionsAndThePins(t *testing.T) {
	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	rows, err := pool.Query(t.Context(),
		"SELECT version_id FROM goose_db_version WHERE version_id > 0 ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}

	versions, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil || !slices.Equal(versions, []int64{1, 2}) {
		t.Errorf("got the migrations %v (%v), want [1 2]", versions, err)
	}

	rows, err = pool.Query(t.Context(),
		"SELECT tablename FROM pg_tables WHERE schemaname = current_schema() ORDER BY tablename")
	if err != nil {
		t.Fatal(err)
	}

	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil || !slices.Equal(tables, []string{"goose_db_version", "pins", "sessions"}) {
		t.Errorf("got the tables %v (%v), want goose_db_version, pins and sessions", tables, err)
	}
}
