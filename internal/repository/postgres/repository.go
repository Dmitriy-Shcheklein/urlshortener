package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type (
	PgxRow  = pgx.Row
	PgxRows = pgx.Rows
)

type Pool interface {
	Ping() error
	QueryRow(ctx context.Context, sql string, args ...any) PgxRow
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
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

func (r *Repository) Save(originalUrl []byte, shortUrl []byte, userID []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	userIDStr := string(userID)

	query := fmt.Sprintf(
		"INSERT INTO %s (short_url, original_url, user_id) VALUES ('%s', '%s', '%s') ON CONFLICT (original_url) DO NOTHING",
		"links", string(shortUrl), string(originalUrl), userIDStr,
	)

	res, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	if res.RowsAffected() == 0 {
		shortenFromDB, err := r.geeByOriginalURL(originalUrl)
		if err != nil {
			return err
		}
		return NewConflictError(originalUrl, shortenFromDB)
	}
	return nil
}

func (r *Repository) geeByOriginalURL(url []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var shortURL string
	query := fmt.Sprintf("SELECT short_url from %s WHERE original_url = $1", "links")

	err := r.pool.QueryRow(ctx, query, string(url)).Scan(&shortURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return []byte(shortURL), nil
}

func (r *Repository) SaveMany(values []model.LinkRow, userID []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if len(values) == 0 {
		return nil
	}

	userIDStr := string(userID)

	query := "INSERT INTO links (short_url, original_url, user_id) VALUES "
	valueRanges := make([]string, 0, len(values))

	for _, item := range values {
		valueRanges = append(
			valueRanges, fmt.Sprintf("('%s', '%s', '%s')", item.ShortURL, item.OriginalURL, userIDStr),
		)
	}

	query += strings.Join(valueRanges, ", ")

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	return nil
}

func (r *Repository) FindByUserID(userID []byte) ([]model.LinkRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	userIDStr := string(userID)

	query := fmt.Sprintf(
		"SELECT id, short_url, original_url, user_id from %s WHERE user_id = '%s'::varchar", "links", userIDStr,
	)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	links, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[model.LinkRow])
	if err != nil {
		return nil, err
	}
	return links, nil
}

func (r *Repository) Delete(shortLinks []string) error {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := fmt.Sprintf("UPDATE links SET is_deleted = true WHERE short_url = ANY($1)")

	_, err := r.pool.Exec(ctx, query, shortLinks)
	if err != nil {
		return err
	}
	return nil
}
