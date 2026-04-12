package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PgxRow = pgx.Row

type Pool interface {
	Ping() error
	QueryRow(ctx context.Context, sql string, args ...any) PgxRow
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
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

	query := fmt.Sprintf(
		"INSERT INTO %s (short_url, original_url) VALUES ($1, $2) ON CONFLICT (original_url) DO NOTHING", "links",
	)

	res, err := r.pool.Exec(
		ctx, query, string(shortUrl), string(originalUrl),
	)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	if res.RowsAffected() != 0 {
		return NewConflictError(originalUrl, shortUrl)
	}
	return nil
}

func (r *Repository) SaveMany(values []model.LinkRow) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if len(values) == 0 {
		return nil
	}

	query := "INSERT INTO links (short_url, original_url) VALUES "
	args := make([]any, 0, len(values)*2)

	for i, item := range values {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
		args = append(args, item.ShortURL, item.OriginalURL)
	}
	query += " ON CONFLICT (original_url) DO NOTHING"

	_, err := r.pool.Exec(
		ctx, query, args...,
	)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	return nil
}
