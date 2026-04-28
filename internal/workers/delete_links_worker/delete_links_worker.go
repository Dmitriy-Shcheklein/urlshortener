package delete_links_worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
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
	initialized atomic.Bool
	accMu       sync.Mutex
	timeout     time.Duration
}

func New(service Service) *DeleteLinksWorker {
	worker := &DeleteLinksWorker{
		service: service, mainQueue: make(chan *model.LinkToDelete),
		timeout: time.Second,
	}

	worker.initialized.Store(false)
	return worker
}

func (d *DeleteLinksWorker) AddToQueue(urls []string, userID string) {
	for _, value := range urls {
		d.mainQueue <- &model.LinkToDelete{Link: value, UserID: userID}
	}
}

func (d *DeleteLinksWorker) Start(ctx context.Context) chan error {
	if d.initialized.Load() {
		return nil
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	d.cancelFunc = func() {
		cancel()
	}

	errChan := make(chan error)
	d.wg.Add(1)
	go d.removeLinks(ctxWithCancel, errChan)
	d.initialized.Store(true)
	return errChan
}

func (d *DeleteLinksWorker) removeLinks(ctx context.Context, errChan chan error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error().Interface("panic", r).Stack().
				Msg("panic in removeLinks")
			errChan <- fmt.Errorf("panic: %v", r)
		}
	}()
	defer d.wg.Done()

	ticker := time.NewTicker(d.timeout)

	acc := make([]*model.LinkToDelete, 0)

	for {
		select {
		case <-ctx.Done():
			d.flush(&acc)
			return
		case value, ok := <-d.mainQueue:
			if !ok {
				d.flush(&acc)
				return
			}
			d.addToAcc(&acc, value)
			if len(acc) > 99 {
				d.flush(&acc)
			}
		case <-ticker.C:
			if len(acc) > 0 {
				d.flush(&acc)
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
	d.wg.Wait()
}

func (d *DeleteLinksWorker) flush(acc *[]*model.LinkToDelete) {
	d.accMu.Lock()
	defer d.accMu.Unlock()
	if len(*acc) == 0 {
		return
	}
	if err := d.service.Delete(*acc); err != nil {
		log.Error().Err(err).Msg("error while delete links")
		return
	}
	*acc = (*acc)[:0]
}
