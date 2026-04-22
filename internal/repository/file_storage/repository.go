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
	"github.com/rs/zerolog/log"
)

type FileRow struct {
	ID          string `json:"id" db:"id"`
	ShortURL    string `json:"short_url" db:"short_url"`
	OriginalURL string `json:"original_url" db:"original_url"`
	UserID      string `json:"user_id" db:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}

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
	defer func() {
		if err = file.Close(); err != nil {
			log.Err(err).Msg("error while close file")
		}
	}()
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
	defer func() {
		if err = file.Close(); err != nil {
			log.Err(err).Msg("error while close file")
		}
	}()
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
	defer func() {
		if err = file.Close(); err != nil {
			log.Err(err).Msg("error while close file")
		}
	}()
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
	defer func() {
		if err = file.Close(); err != nil {
			log.Err(err).Msg("error while close file")
		}
	}()
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

func (r *Repository) Delete(shortLinks []string) error {
	file, err := os.OpenFile(r.cfg.FileStoragePath, os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}

	urlIndex := make(map[string]struct{})
	for _, link := range shortLinks {
		urlIndex[link] = struct{}{}
	}

	lines := make([]FileRow, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		var value FileRow
		if err = json.Unmarshal([]byte(line), &value); err != nil {
			return err
		}
	}

	for _, line := range lines {
		if _, ok := urlIndex[line.ShortURL]; ok {
			line.IsDeleted = true
		}
	}
	if err = file.Close(); err != nil {
		return err
	}

	if err = os.Remove(r.cfg.FileStoragePath); err != nil {
		return err
	}

	newFile, err := os.OpenFile(r.cfg.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err = newFile.Close(); err != nil {
			log.Err(err).Msg("error while close file")
		}
	}()
	encoder := json.NewEncoder(file)
	if err = encoder.Encode(lines); err != nil {
		return err
	}
	return nil

}
