package model

import "database/sql"

type LinkRow struct {
	ID          string       `json:"id" db:"id"`
	ShortURL    string       `json:"short_url" db:"short_url"`
	OriginalURL string       `json:"original_url" db:"original_url"`
	UserID      string       `json:"user_id" db:"user_id"`
	IsDeleted   sql.NullBool `db:"is_deleted"`
}
