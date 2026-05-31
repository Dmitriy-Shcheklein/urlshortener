package bootstrap

import (
	"context"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/shortener"
	pool "github.com/Dmitriy-Shcheklein/urlshortener/internal/infrastructure/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/fs"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	shService "github.com/Dmitriy-Shcheklein/urlshortener/internal/services/shortener"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/workers/deletelinks"
	"github.com/go-chi/chi"
)

type InitResult struct {
	ErrChannel chan error
	Shutdowns  []func()
}

func InitShortener(ctx context.Context, cfg *config.Config, pool *pool.Pool, router *chi.Mux) (*InitResult, error) {
	var repository shService.LinkRepository
	if pool != nil {
		postgresRepo, err := postgres.New(pool)
		if err != nil {
			return nil, err
		}
		repository = postgresRepo
	} else {
		repository = fs.New(cfg)
	}

	svc := shService.New(repository)
	deleteWorker := deletelinks.New(svc)

	handler, err := shortener.New(svc, cfg, deleteWorker, middlewares.NewAuthService())
	if err != nil {
		return nil, err
	}

	errChan := deleteWorker.Start(ctx)
	router.Post("/", handler.CreateShort)
	router.Get("/{id}", handler.GetByID)
	router.Post("/api/shorten", handler.CreateFromJSONBody)
	router.Post("/api/shorten/batch", handler.CreateMany)
	router.Get("/api/user/urls", handler.GetByUserID)
	router.Delete("/api/user/urls", handler.DeleteLinks)

	return &InitResult{ErrChannel: errChan, Shutdowns: []func(){deleteWorker.Stop}}, nil
}
