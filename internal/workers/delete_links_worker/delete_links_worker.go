package delete_links_worker

import (
	"context"
	"sync"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog/log"
)

type Service interface {
	Delete(values []*model.LinkToDelete) error
}

type DeleteLinksWorker struct {
	semaphore  chan chan *model.LinkToDelete
	mainQueue  chan *model.LinkToDelete
	cancelFunc context.CancelFunc
	service    Service
	wg         sync.WaitGroup
}

func New(service Service) *DeleteLinksWorker {
	semLength := 10
	semaphore := make(chan chan *model.LinkToDelete, semLength)
	for range semLength {
		semaphore <- make(chan *model.LinkToDelete)
	}

	return &DeleteLinksWorker{
		semaphore: semaphore, service: service, mainQueue: make(chan *model.LinkToDelete, 1_000), wg: sync.WaitGroup{},
	}
}

func (d *DeleteLinksWorker) AddToQueue(urls []string, userID string) {
	out := make(chan *model.LinkToDelete, len(urls))

	wg := sync.WaitGroup{}

	for _, link := range urls {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			out <- &model.LinkToDelete{
				UserID: userID,
				Link:   link,
			}
		}(link)

	}

	go func() {
		wg.Wait()
		close(out)

	}()

	d.semaphore <- out
}

func (d *DeleteLinksWorker) Start(ctx context.Context) {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	d.cancelFunc = cancel

	d.wg.Add(2)
	go d.fanInLinks(ctxWithCancel)
	go d.removeLinks(ctxWithCancel)

}

func (d *DeleteLinksWorker) fanInLinks(ctx context.Context) {
	defer d.wg.Done()
	defer close(d.mainQueue)

	wg := sync.WaitGroup{}

	for q := range d.semaphore {
		wg.Add(1)
		go func(in chan *model.LinkToDelete) {
			defer wg.Done()
			for v := range in {
				select {
				case <-ctx.Done():
					return
				case d.mainQueue <- v:
				}
			}
		}(q)
	}
	wg.Wait()
}

func (d *DeleteLinksWorker) removeLinks(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(time.Second)
	acc := make([]*model.LinkToDelete, 0)

	flush := func() {
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
		case <-ctx.Done():
			flush()
			return
		case value, ok := <-d.mainQueue:
			if !ok {
				flush()
				return
			}
			acc = append(acc, value)
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

func (d *DeleteLinksWorker) Stop() {
	if d.cancelFunc != nil {
		d.cancelFunc()
	}
	d.wg.Wait()
	close(d.semaphore)
}
