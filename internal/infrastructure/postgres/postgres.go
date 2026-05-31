package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	pool *pgxpool.Pool
}

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

func (p *Pool) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := p.pool.Ping(ctx); err != nil {
		return err
	}
	return nil
}

func (p *Pool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

func (p *Pool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

func (p *Pool) Stop() {
	p.pool.Close()
}
