package auditor

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newLogger() *zerolog.Logger {
	return new(zerolog.Nop())
}

type blockingObserver struct {
	block chan struct{}
}

func (o *blockingObserver) HandleMessage(msg model.AuditMsg) {
	<-o.block
}

func TestAuditor(t *testing.T) {
	t.Run(
		"Тест создания Auditor", func(t *testing.T) {
			t.Run(
				"Должен создать Auditor", func(t *testing.T) {
					logger := newLogger()
					a := NewAuditor(logger)

					assert.NotNil(t, a)
					assert.Equal(t, logger, a.logger)
					assert.Empty(t, a.pools)
				},
			)
		},
	)

	t.Run(
		"Тест WithObserver", func(t *testing.T) {
			t.Run(
				"Должен добавить одного observer", func(t *testing.T) {
					a := NewAuditor(newLogger(), NewMockObserver(t))

					assert.Len(t, a.pools, 1)
				},
			)

			t.Run(
				"Должен добавить нескольких observers", func(t *testing.T) {
					a := NewAuditor(newLogger(), NewMockObserver(t), NewMockObserver(t))

					assert.Len(t, a.pools, 2)
				},
			)
		},
	)

	t.Run(
		"Тест Audit", func(t *testing.T) {
			t.Run(
				"Должен ничего не делать если нет observers", func(t *testing.T) {
					a := NewAuditor(newLogger())

					assert.NotPanics(
						t, func() {
							a.Audit(nil, "create", "http://example.com")
						},
					)
				},
			)

			t.Run(
				"Должен отправить сообщение всем observers", func(t *testing.T) {
					var wg sync.WaitGroup
					var called1, called2 bool
					wg.Add(2)

					obs1 := NewMockObserver(t)
					obs1.EXPECT().HandleMessage(mock.Anything).Run(
						func(msg model.AuditMsg) {
							called1 = true
							wg.Done()
						},
					).Return()

					obs2 := NewMockObserver(t)
					obs2.EXPECT().HandleMessage(mock.Anything).Run(
						func(msg model.AuditMsg) {
							called2 = true
							wg.Done()
						},
					).Return()

					a := NewAuditor(newLogger(), obs1, obs2)
					defer a.Shutdown()

					a.Audit(new("user1"), "create", "http://example.com")
					wg.Wait()

					assert.True(t, called1)
					assert.True(t, called2)
				},
			)

			t.Run(
				"Должен передать корректные данные в сообщение", func(t *testing.T) {
					var received model.AuditMsg
					var wg sync.WaitGroup
					wg.Add(1)

					obs := NewMockObserver(t)
					obs.EXPECT().HandleMessage(mock.Anything).Run(
						func(msg model.AuditMsg) {
							received = msg
							wg.Done()
						},
					).Return()

					a := NewAuditor(newLogger(), obs)
					defer a.Shutdown()

					userID := "user1"
					a.Audit(&userID, "delete", "http://example.com/123")
					wg.Wait()

					assert.Equal(t, "delete", received.Action)
					assert.Equal(t, "http://example.com/123", received.URL)
					assert.Equal(t, &userID, received.UserID)
					assert.Greater(t, received.Ts, int64(0))
				},
			)

			t.Run(
				"Должен передать nil UserID", func(t *testing.T) {
					var received model.AuditMsg
					var wg sync.WaitGroup
					wg.Add(1)

					obs := NewMockObserver(t)
					obs.EXPECT().HandleMessage(mock.Anything).Run(
						func(msg model.AuditMsg) {
							received = msg
							wg.Done()
						},
					).Return()

					a := NewAuditor(newLogger(), obs)
					defer a.Shutdown()

					a.Audit(nil, "create", "http://example.com")
					wg.Wait()

					assert.Nil(t, received.UserID)
				},
			)
		},
	)

	t.Run(
		"Тест Shutdown", func(t *testing.T) {
			t.Run(
				"Должен обработать все сообщения при Shutdown", func(t *testing.T) {
					var count atomic.Int32

					obs := NewMockObserver(t)
					obs.EXPECT().HandleMessage(mock.Anything).Run(
						func(msg model.AuditMsg) {
							count.Add(1)
						},
					).Return()

					a := NewAuditor(newLogger(), obs)

					for i := 0; i < 10; i++ {
						a.Audit(nil, "test", "http://example.com")
					}
					a.Shutdown()

					assert.Equal(t, int32(10), count.Load())
				},
			)

			t.Run(
				"Должен не блокироваться при переполнении канала", func(t *testing.T) {
					obs := &blockingObserver{block: make(chan struct{})}
					a := NewAuditor(newLogger(), obs)

					done := make(chan struct{})
					go func() {
						for i := 0; i < defaultChannelBuffer+10; i++ {
							a.Audit(nil, "test", "http://example.com")
						}
						close(done)
					}()

					select {
					case <-done:
					case <-time.After(time.Second):
						t.Fatal("Audit blocked when channel was full")
					}

					close(obs.block)
					a.Shutdown()
				},
			)

			t.Run(
				"Shutdown не должен паниковать при повторном вызове", func(t *testing.T) {
					obs := &blockingObserver{block: make(chan struct{})}
					close(obs.block)

					a := NewAuditor(newLogger(), obs)
					a.Shutdown()
					assert.NotPanics(t, func() { a.Shutdown() })
				},
			)
		},
	)
}
