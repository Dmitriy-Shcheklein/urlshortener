package delete_links_worker

import (
	"context"
	"testing"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeleteLinksWorker(t *testing.T) {
	var (
		service *MockService
		worker  *DeleteLinksWorker
	)

	setup := func(t *testing.T) {
		service = NewMockService(t)
		worker = New(service)
		logger.Logger = new(zerolog.Nop())
	}

	t.Run(
		"Тест создания воркера ", func(t *testing.T) {
			t.Run(
				"Должен создать экземпляр", func(t *testing.T) {
					svc := NewMockService(t)

					w := New(svc)

					require.NotNil(t, w)
					assert.Equal(t, svc, w.service)
				},
			)
		},
	)

	t.Run(
		"Тест метода Start", func(t *testing.T) {
			ctx := context.Background()

			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					worker.Start(ctx)

					require.NotNil(t, worker.cancelFunc)
				},
			)
		},
	)

	t.Run(
		"Тест метода AddToQueue", func(t *testing.T) {
			urls1 := []string{"1", "2"}
			userID1 := "userID1"
			urls2 := []string{"3"}
			userID2 := "userID2"

			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					runs := []struct {
						urls   []string
						userID string
					}{
						{urls: urls1, userID: userID1},
						{urls: urls2, userID: userID2},
					}
					var forDelete []*model.LinkToDelete
					worker.timeout = time.Millisecond
					worker.Start(context.Background())

					service.EXPECT().Delete(
						mock.MatchedBy(
							func(links []*model.LinkToDelete) bool {
								forDelete = links
								return true
							},
						),
					).Return(nil)

					for _, run := range runs {
						go worker.AddToQueue(run.urls, run.userID)
					}

					assert.Eventually(
						t, func() bool {
							return len(forDelete) == 3
						}, 2*time.Second, 100*time.Millisecond,
					)
				},
			)

			t.Run(
				"Тест паники", func(t *testing.T) {
					setup(t)

					service.EXPECT().Delete(mock.Anything).Panic("panic")

					errChan := worker.Start(context.Background())

					worker.AddToQueue([]string{"url"}, "userID")

					require.Error(t, <-errChan)
				},
			)
		},
	)

	t.Run(
		"Тест метода Stop", func(t *testing.T) {
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					worker.Start(context.Background())

					require.NotPanics(t, func() { worker.Stop() })
				},
			)
		},
	)
}
