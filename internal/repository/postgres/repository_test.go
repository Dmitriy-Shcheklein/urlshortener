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
		mockPool    *MockPool
		mockPgxRow  *MockPgxRow
		mockPgxRows *MockPgxRows
		repository  *Repository
	)

	setup := func(t *testing.T) {
		mockPool = NewMockPool(t)
		mockPgxRow = NewMockPgxRow(t)
		mockPgxRows = NewMockPgxRows(t)

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
			userID := []byte("userID")
			incoming := []model.LinkRow{
				{ShortURL: "firstShort", OriginalURL: "firstOriginal"},
				{ShortURL: "secondShort", OriginalURL: "secondOriginal"},
			}
			expectedQueryRaw := "INSERT INTO links (short_url, original_url, user_id) VALUES ('firstShort', 'firstOriginal', 'userID'), ('secondShort', 'secondOriginal', 'userID')"
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockPool.EXPECT().Exec(
						mock.Anything,
						expectedQueryRaw,
					).Return(pgconn.CommandTag{}, nil)

					assert.NoError(t, repository.SaveMany(incoming, userID))
				},
			)

			t.Run(
				"Ошибка при выполнении запроса", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					mockPool.EXPECT().Exec(
						mock.Anything, expectedQueryRaw,
					).Return(pgconn.CommandTag{}, testError)

					require.Error(t, repository.SaveMany(incoming, userID))
				},
			)
		},
	)

	t.Run(
		"Тест метода Save", func(t *testing.T) {
			originalUrl := []byte("original")
			shortUrl := []byte("short")
			userID := []byte("userID")
			expectedQueryRaw := "INSERT INTO links (short_url, original_url, user_id) VALUES ('short', 'original', 'userID') ON CONFLICT (original_url) DO NOTHING"
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockPool.EXPECT().Exec(
						mock.Anything, expectedQueryRaw,
					).Return(pgconn.NewCommandTag("1"), nil)

					assert.NoError(t, repository.Save(originalUrl, shortUrl, userID))
				},
			)

			t.Run(
				"Ошибка при выполнении запроса", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					mockPool.EXPECT().Exec(
						mock.Anything, expectedQueryRaw,
					).Return(pgconn.NewCommandTag("1"), testError)

					require.Error(t, repository.Save(originalUrl, shortUrl, userID))
				},
			)

			t.Run(
				"Ошибка конфликт по original_url", func(t *testing.T) {
					setup(t)
					var targetError *ConflictError

					mockPool.EXPECT().Exec(
						mock.Anything, expectedQueryRaw,
					).Return(pgconn.NewCommandTag(""), nil)
					mockPool.EXPECT().QueryRow(
						mock.Anything, "SELECT short_url from links WHERE original_url = $1",
						[]interface{}{string(originalUrl)},
					).Return(mockPgxRow)
					mockPgxRow.EXPECT().Scan(mock.Anything).Run(
						func(args ...any) {
							*args[0].(*string) = string(shortUrl)
						},
					).Return(nil)

					err := repository.Save(originalUrl, shortUrl, userID)

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

	t.Run(
		"Тест метода FindByUserID", func(t *testing.T) {
			userID := []byte("userID")
			expectedQuery := "SELECT id, short_url, original_url, user_id from links WHERE user_id = 'userID'::varchar"
			expectedRes := []model.LinkRow{
				{
					ID:          "id1",
					OriginalURL: "original1",
					ShortURL:    "short1",
					UserID:      "userID",
				},
				{
					ID:          "id2",
					OriginalURL: "original2",
					ShortURL:    "short2",
					UserID:      "userID",
				},
			}
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockPool.EXPECT().Query(
						mock.Anything, expectedQuery,
					).Return(mockPgxRows, nil)

					for i := 0; i < len(expectedRes); i++ {
						mockPgxRows.EXPECT().FieldDescriptions().Return(
							[]pgconn.FieldDescription{
								{Name: "id"},
								{Name: "short_url"},
								{Name: "original_url"},
								{Name: "user_id"},
							},
						).Once()
					}

					mockPgxRows.EXPECT().Next().Return(true).Once()
					mockPgxRows.EXPECT().Scan(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(
						func(args ...any) {
							*args[0].(*string) = "id1"
							*args[1].(*string) = "short1"
							*args[2].(*string) = "original1"
							*args[3].(*string) = "userID"
						},
					).Return(nil).Once()

					mockPgxRows.EXPECT().Next().Return(true).Once()
					mockPgxRows.EXPECT().Scan(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(
						func(args ...any) {
							*args[0].(*string) = "id2"
							*args[1].(*string) = "short2"
							*args[2].(*string) = "original2"
							*args[3].(*string) = "userID"
						},
					).Return(nil).Once()

					mockPgxRows.EXPECT().Next().Return(false).Once()
					mockPgxRows.EXPECT().Err().Return(nil).Once()
					mockPgxRows.EXPECT().Close().Return().Once()

					res, err := repository.FindByUserID(userID)

					require.NoError(t, err)
					assert.Equal(t, res, expectedRes)
				},
			)

			t.Run(
				"Ошибка выполнения запроса", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					mockPool.EXPECT().Query(
						mock.Anything, expectedQuery,
					).Return(mockPgxRows, testError)

					res, err := repository.FindByUserID(userID)

					require.Error(t, err)
					assert.Equal(t, testError, err)
					assert.Nil(t, res)
				},
			)

			t.Run(
				"Ошибка десериализации данных", func(t *testing.T) {
					setup(t)

					testError := assert.AnError
					mockPool.EXPECT().Query(
						mock.Anything, expectedQuery,
					).Return(mockPgxRows, nil)

					mockPgxRows.EXPECT().FieldDescriptions().Return(
						[]pgconn.FieldDescription{
							{Name: "id"},
							{Name: "short_url"},
							{Name: "original_url"},
							{Name: "user_id"},
						},
					).Once()

					mockPgxRows.EXPECT().Next().Return(true).Once()
					mockPgxRows.EXPECT().Scan(
						mock.Anything, mock.Anything, mock.Anything, mock.Anything,
					).Return(testError).Once()

					mockPgxRows.EXPECT().Close().Return().Once()

					res, err := repository.FindByUserID(userID)

					require.Error(t, err)
					assert.Equal(t, testError, err)
					assert.Nil(t, res)
				},
			)
		},
	)
}
