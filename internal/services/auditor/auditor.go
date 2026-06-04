package auditor

import (
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
)

type Observer interface {
	HandleMessage(msg model.AuditMsg)
}

func NewAuditor(logger *zerolog.Logger) *Auditor {
	return &Auditor{logger: logger}
}

func (a *Auditor) WithObserver(observer Observer) {
	if a.observers == nil {
		a.observers = []Observer{observer}
	} else {
		a.observers = append(a.observers, observer)
	}
}

type Auditor struct {
	logger    *zerolog.Logger
	observers []Observer
}

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
