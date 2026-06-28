// Package model contains domain models for the URL shortener service.
package model

// LinkRow represents a shortened URL record stored in the repository.
// It maps to a row in the "links" database table or a JSON line in file storage.
type LinkRow struct {
	ID          string `json:"id" db:"id"`
	ShortURL    string `json:"short_url" db:"short_url"`
	OriginalURL string `json:"original_url" db:"original_url"`
	UserID      string `json:"user_id" db:"user_id"`
	IsDeleted   bool   `json:"is_deleted" db:"is_deleted"`
}

// LinkToDelete represents a link deletion request containing the short URL
// identifier and the ID of the user who owns the link.
type LinkToDelete struct {
	Link   string
	UserID string
}

// CreateManyBodyRaw represents a single item in a batch URL shortening request.
// CorrelationID is a client-assigned identifier used to correlate request items
// with response items in the batch result.
type CreateManyBodyRaw struct {
	CorrelationID string `json:"correlation_id" validate:"required"`
	OriginalURL   string `json:"original_url" validate:"required"`
}

// CreateManyResponseRaw represents a single item in a batch URL shortening response.
// It contains the client's CorrelationID and the generated ShortURL.
type CreateManyResponseRaw struct {
	CorrelationID string `json:"correlation_id" validate:"required"`
	ShortURL      string `json:"short_url" validate:"required"`
}
