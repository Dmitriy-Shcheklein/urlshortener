// Package auditor provides an event-driven audit logging system.
// It dispatches audit messages to registered observers asynchronously.
package auditor

import (
	"sync"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
)

const (
	defaultWorkerCount   = 5
	defaultChannelBuffer = 256
)

// Observer defines the interface for audit event consumers.
// Observers receive audit messages for processing (e.g., writing to file or HTTP endpoint).
type Observer interface {
	// HandleMessage processes an audit event message.
	HandleMessage(msg model.AuditMsg)
}

type observerPool struct {
	observer Observer
	ch       chan model.AuditMsg
	wg       sync.WaitGroup
}

// NewAuditor creates a new Auditor with the given logger and observers.
// Each observer gets its own worker pool with a buffered channel.
func NewAuditor(logger *zerolog.Logger, observers ...Observer) *Auditor {
	a := &Auditor{logger: logger}
	for _, obs := range observers {
		p := &observerPool{
			observer: obs,
			ch:       make(chan model.AuditMsg, defaultChannelBuffer),
		}
		a.pools = append(a.pools, p)
	}
	for _, p := range a.pools {
		a.startWorkers(p)
	}
	return a
}

// Auditor dispatches audit events to registered observers via worker pools.
type Auditor struct {
	logger       *zerolog.Logger
	pools        []*observerPool
	shutdownOnce sync.Once
}

func (a *Auditor) startWorkers(p *observerPool) {
	for i := 0; i < defaultWorkerCount; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for msg := range p.ch {
				p.observer.HandleMessage(msg)
			}
		}()
	}
}

// Audit sends an audit event to all registered observers.
// userID may be nil for anonymous actions (e.g., following a link).
// Messages are dropped if the observer channel is full.
func (a *Auditor) Audit(userID *string, action string, URL string) {
	if len(a.pools) == 0 {
		return
	}
	msg := model.AuditMsg{Ts: time.Now().Unix(), UserID: userID, Action: action, URL: URL}
	for i := range a.pools {
		select {
		case a.pools[i].ch <- msg:
		default:
			a.logger.Warn().Msg("audit: observer channel full, dropping message")
		}
	}
}

// Shutdown gracefully shuts down all observer worker pools.
// It closes channels and waits for all workers to drain pending messages.
// Safe to call multiple times.
func (a *Auditor) Shutdown() {
	a.shutdownOnce.Do(
		func() {
			for i := range a.pools {
				close(a.pools[i].ch)
				a.pools[i].wg.Wait()
			}
		},
	)
}
