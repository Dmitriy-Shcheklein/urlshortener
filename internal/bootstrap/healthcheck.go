package bootstrap

import (
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	pool "github.com/Dmitriy-Shcheklein/urlshortener/internal/config/db/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/healthcheck"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	hcService "github.com/Dmitriy-Shcheklein/urlshortener/internal/services/healthcheck"
	"github.com/go-chi/chi"
)

func InitHealthcheck(_ *config.Config, pool *pool.Pool, router *chi.Mux) error {
	pgRepo, err := postgres.New(pool)
	if err != nil {
		return err
	}
	hcs, err := hcService.New(pgRepo)
	if err != nil {
		return err
	}

	hcHandlers, err := healthcheck.New(hcs)
	if err != nil {
		return err
	}

	router.Get("/ping", hcHandlers.PingDB)

	return err
}
