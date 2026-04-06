package postgres

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthcheckRepository(t *testing.T) {
	var (
		mockPool   *PoolMock
		repository *Repository
	)

	setup := func(t *testing.T) {
		ctrl := minimock.NewController(t)
		mockPool = NewPoolMock(ctrl)

		repository = &Repository{pool: mockPool}
	}

	t.Run(
		"Тест создания репозитория", func(t *testing.T) {

			t.Run(
				"Должен создать экземпляр без ошибок", func(t *testing.T) {
					repo, err := New(&PoolMock{})

					require.NoError(t, err)
					assert.NotNil(t, repo)
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

					mockPool.PingMock.Expect().Return(nil)

					err := repository.Ping()

					require.NoError(t, err)
				},
			)

			t.Run(
				"Ошибка при проверке БД", func(t *testing.T) {
					setup(t)

					expectedErr := assert.AnError
					mockPool.PingMock.Expect().Return(expectedErr)

					err := repository.Ping()

					assert.Equal(t, expectedErr, err)
				},
			)
		},
	)
}
