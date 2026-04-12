package model

type LinkRow struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
