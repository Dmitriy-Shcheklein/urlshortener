// Package deletelinks provides a background worker for batch-deleting shortened URLs.
// It accumulates deletion requests and processes them in batches to reduce database load.
package deletelinks

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

// Service defines the deletion operations used by the worker.
type Service interface {
	// Delete marks the specified links as deleted.
	Delete(values []*model.LinkToDelete) error
}

// DeleteLinksWorker is a background worker that accumulates link deletion requests
// and processes them in batches. It flushes accumulated items every second or when
// the batch size reaches 100 items.
//
// Use [New] to create an instance and [Start] to begin processing.
// Call [Stop] to gracefully shut down the worker.
type DeleteLinksWorker struct {
	mainQueue   chan *model.LinkToDelete
	cancelFunc  context.CancelFunc
	service     Service
	wg          sync.WaitGroup
	initialized atomic.Bool
	accMu       sync.Mutex
	timeout     time.Duration
}

// New creates a new DeleteLinksWorker backed by the given service.
// The worker must be started with [Start] before it can process deletions.
func New(service Service) *DeleteLinksWorker {
	worker := &DeleteLinksWorker{
		service: service, mainQueue: make(chan *model.LinkToDelete),
		timeout: time.Second,
	}

	worker.initialized.Store(false)
	return worker
}

// AddToQueue enqueues a batch of short URL identifiers for deletion under the
// given user ID. This method is safe for concurrent use.
func (d *DeleteLinksWorker) AddToQueue(urls []string, userID string) {
	for _, value := range urls {
		d.mainQueue <- &model.LinkToDelete{Link: value, UserID: userID}
	}
}

// Start begins the background deletion processing. It returns an error channel
// that will receive any errors encountered during processing.
// Returns nil if the worker is already running. The worker stops when the
// provided context is cancelled.
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

// Stop gracefully shuts down the worker by cancelling the processing context
// and waiting for all pending operations to complete.
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
