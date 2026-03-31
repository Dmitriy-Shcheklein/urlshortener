package repository

import (
	"context"
	"database/sql/driver"

	"github.com/stoolap/stoolap-go"
)

type Repository struct {
	db *stoolap.DB
}

func New(db *stoolap.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetById(ID string) ([]byte, error) {
	ctx := context.Background()
	rows := r.db.QueryRow(
		ctx, "SELECT url FROM links WHERE short = $1", driver.NamedValue{Ordinal: 1, Value: ID},
	)
	var url []byte
	err := rows.Scan(&url)
	if err != nil {
		return url, err
	}
	return url, nil
}

func (r *Repository) Save(originalUrl []byte, short []byte) error {
	ctx := context.Background()

	_, err := r.db.ExecContext(
		ctx, "INSERT INTO links (url, short) VALUES ($1, $2)",
		driver.NamedValue{Ordinal: 1, Value: string(originalUrl)},
		driver.NamedValue{Ordinal: 2, Value: string(short)},
	)
	if err != nil {
		return err
	}
	return nil
}
