package shortener

import (
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	var (
		service    *Service
		repository *MockLinkRepository
	)

	setup := func(t *testing.T) {
		repository = NewMockLinkRepository(t)
		service = New(repository)
	}

	t.Run(
		"Тест создания сервиса", func(t *testing.T) {
			t.Run(
				"Должен создать сервис", func(t *testing.T) {
					repo := NewMockLinkRepository(t)
					srv := New(repo)

					assert.Equal(t, repo, srv.linkRepository)
				},
			)
		},
	)

	t.Run(
		"Тест GetByID", func(t *testing.T) {
			ID := "ID"
			result := []byte("result")
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					repository.EXPECT().GetByID(ID).Return(result, nil)

					res, err := service.GetByID(ID)

					require.NoError(t, err)
					assert.Equal(t, result, res)
				},
			)

			t.Run(
				"Ошибка получения ссылки", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					repository.EXPECT().GetByID(ID).Return(result, testError)

					_, err := service.GetByID(ID)

					require.Error(t, err)
					assert.Equal(t, testError, err)
				},
			)
		},
	)

	t.Run(
		"Тест CreateShort", func(t *testing.T) {
			originalURL := []byte("original")
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					repository.EXPECT().Save(
						originalURL, mock.Anything,
					).Return(nil)

					_, err := service.CreateShort(originalURL)

					require.NoError(t, err)
				},
			)

			t.Run(
				"Ошибка получения ссылки", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					repository.EXPECT().Save(originalURL, mock.Anything).Return(testError)

					_, err := service.CreateShort(originalURL)

					require.Error(t, err)
					assert.Equal(t, testError, err)
				},
			)
		},
	)

	t.Run(
		"Тест CreateMany", func(t *testing.T) {
			values := []model.CreateManyBodyRaw{
				{CorrelationID: "firstID", OriginalURL: "firstURL"},
				{CorrelationID: "SecondID", OriginalURL: "secondURL"},
			}
			result := []struct{ CorrelationId string }{
				{CorrelationId: values[0].CorrelationID},
				{CorrelationId: values[1].CorrelationID},
			}
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					repository.EXPECT().SaveMany(
						mock.MatchedBy(
							func(value []model.LinkRow) bool {
								return value[0].OriginalURL == values[0].OriginalURL &&
									value[1].OriginalURL == values[1].OriginalURL
							},
						),
					).Return(nil)

					res, err := service.CreateMany(values)

					require.NoError(t, err)
					assert.Equal(t, result[0].CorrelationId, res[0].CorrelationID)
					assert.Equal(t, result[1].CorrelationId, res[1].CorrelationID)
				},
			)

			t.Run(
				"Ошибка при создании ссылок", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					repository.EXPECT().SaveMany(mock.Anything).Return(testError)

					_, err := service.CreateMany(values)

					require.Error(t, err)
					assert.Equal(t, testError, err)
				},
			)
		},
	)
}
