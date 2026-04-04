package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	"github.com/google/uuid"
)

type FileRaw struct {
	ID          string `json:"uuid" validate:"required"`
	ShortURL    string `json:"short_url" validate:"required"`
	OriginalURL string `json:"original_url" validate:"required"`
}

type Repository struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Repository {
	return &Repository{cfg: cfg}
}

func (r *Repository) GetByID(id string) ([]byte, error) {
	file, err := os.OpenFile(r.cfg.FileStoragePath, os.O_RDONLY, 0600)
	if err != nil {
		return []byte{}, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, id) {
			var raw FileRaw

			if err = json.Unmarshal([]byte(line), &raw); err != nil {
				return []byte{}, err
			}

			return []byte(raw.OriginalURL), nil
		}
	}
	return []byte{}, errors.New("link by id not found")
}

func (r *Repository) Save(originalURL []byte, short []byte) error {
	fileRaw := &FileRaw{
		OriginalURL: string(originalURL),
		ShortURL:    string(short),
		ID:          uuid.NewString(),
	}

	file, err := os.OpenFile(r.cfg.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	if err = encoder.Encode(fileRaw); err != nil {
		return err
	}
	return nil
}
