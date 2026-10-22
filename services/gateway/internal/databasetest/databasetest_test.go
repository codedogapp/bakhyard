package databasetest_test

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/codedogapp/bakhyard/services/gateway/internal/databasetest"
)

func TestEachTestGetsAnEmptySchemaOfItsOwn(t *testing.T) {
	schema := databasetest.NewSchema(t)
	if other := databasetest.NewSchema(t); other.URL == schema.URL {
		t.Errorf("got %q twice, want a new schema each time", schema.URL)
	}

	target, err := url.Parse(schema.URL)
	if err != nil {
		t.Fatal(err)
	}

	if _, set := target.User.Password(); set {
		t.Error("the URL has the password, want it apart")
	}

	config, err := pgx.ParseConfig(schema.URL)
	if err != nil {
		t.Fatal(err)
	}

	config.Password = schema.Password

	var (
		current string
		tables  int
	)

	err = connect(t, config).QueryRow(t.Context(), `SELECT current_schema(),
		(SELECT count(*) FROM pg_tables WHERE schemaname = current_schema())`,
	).Scan(&current, &tables)

	name := target.Query().Get("search_path")
	if err != nil || current != name || tables != 0 {
		t.Errorf("got %d tables in %q (%v), want an empty %q", tables, current, err, name)
	}
}

func TestTheSchemaGoesWhenItsTestEnds(t *testing.T) {
	var name string

	t.Run("a test", func(t *testing.T) {
		target, err := url.Parse(databasetest.NewSchema(t).URL)
		if err != nil {
			t.Fatal(err)
		}

		name = target.Query().Get("search_path")
	})

	config, err := pgx.ParseConfig(os.Getenv("BAKHYARD_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}

	var exists bool

	err = connect(t, config).QueryRow(t.Context(),
		"SELECT EXISTS (SELECT FROM pg_namespace WHERE nspname = $1)", name,
	).Scan(&exists)
	if err != nil || name == "" || exists {
		t.Errorf("the schema %q exists (%v) after its test, want it dropped", name, err)
	}
}

// connect connects with config until t ends.
func connect(t *testing.T, config *pgx.ConnConfig) *pgx.Conn {
	t.Helper()

	conn, err := pgx.ConnectConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}

	// The test's context is done by the time its cleanup runs.
	t.Cleanup(func() {
		err := conn.Close(context.Background())
		if err != nil {
			t.Error(err)
		}
	})

	return conn
}
