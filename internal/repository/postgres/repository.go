package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

//go:generate minimock -i Pool -o repository_mock_test.go
type Pool interface {
	Ping() error
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type LinkRow struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Repository struct {
	pool Pool
}

func New(pool Pool) (*Repository, error) {
	repository := &Repository{}
	if pool == nil {
		return repository, errors.New("pool must be not nil")
	}
	repository.pool = pool
	return repository, nil
}

func (r *Repository) Ping() error {
	return r.pool.Ping()
}

func (r *Repository) GetByID(ID string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var originalURL string
	query := fmt.Sprintf("SELECT original_url from %s WHERE short_url = $1", "links")

	err := r.pool.QueryRow(ctx, query, ID).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return []byte(originalURL), nil
}

func (r *Repository) Save(originalUrl []byte, shortUrl []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	query := fmt.Sprintf("INSERT INTO %s (short_url, original_url) VALUES ($1, $2)", "links")

	_, err := r.pool.Exec(ctx, query, string(shortUrl), string(originalUrl))
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	return nil
}
