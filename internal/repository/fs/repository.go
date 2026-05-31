package fs

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Config interface {
	GetFSPath() string
}

type Repository struct {
	cfg Config
}

func New(cfg Config) *Repository {
	return &Repository{cfg: cfg}
}

func (r *Repository) GetByID(id string) ([]byte, error) {
	file, err := os.OpenFile(r.cfg.GetFSPath(), os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
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
			if raw.IsDeleted {
				continue
			}
			return []byte(raw.OriginalURL), nil
		}
	}
	return nil, errors.New("link by id not found")
}

func (r *Repository) Save(originalURL []byte, short []byte, userID []byte) error {
	fileRaw := &model.LinkRow{
		OriginalURL: string(originalURL),
		ShortURL:    string(short),
		ID:          uuid.NewString(),
		UserID:      string(userID),
	}

	file, err := os.OpenFile(r.cfg.GetFSPath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
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
	file, err := os.OpenFile(r.cfg.GetFSPath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
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
	file, err := os.OpenFile(r.cfg.GetFSPath(), os.O_RDONLY|os.O_CREATE, 0o600)
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

func (r *Repository) Delete(in []*model.LinkToDelete) error {
	if len(in) == 0 {
		return nil
	}

	file, err := os.OpenFile(r.cfg.GetFSPath(), os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}

	urlIndex := make(map[string]string)
	for _, item := range in {
		urlIndex[item.Link] = item.UserID
	}

	lines := make([]model.LinkRow, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		var value model.LinkRow
		if err = json.Unmarshal([]byte(line), &value); err != nil {
			return err
		}
		lines = append(lines, value)
	}

	for i := range lines {
		if savedUserID, ok := urlIndex[lines[i].ShortURL]; ok {
			if lines[i].UserID == savedUserID {
				lines[i].IsDeleted = true
			}
		}
	}
	if err = file.Close(); err != nil {
		return err
	}

	if err = os.Remove(r.cfg.GetFSPath()); err != nil {
		return err
	}

	newFile, err := os.OpenFile(r.cfg.GetFSPath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err = newFile.Close(); err != nil {
			log.Err(err).Msg("error while close file")
		}
	}()
	encoder := json.NewEncoder(newFile)
	for _, line := range lines {
		if err = encoder.Encode(line); err != nil {
			return err
		}
	}
	return nil
}
