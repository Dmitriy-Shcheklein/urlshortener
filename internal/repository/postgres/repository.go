// Package postgres provides a PostgreSQL-backed repository for storing and
// retrieving shortened URLs. It uses pgx as the database driver.
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
	// PgxRow is an alias for pgx.Row.
	PgxRow = pgx.Row
	// PgxRows is an alias for pgx.Rows.
	PgxRows = pgx.Rows
)

// Pool defines the database connection pool interface used by the repository.
// This allows for easy mocking in tests.
type Pool interface {
	// Ping verifies connectivity to the database.
	Ping() error
	// QueryRow executes a query that returns a single row.
	QueryRow(ctx context.Context, sql string, args ...any) PgxRow
	// Exec executes a query without returning any rows.
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	// Query executes a query that returns multiple rows.
	Query(ctx context.Context, sql string, args ...any) (PgxRows, error)
}

// Repository implements the shortener.LinkRepository interface using PostgreSQL
// as the backing store.
type Repository struct {
	pool Pool
}

// New creates a new PostgreSQL repository. The pool parameter must not be nil.
func New(pool Pool) (*Repository, error) {
	repository := &Repository{}
	if pool == nil {
		return repository, errors.New("pool must be not nil")
	}
	repository.pool = pool
	return repository, nil
}

// Ping verifies connectivity to the PostgreSQL database.
func (r *Repository) Ping() error {
	return r.pool.Ping()
}

// GetByID retrieves the original URL by its short identifier.
// Returns pgx.ErrNoRows if the link does not exist or has been soft-deleted.
func (r *Repository) GetByID(ID string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var originalURL string
	query := fmt.Sprintf("SELECT original_url from %s WHERE short_url = $1 AND is_deleted != true", "links")

	err := r.pool.QueryRow(ctx, query, ID).Scan(&originalURL)
	if err != nil {
		return nil, err
	}
	return []byte(originalURL), nil
}

// Save persists a new short URL mapping. If the original URL already exists,
// it returns a ConflictError containing the existing short URL.
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
		shortenFromDB, err := r.getByOriginalURL(originalUrl)
		if err != nil {
			return err
		}
		return NewConflictError(originalUrl, shortenFromDB)
	}
	return nil
}

func (r *Repository) getByOriginalURL(url []byte) ([]byte, error) {
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

// SaveMany persists multiple short URL mappings in a single batch INSERT statement.
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
			valueRanges,
			fmt.Sprintf("('%s', '%s', '%s')", item.ShortURL, item.OriginalURL, userIDStr),
		)
	}

	query += strings.Join(valueRanges, ", ")

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	return nil
}

// FindByUserID returns all non-deleted shortened URLs owned by the given user.
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

// Delete marks the specified links as deleted by setting the is_deleted flag.
// Only links matching both the short URL and user ID are affected.
func (r *Repository) Delete(in []*model.LinkToDelete) error {
	if len(in) == 0 {
		return nil
	}

	query := "UPDATE links SET is_deleted = true FROM (VALUES %s) AS data(short_url, user_id) WHERE links.short_url = data.short_url AND links.user_id = data.user_id"

	values := make([]string, len(in))
	args := make([]interface{}, 0, len(in)*2)

	for i, item := range in {
		values[i] = fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
		args = append(args, item.Link, item.UserID)
	}

	query = fmt.Sprintf(query, strings.Join(values, ", "))

	_, err := r.pool.Exec(context.Background(), query, args...)
	return err
}
