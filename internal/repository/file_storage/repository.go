package file_storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/google/uuid"
)

type Repository struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Repository {
	return &Repository{cfg: cfg}
}

func (r *Repository) GetByID(id string) ([]byte, error) {
	file, err := os.OpenFile(r.cfg.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return []byte{}, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, id) {
			var raw model.LinkRow

			if err = json.Unmarshal([]byte(line), &raw); err != nil {
				return []byte{}, err
			}

			return []byte(raw.OriginalURL), nil
		}
	}
	return []byte{}, errors.New("link by id not found")
}

func (r *Repository) Save(originalURL []byte, short []byte, userID []byte) error {
	fileRaw := &model.LinkRow{
		OriginalURL: string(originalURL),
		ShortURL:    string(short),
		ID:          uuid.NewString(),
		UserID:      string(userID),
	}

	file, err := os.OpenFile(r.cfg.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	if err = encoder.Encode(fileRaw); err != nil {
		return err
	}
	return nil
}

func (r *Repository) SaveMany(values []model.LinkRow, userID []byte) error {
	file, err := os.OpenFile(r.cfg.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	raws := make([]model.LinkRow, len(values))
	for i := range values {
		raws[i].ID = uuid.NewString()
		raws[i].ShortURL = values[i].ShortURL
		raws[i].OriginalURL = values[i].OriginalURL
		raws[i].UserID = string(userID)
	}
	encoder := json.NewEncoder(file)
	if err = encoder.Encode(raws); err != nil {
		return err
	}
	return nil
}

func (r *Repository) FindByUserID(userID []byte) ([]model.LinkRow, error) {
	file, err := os.OpenFile(r.cfg.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return []model.LinkRow{}, err
	}
	out := make([]model.LinkRow, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, string(userID)) {
			var raw model.LinkRow

			if err = json.Unmarshal([]byte(line), &raw); err != nil {
				return out, err
			}
			out = append(out, raw)
		}
	}
	return out, nil
}
