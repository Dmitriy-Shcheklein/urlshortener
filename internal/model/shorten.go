package model

type CreateManyBodyRaw struct {
	CorrelationID string `json:"correlation_id" validate:"required"`
	OriginalURL   string `json:"original_url" validate:"required"`
}

type CreateManyResponseRaw struct {
	CorrelationID string `json:"correlation_id" validate:"required"`
	ShortURL      string `json:"short_url" validate:"required"`
}
