// Package database opens the gateway's own Postgres database and brings its schema up to date
// with the migrations embedded in the binary.
package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// openTimeout bounds connecting to the database and migrating it.
const openTimeout = 10 * time.Second

//go:embed migrations/*.sql
var migrations embed.FS

// Open connects to the database at address with password and runs the migrations it doesn't
// have yet. It gives up after openTimeout.
func Open(ctx context.Context, address, password string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(address)
	if err != nil {
		return nil, fmt.Errorf("parse the database URL: %w", err)
	}

	config.ConnConfig.Password = password

	ctx, cancel := context.WithTimeout(ctx, openTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open the database: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()

		return nil, fmt.Errorf("reach the database: %w", err)
	}

	err = migrate(ctx, pool)
	if err != nil {
		pool.Close()

		return nil, err
	}

	return pool, nil
}

// migrate runs the migrations that pool's database doesn't have yet.
func migrate(ctx context.Context, pool *pgxpool.Pool) (err error) {
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read the migrations: %w", err)
	}

	// goose needs a database/sql handle. Closing it leaves pool open.
	handle := stdlib.OpenDBFromPool(pool)
	defer func() { err = errors.Join(err, handle.Close()) }()

	provider, err := goose.NewProvider(goose.DialectPostgres, handle, files)
	if err != nil {
		return fmt.Errorf("load the migrations: %w", err)
	}

	_, err = provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate the database: %w", err)
	}

	return nil
}
