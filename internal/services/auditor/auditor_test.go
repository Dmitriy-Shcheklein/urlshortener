package auditor

import (
	"sync"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newLogger() *zerolog.Logger {
	return new(zerolog.Nop())
}

func TestAuditor(t *testing.T) {
	t.Run("Тест создания Auditor", func(t *testing.T) {
		t.Run("Должен создать Auditor", func(t *testing.T) {
			logger := newLogger()
			a := NewAuditor(logger)

			assert.NotNil(t, a)
			assert.Equal(t, logger, a.logger)
			assert.Empty(t, a.observers)
		})
	})

	t.Run("Тест WithObserver", func(t *testing.T) {
		t.Run("Должен добавить одного observer", func(t *testing.T) {
			a := NewAuditor(newLogger())
			obs := NewMockObserver(t)

			a.WithObserver(obs)

			assert.Len(t, a.observers, 1)
		})

		t.Run("Должен добавить нескольких observers", func(t *testing.T) {
			a := NewAuditor(newLogger())
			obs1 := NewMockObserver(t)
			obs2 := NewMockObserver(t)

			a.WithObserver(obs1)
			a.WithObserver(obs2)

			assert.Len(t, a.observers, 2)
		})
	})

	t.Run("Тест Audit", func(t *testing.T) {
		t.Run("Должен ничего не делать если нет observers", func(t *testing.T) {
			a := NewAuditor(newLogger())

			assert.NotPanics(t, func() {
				a.Audit(nil, "create", "http://example.com")
			})
		})

		t.Run("Должен отправить сообщение всем observers", func(t *testing.T) {
			var wg sync.WaitGroup
			var called1, called2 bool
			wg.Add(2)

			obs1 := NewMockObserver(t)
			obs1.EXPECT().HandleMessage(mock.Anything).Run(func(msg model.AuditMsg) {
				called1 = true
				wg.Done()
			}).Return()

			obs2 := NewMockObserver(t)
			obs2.EXPECT().HandleMessage(mock.Anything).Run(func(msg model.AuditMsg) {
				called2 = true
				wg.Done()
			}).Return()

			a := NewAuditor(newLogger())
			a.WithObserver(obs1)
			a.WithObserver(obs2)

			a.Audit(new("user1"), "create", "http://example.com")
			wg.Wait()

			assert.True(t, called1)
			assert.True(t, called2)
		})

		t.Run("Должен передать корректные данные в сообщение", func(t *testing.T) {
			var received model.AuditMsg
			var wg sync.WaitGroup
			wg.Add(1)

			obs := NewMockObserver(t)
			obs.EXPECT().HandleMessage(mock.Anything).Run(func(msg model.AuditMsg) {
				received = msg
				wg.Done()
			}).Return()

			a := NewAuditor(newLogger())
			a.WithObserver(obs)

			userID := "user1"
			a.Audit(&userID, "delete", "http://example.com/123")
			wg.Wait()

			assert.Equal(t, "delete", received.Action)
			assert.Equal(t, "http://example.com/123", received.URL)
			assert.Equal(t, &userID, received.UserID)
			assert.Greater(t, received.Ts, int64(0))
		})

		t.Run("Должен передать nil UserID", func(t *testing.T) {
			var received model.AuditMsg
			var wg sync.WaitGroup
			wg.Add(1)

			obs := NewMockObserver(t)
			obs.EXPECT().HandleMessage(mock.Anything).Run(func(msg model.AuditMsg) {
				received = msg
				wg.Done()
			}).Return()

			a := NewAuditor(newLogger())
			a.WithObserver(obs)

			a.Audit(nil, "create", "http://example.com")
			wg.Wait()

			assert.Nil(t, received.UserID)
		})
	})
}
