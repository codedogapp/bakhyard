package database_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/codedogapp/bakhyard/services/gateway/internal/database"
	"github.com/codedogapp/bakhyard/services/gateway/internal/databasetest"
)

func TestOpenMigratesTheDatabaseOnce(t *testing.T) {
	schema := databasetest.NewSchema(t)

	// The second start finds the migrations done; running one again would fail.
	for range 2 {
		pool, err := database.Open(t.Context(), schema.URL, schema.Password)
		if err != nil {
			t.Fatal(err)
		}

		rows, err := pool.Query(t.Context(),
			"SELECT version_id FROM goose_db_version WHERE version_id > 0 ORDER BY id")
		if err != nil {
			t.Fatal(err)
		}

		versions, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil || !slices.Equal(versions, []int64{1, 2}) {
			t.Errorf("got the migrations %v (%v), want [1 2]", versions, err)
		}

		pool.Close()
	}
}

func TestOpenFailsWhenAMigrationFails(t *testing.T) {
	schema := databasetest.NewSchema(t)

	// A sessions table that the migrations didn't make stops the first one.
	config, err := pgx.ParseConfig(schema.URL)
	if err != nil {
		t.Fatal(err)
	}

	config.Password = schema.Password

	conn, err := pgx.ConnectConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}

	_, err = conn.Exec(t.Context(), "CREATE TABLE sessions (id int)")
	if err != nil {
		t.Fatal(err)
	}

	err = conn.Close(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	_, err = database.Open(t.Context(), schema.URL, schema.Password)
	if err == nil || !strings.HasPrefix(err.Error(), "migrate the database: ") {
		t.Errorf("got %v, want an error that starts with migrate the database", err)
	}
}

func TestOpenFailsWithoutADatabase(t *testing.T) {
	for address, want := range map[string]string{
		// Nothing listens on port 1.
		"postgres://nobody@127.0.0.1:1/nothing": "reach the database: ",
		"postgres://nobody@127.0.0.1:%zz/":      "parse the database URL: ",
	} {
		_, err := database.Open(t.Context(), address, "")
		if err == nil || !strings.HasPrefix(err.Error(), want) {
			t.Errorf("%s: got %v, want an error that starts with %q", address, err, want)
		}
	}
}
