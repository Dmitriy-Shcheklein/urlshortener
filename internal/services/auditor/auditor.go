// Package auditor provides an event-driven audit logging system.
// It dispatches audit messages to registered observers asynchronously.
package auditor

import (
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
)

// Observer defines the interface for audit event consumers.
// Observers receive audit messages for processing (e.g., writing to file or HTTP endpoint).
type Observer interface {
	// HandleMessage processes an audit event message.
	HandleMessage(msg model.AuditMsg)
}

// NewAuditor creates a new Auditor with the given logger.
func NewAuditor(logger *zerolog.Logger) *Auditor {
	return &Auditor{logger: logger}
}

// WithObserver registers an observer to receive audit events.
// Multiple observers can be registered; each receives every event concurrently.
func (a *Auditor) WithObserver(observer Observer) {
	if a.observers == nil {
		a.observers = []Observer{observer}
	} else {
		a.observers = append(a.observers, observer)
	}
}

// Auditor dispatches audit events to registered observers.
// It implements the Observer pattern and sends messages to all observers concurrently.
type Auditor struct {
	logger    *zerolog.Logger
	observers []Observer
}

// Audit sends an audit event to all registered observers asynchronously.
// userID may be nil for anonymous actions (e.g., following a link).
func (a *Auditor) Audit(userID *string, action string, URL string) {
	if len(a.observers) == 0 {
		return
	}
	msg := model.AuditMsg{Ts: time.Now().Unix(), UserID: userID, Action: action, URL: URL}
	for i := range a.observers {
		go func() {
			a.observers[i].HandleMessage(msg)
		}()
	}
}
