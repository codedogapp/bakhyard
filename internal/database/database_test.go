package database_test

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
)

func TestOpenMigratesTheDatabaseOnce(t *testing.T) {
	schema := databasetest.NewSchema(t)

	// The second start finds the migrations done; running one again would fail.
	for range 2 {
		pool, err := database.Open(t.Context(), schema.URL, schema.Password, twoTables())
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

		var tables int

		err = pool.QueryRow(t.Context(), "SELECT count(*) FROM pg_tables "+
			"WHERE schemaname = current_schema() AND tablename IN ('first', 'second')").Scan(&tables)
		if err != nil || tables != 2 {
			t.Errorf("got %d of the tables (%v), want both", tables, err)
		}

		pool.Close()
	}
}

func TestOpenFails(t *testing.T) {
	broken := fstest.MapFS{"00001_broken.sql": migration("CREATE TABLE")}

	for name, test := range map[string]struct {
		migrations fstest.MapFS
		want       string
	}{
		"when a migration fails": {broken, "migrate the database: "},
		"without migrations":     {fstest.MapFS{}, "load the migrations: "},
	} {
		schema := databasetest.NewSchema(t)

		_, err := database.Open(t.Context(), schema.URL, schema.Password, test.migrations)
		if err == nil || !strings.HasPrefix(err.Error(), test.want) {
			t.Errorf("%s: got %v, want an error that starts with %q", name, err, test.want)
		}
	}
}

func TestOpenFailsWithoutADatabase(t *testing.T) {
	for address, want := range map[string]string{
		// Nothing listens on port 1.
		"postgres://nobody@127.0.0.1:1/nothing": "reach the database: ",
		"postgres://nobody@127.0.0.1:%zz/":      "parse the database URL: ",
	} {
		_, err := database.Open(t.Context(), address, "", twoTables())
		if err == nil || !strings.HasPrefix(err.Error(), want) {
			t.Errorf("%s: got %v, want an error that starts with %q", address, err, want)
		}
	}
}

// twoTables returns two migrations, as a service embeds them: each makes a table.
func twoTables() fstest.MapFS {
	return fstest.MapFS{
		"00001_first.sql":  migration("CREATE TABLE first (id int);"),
		"00002_second.sql": migration("CREATE TABLE second (id int);"),
	}
}

// migration returns a goose migration file whose Up runs sql.
func migration(sql string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("-- +goose Up\n" + sql + "\n")}
}
