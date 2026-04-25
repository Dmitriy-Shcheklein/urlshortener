package bootstrap

import (
	"context"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	pool "github.com/Dmitriy-Shcheklein/urlshortener/internal/config/db/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/shortener"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/file_storage"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	shService "github.com/Dmitriy-Shcheklein/urlshortener/internal/services/shortener"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/workers/delete_links_worker"
	"github.com/go-chi/chi"
)

func InitShortener(ctx context.Context, cfg *config.Config, pool *pool.Pool, router *chi.Mux) ([]func(), error) {
	var repository shService.LinkRepository
	if pool != nil {
		postgresRepo, err := postgres.New(pool)
		if err != nil {
			return nil, err
		}
		repository = postgresRepo
	} else {
		repository = file_storage.New(cfg)
	}

	svc := shService.New(repository)
	deleteWorker := delete_links_worker.New(svc)

	handler, err := shortener.New(svc, cfg)
	if err != nil {
		return nil, err
	}

	deleteWorker.Start(ctx)
	router.Post("/", handler.CreateShort)
	router.Get("/{id}", handler.GetByID)
	router.Post("/api/shorten", handler.CreateFromJSONBody)
	router.Post("/api/shorten/batch", handler.CreateMany)
	router.Get("/api/user/urls", handler.GetByUserID)

	return []func(){deleteWorker.Stop}, nil
}
