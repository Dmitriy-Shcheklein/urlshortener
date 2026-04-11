package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthcheckRepository(t *testing.T) {
	var (
		mockPool   *MockPool
		repository *Repository
	)

	setup := func(t *testing.T) {
		mockPool = NewMockPool(t)

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
}
