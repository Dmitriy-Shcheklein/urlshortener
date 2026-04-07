package bootstrap

import (
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	pool "github.com/Dmitriy-Shcheklein/urlshortener/internal/config/db/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/shortener"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/file_storage"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	shService "github.com/Dmitriy-Shcheklein/urlshortener/internal/services/shortener"
	"github.com/go-chi/chi"
)

func InitShortener(cfg *config.Config, pool *pool.Pool, router *chi.Mux) error {
	var repository shService.LinkRepository
	if cfg.DbDSN.IsValid {
		postgresRepo, err := postgres.New(pool)
		if err != nil {
			return err
		}
		repository = postgresRepo
	} else {
		repository = file_storage.New(cfg)
	}

	handler, err := shortener.New(shService.New(repository), cfg)
	if err != nil {
		return err
	}

	router.Post("/", handler.CreateShort)
	router.Get("/{id}", handler.GetByID)
	router.Post("/api/shorten", handler.CreateFromJSONBody)

	return nil
}
