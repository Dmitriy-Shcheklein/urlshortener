package delete_links_worker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog/log"
)

type Service interface {
	Delete(values []*model.LinkToDelete) error
}

type DeleteLinksWorker struct {
	mainQueue   chan *model.LinkToDelete
	cancelFunc  context.CancelFunc
	service     Service
	wg          sync.WaitGroup
	ctx         context.Context
	inWg        sync.WaitGroup
	stopOnce    sync.Once
	initialized atomic.Bool
	accMu       sync.Mutex
}

func New(service Service) *DeleteLinksWorker {
	worker := &DeleteLinksWorker{
		service: service, mainQueue: make(chan *model.LinkToDelete),
	}

	worker.initialized.Store(false)
	return worker
}

func (d *DeleteLinksWorker) AddToQueue(urls []string, userID string) {
	if d.ctx == nil {
		log.Error().Msg("ctx is nil")
		return
	}

	d.inWg.Add(1)
	go func() {
		defer d.inWg.Done()
		for _, value := range urls {
			select {
			case <-d.ctx.Done():
				return
			case d.mainQueue <- &model.LinkToDelete{Link: value, UserID: userID}:
			}
		}
	}()
}

func (d *DeleteLinksWorker) Start(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Stack().
				Msg("panic in removeLinks")
			d.wg.Done()
		}
	}()
	if d.initialized.Load() {
		return
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	d.ctx = ctxWithCancel
	d.cancelFunc = func() {
		cancel()
	}

	d.wg.Add(1)
	go d.removeLinks()
	d.initialized.Store(true)
}

func (d *DeleteLinksWorker) removeLinks() {
	defer d.wg.Done()

	ticker := time.NewTicker(time.Second)

	acc := make([]*model.LinkToDelete, 0)

	flush := func() {
		d.accMu.Lock()
		defer d.accMu.Unlock()
		if len(acc) == 0 {
			return
		}
		if err := d.service.Delete(acc); err != nil {
			log.Error().Err(err).Msg("error while delete links")
			return
		}
		acc = acc[:0]
	}

	for {
		select {
		case <-d.ctx.Done():
			flush()
			return
		case value, ok := <-d.mainQueue:

			if !ok {
				flush()
				return
			}
			d.addToAcc(&acc, value)
			if len(acc) > 99 {
				flush()
			}
		case <-ticker.C:
			if len(acc) > 0 {
				flush()
			}
		}
	}
}

func (d *DeleteLinksWorker) addToAcc(acc *[]*model.LinkToDelete, val *model.LinkToDelete) {
	d.accMu.Lock()
	*acc = append(*acc, val)
	d.accMu.Unlock()
}

func (d *DeleteLinksWorker) Stop() {
	if d.cancelFunc != nil {
		d.cancelFunc()
	}
	d.inWg.Wait()
	d.wg.Wait()
	d.stopOnce.Do(func() { close(d.mainQueue) })
}
