// Package databasetest gives each test its own empty schema in the test database, whose URL is in
// BAKHYARD_TEST_DATABASE (docs/setup.md), so that tests can run side by side.
package databasetest

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// testDatabaseVar holds the URL of the test database, with its password.
const testDatabaseVar = "BAKHYARD_TEST_DATABASE"

// Schema is a test's own schema in the test database.
type Schema struct {
	// URL reaches the schema, without the password, as a service's BAKHYARD_DATABASE does.
	URL      string
	Password string
}

// NewSchema creates an empty schema in the test database, which t's cleanup drops. It fails t
// when BAKHYARD_TEST_DATABASE isn't set.
func NewSchema(t *testing.T) Schema {
	t.Helper()

	address := os.Getenv(testDatabaseVar)
	if address == "" {
		t.Fatalf("%s is not set: docs/setup.md says how to set up the test database",
			testDatabaseVar)
	}

	name := "test_" + strings.ToLower(rand.Text())

	schema, err := within(address, name)
	if err != nil {
		t.Fatalf("parse %s: %v", testDatabaseVar, err)
	}

	quoted := pgx.Identifier{name}.Sanitize()

	err = exec(t.Context(), address, "CREATE SCHEMA "+quoted)
	if err != nil {
		t.Fatalf("create a schema in the test database: %v", err)
	}

	// The test's context is done by the time its cleanup runs.
	t.Cleanup(func() {
		err := exec(context.Background(), address, "DROP SCHEMA "+quoted+" CASCADE")
		if err != nil {
			t.Errorf("drop the schema %s: %v", name, err)
		}
	})

	return schema
}

// within returns the schema name in the database at address.
func within(address, name string) (Schema, error) {
	target, err := url.Parse(address)
	if err != nil {
		return Schema{}, err
	}

	password, _ := target.User.Password()
	target.User = url.User(target.User.Username())

	query := target.Query()
	query.Set("search_path", name)
	target.RawQuery = query.Encode()

	return Schema{URL: target.String(), Password: password}, nil
}

// exec runs sql in the database at address.
func exec(ctx context.Context, address, sql string) error {
	conn, err := pgx.Connect(ctx, address)
	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx, sql)

	return errors.Join(err, conn.Close(ctx))
}
