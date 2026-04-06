package postgres

import (
	"context"
	"time"

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

func (p *Pool) Stop() {
	p.pool.Close()
}
