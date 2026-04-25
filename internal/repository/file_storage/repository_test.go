package file_storage

import (
	"os"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindByUserID(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.json")
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(tmpFile.Name()); err != nil {
			log.Err(err)
		}
	}()
	cfg := &config.Config{
		FileStoragePath: tmpFile.Name(),
	}

	repo := New(cfg)

	userID1 := uuid.NewString()
	userID2 := uuid.NewString()

	link1 := &model.LinkRow{
		ID:          uuid.NewString(),
		ShortURL:    "abc123",
		OriginalURL: "http://example.com/1",
		UserID:      userID1,
	}

	link2 := &model.LinkRow{
		ID:          uuid.NewString(),
		ShortURL:    "def456",
		OriginalURL: "http://example.com/2",
		UserID:      userID1,
	}

	link3 := &model.LinkRow{
		ID:          uuid.NewString(),
		ShortURL:    "ghi789",
		OriginalURL: "http://example.com/3",
		UserID:      userID2,
	}

	err = repo.Save([]byte(link1.OriginalURL), []byte(link1.ShortURL), []byte(link1.UserID))
	require.NoError(t, err)

	err = repo.Save([]byte(link2.OriginalURL), []byte(link2.ShortURL), []byte(link2.UserID))
	require.NoError(t, err)

	err = repo.Save([]byte(link3.OriginalURL), []byte(link3.ShortURL), []byte(link3.UserID))
	require.NoError(t, err)

	results, err := repo.FindByUserID([]byte(userID1))
	require.NoError(t, err)
	assert.Len(t, results, 2)

	results2, err := repo.FindByUserID([]byte(userID2))
	require.NoError(t, err)
	assert.Len(t, results2, 1)
}

func TestFindByUserID_Empty(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.json")
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(tmpFile.Name()); err != nil {
			log.Err(err)
		}
	}()

	cfg := &config.Config{
		FileStoragePath: tmpFile.Name(),
	}

	repo := New(cfg)

	userID := uuid.NewString()

	results, err := repo.FindByUserID([]byte(userID))
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestRepository_Delete(t *testing.T) {
	t.Run(
		"Успешное удаление существующих ссылок", func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-*.json")
			require.NoError(t, err)
			defer func() {
				if err = os.Remove(tmpFile.Name()); err != nil {
					log.Err(err)
				}
			}()

			cfg := &config.Config{
				FileStoragePath: tmpFile.Name(),
			}

			repo := New(cfg)

			userID := uuid.NewString()

			link1 := &model.LinkRow{
				ID:          uuid.NewString(),
				ShortURL:    "abc123",
				OriginalURL: "http://example.com/1",
				UserID:      userID,
			}

			link2 := &model.LinkRow{
				ID:          uuid.NewString(),
				ShortURL:    "def456",
				OriginalURL: "http://example.com/2",
				UserID:      userID,
			}

			link3 := &model.LinkRow{
				ID:          uuid.NewString(),
				ShortURL:    "ghi789",
				OriginalURL: "http://example.com/3",
				UserID:      userID,
			}

			err = repo.Save([]byte(link1.OriginalURL), []byte(link1.ShortURL), []byte(link1.UserID))
			require.NoError(t, err)

			err = repo.Save([]byte(link2.OriginalURL), []byte(link2.ShortURL), []byte(link2.UserID))
			require.NoError(t, err)

			err = repo.Save([]byte(link3.OriginalURL), []byte(link3.ShortURL), []byte(link3.UserID))
			require.NoError(t, err)

			err = repo.Delete([]*model.LinkToDelete{
				{Link: "abc123", UserID: userID},
				{Link: "def456", UserID: userID},
			})
			require.NoError(t, err)

			results, err := repo.FindByUserID([]byte(userID))
			require.NoError(t, err)
			assert.Len(t, results, 3)

			for _, result := range results {
				if result.ShortURL == "abc123" || result.ShortURL == "def456" {
					assert.True(t, result.IsDeleted.Valid)
					assert.True(t, result.IsDeleted.Bool)
				} else {
					assert.False(t, result.IsDeleted.Valid)
				}
			}
		},
	)

	t.Run(
		"Удаление несуществующих ссылок", func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-*.json")
			require.NoError(t, err)
			defer func() {
				if err = os.Remove(tmpFile.Name()); err != nil {
					log.Err(err)
				}
			}()

			cfg := &config.Config{
				FileStoragePath: tmpFile.Name(),
			}

			repo := New(cfg)

			userID := uuid.NewString()

			link1 := &model.LinkRow{
				ID:          uuid.NewString(),
				ShortURL:    "abc123",
				OriginalURL: "http://example.com/1",
				UserID:      userID,
			}

			err = repo.Save([]byte(link1.OriginalURL), []byte(link1.ShortURL), []byte(link1.UserID))
			require.NoError(t, err)

			err = repo.Delete([]*model.LinkToDelete{
				{Link: "nonexistent1", UserID: userID},
				{Link: "nonexistent2", UserID: userID},
			})
			require.NoError(t, err)

			results, err := repo.FindByUserID([]byte(userID))
			require.NoError(t, err)
			assert.Len(t, results, 1)

			assert.False(t, results[0].IsDeleted.Valid)
		},
	)

	t.Run(
		"Проверка сохранения файла после удаления", func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-*.json")
			require.NoError(t, err)
			defer func() {
				if err = os.Remove(tmpFile.Name()); err != nil {
					log.Err(err)
				}
			}()

			cfg := &config.Config{
				FileStoragePath: tmpFile.Name(),
			}

			repo := New(cfg)

			userID := uuid.NewString()

			link1 := &model.LinkRow{
				ID:          uuid.NewString(),
				ShortURL:    "abc123",
				OriginalURL: "http://example.com/1",
				UserID:      userID,
			}

			err = repo.Save([]byte(link1.OriginalURL), []byte(link1.ShortURL), []byte(link1.UserID))
			require.NoError(t, err)

			err = repo.Delete([]*model.LinkToDelete{
				{Link: "abc123", UserID: userID},
			})
			require.NoError(t, err)

			results, err := repo.FindByUserID([]byte(userID))
			require.NoError(t, err)
			assert.Len(t, results, 1)
			assert.True(t, results[0].IsDeleted.Valid)
			assert.True(t, results[0].IsDeleted.Bool)
		},
	)

	t.Run(
		"Обработка ошибок файла", func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-*.json")
			require.NoError(t, err)
			defer func() {
				if err = os.Remove(tmpFile.Name()); err != nil {
					log.Err(err)
				}
			}()

			cfg := &config.Config{
				FileStoragePath: tmpFile.Name(),
			}

			repo := New(cfg)

			userID := uuid.NewString()

			link1 := &model.LinkRow{
				ID:          uuid.NewString(),
				ShortURL:    "abc123",
				OriginalURL: "http://example.com/1",
				UserID:      userID,
			}

			err = repo.Save([]byte(link1.OriginalURL), []byte(link1.ShortURL), []byte(link1.UserID))
			require.NoError(t, err)

			err = os.Chmod(tmpFile.Name(), 0o000)
			require.NoError(t, err)

			err = repo.Delete([]*model.LinkToDelete{
				{Link: "abc123", UserID: userID},
			})
			assert.Error(t, err)

			err = os.Chmod(tmpFile.Name(), 0o600)
			require.NoError(t, err)
		},
	)
}
