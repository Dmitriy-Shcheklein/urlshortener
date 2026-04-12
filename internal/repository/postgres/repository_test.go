package postgres

import (
	"errors"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHealthcheckRepository(t *testing.T) {
	var (
		mockPool   *MockPool
		mockPgxRow *MockPgxRow
		repository *Repository
	)

	setup := func(t *testing.T) {
		mockPool = NewMockPool(t)
		mockPgxRow = NewMockPgxRow(t)

		repository = &Repository{pool: mockPool}
	}

	t.Run(
		"Тест создания репозитория", func(t *testing.T) {
			t.Run(
				"Должен создать экземпляр без ошибок", func(t *testing.T) {
					mockPool = &MockPool{}
					repo, err := New(mockPool)

					require.NoError(t, err)
					assert.NotNil(t, repo)
					assert.Equal(t, mockPool, repo.pool)
				},
			)

			t.Run(
				"Ошибка, не передан pool", func(t *testing.T) {
					_, err := New(nil)

					require.Error(t, err)
				},
			)
		},
	)

	t.Run(
		"Тест метода Ping", func(t *testing.T) {
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockPool.EXPECT().Ping().Return(nil)

					err := repository.Ping()

					require.NoError(t, err)
				},
			)

			t.Run(
				"Ошибка при проверке БД", func(t *testing.T) {
					setup(t)

					expectedErr := assert.AnError
					mockPool.EXPECT().Ping().Return(expectedErr)

					err := repository.Ping()

					assert.Equal(t, expectedErr, err)
				},
			)
		},
	)

	t.Run(
		"Тест метода SaveMany", func(t *testing.T) {
			incoming := []model.LinkRow{
				{ShortURL: "firstShort", OriginalURL: "firstOriginal"},
				{ShortURL: "secondShort", OriginalURL: "secondOriginal"},
			}
			expectedQueryRaw := "INSERT INTO links (short_url, original_url) VALUES ($1, $2), ($3, $4) ON CONFLICT (original_url) DO NOTHING"
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockPool.EXPECT().Exec(
						mock.Anything,
						expectedQueryRaw,
						[]any{"firstShort", "firstOriginal", "secondShort", "secondOriginal"},
					).Return(pgconn.CommandTag{}, nil)

					assert.NoError(t, repository.SaveMany(incoming))
				},
			)

			t.Run(
				"Ошибка при выполнении запроса", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					mockPool.EXPECT().Exec(
						mock.Anything, expectedQueryRaw,
						[]any{"firstShort", "firstOriginal", "secondShort", "secondOriginal"},
					).Return(pgconn.CommandTag{}, testError)

					require.Error(t, repository.SaveMany(incoming))
				},
			)
		},
	)

	t.Run(
		"Тест метода Save", func(t *testing.T) {
			originalUrl := []byte("original")
			shortUrl := []byte("short")
			expectedQueryRaw := "INSERT INTO links (short_url, original_url) VALUES ($1, $2) ON CONFLICT (original_url) DO NOTHING"
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockPool.EXPECT().Exec(
						mock.Anything, expectedQueryRaw,
						[]interface{}{string(shortUrl), string(originalUrl)},
					).Return(pgconn.CommandTag{}, nil)

					assert.NoError(t, repository.Save(originalUrl, shortUrl))
				},
			)

			t.Run(
				"Ошибка при выполнении запроса", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					mockPool.EXPECT().Exec(
						mock.Anything, expectedQueryRaw,
						[]interface{}{string(shortUrl), string(originalUrl)},
					).Return(pgconn.CommandTag{}, testError)

					require.Error(t, repository.Save(originalUrl, shortUrl))
				},
			)

			t.Run(
				"Ошибка конфликт по original_url", func(t *testing.T) {
					setup(t)
					var targetError *ConflictError

					mockPool.EXPECT().Exec(
						mock.Anything, expectedQueryRaw,
						[]interface{}{string(shortUrl), string(originalUrl)},
					).Return(pgconn.NewCommandTag("1"), nil)

					err := repository.Save(originalUrl, shortUrl)

					require.Error(t, err)
					assert.True(t, errors.As(err, &targetError))
				},
			)
		},
	)

	t.Run(
		"Тест метода GetByID", func(t *testing.T) {
			ID := "ID"
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockPool.EXPECT().QueryRow(
						mock.Anything, "SELECT original_url from links WHERE short_url = $1",
						[]interface{}{ID},
					).Return(mockPgxRow)
					mockPgxRow.EXPECT().Scan(mock.Anything).Run(
						func(args ...any) {
							*args[0].(*string) = "result"
						},
					).Return(nil)

					res, err := repository.GetByID(ID)

					require.NoError(t, err)
					assert.Equal(t, []byte("result"), res)
				},
			)

			t.Run(
				"Ошибка при выполнении запроса", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					mockPool.EXPECT().QueryRow(
						mock.Anything, "SELECT original_url from links WHERE short_url = $1",
						[]interface{}{ID},
					).Return(mockPgxRow)
					mockPgxRow.EXPECT().Scan(mock.Anything).Return(testError)

					_, err := repository.GetByID(ID)

					require.Error(t, err)
					assert.Equal(t, testError, err)
				},
			)
		},
	)
}
