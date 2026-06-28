// Package postgres provides a PostgreSQL connection pool wrapper using pgx.
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps a pgxpool.Pool and provides methods for database operations.
// It automatically registers the UUID type for PostgreSQL connections.
type Pool struct {
	pool *pgxpool.Pool
}

// NewPool creates a new PostgreSQL connection pool from the given connection string.
// The pool registers the UUID type mapping for all new connections.
func NewPool(connString string) (*Pool, error) {
	pool := &Pool{}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return pool, err
	}

	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(
			&pgtype.Type{
				Name:  "uuid",
				OID:   pgtype.UUIDOID,
				Codec: &pgtype.UUIDCodec{},
			},
		)
		conn.TypeMap().RegisterDefaultPgType(&pgtype.UUID{}, "uuid")
		return nil
	}

	newPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return pool, err
	}
	pool.pool = newPool

	return pool, nil
}

// Ping verifies connectivity to the PostgreSQL database with a 1-second timeout.
func (p *Pool) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := p.pool.Ping(ctx); err != nil {
		return err
	}
	return nil
}

// Query executes a query that returns multiple rows.
func (p *Pool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row.
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// Exec executes a query without returning any rows.
func (p *Pool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

// Stop closes the connection pool and releases all resources.
func (p *Pool) Stop() {
	p.pool.Close()
}
