package healthcheck

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthcheckService(t *testing.T) {
	var (
		mockRepository *RepositoryMock
		service        *Service
	)

	setup := func(t *testing.T) {
		ctrl := minimock.NewController(t)
		mockRepository = NewRepositoryMock(ctrl)

		service = &Service{repository: mockRepository}
	}

	t.Run(
		"Тест создания сервиса", func(t *testing.T) {

			t.Run(
				"Должен создать экземпляр без ошибок", func(t *testing.T) {
					repository, err := New(&RepositoryMock{})

					require.NoError(t, err)
					assert.NotNil(t, repository)
				},
			)

			t.Run(
				"Ошибка, не передан репозиторий", func(t *testing.T) {
					_, err := New(nil)

					require.Error(t, err)
				},
			)
		},
	)

	t.Run(
		"Тест метода PingDB", func(t *testing.T) {
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockRepository.PingMock.Expect().Return(nil)

					err := service.PingDB()

					require.NoError(t, err)
				},
			)

			t.Run(
				"Ошибка при выполнении", func(t *testing.T) {
					setup(t)

					expectedErr := assert.AnError
					mockRepository.PingMock.Expect().Return(expectedErr)

					err := service.PingDB()

					assert.Equal(t, expectedErr, err)
				},
			)
		},
	)
}
