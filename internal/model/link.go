package model

type LinkRow struct {
	ID          string `json:"id" db:"id"`
	ShortURL    string `json:"short_url" db:"short_url"`
	OriginalURL string `json:"original_url" db:"original_url"`
	UserID      string `json:"user_id" db:"user_id"`
	IsDeleted   bool   `json:"is_deleted" db:"is_deleted"`
}

type LinkToDelete struct {
	Link   string
	UserID string
}

type CreateManyBodyRaw struct {
	CorrelationID string `json:"correlation_id" validate:"required"`
	OriginalURL   string `json:"original_url" validate:"required"`
}

type CreateManyResponseRaw struct {
	CorrelationID string `json:"correlation_id" validate:"required"`
	ShortURL      string `json:"short_url" validate:"required"`
}
