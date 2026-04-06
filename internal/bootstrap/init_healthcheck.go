package bootstrap

import (
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	pool "github.com/Dmitriy-Shcheklein/urlshortener/internal/config/db/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/healthcheck"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	hcService "github.com/Dmitriy-Shcheklein/urlshortener/internal/services/healthcheck"
)

func BootstrapHealthcheck(cfg *config.Config, pool *pool.Pool) (*healthcheck.Handler, error) {
	pgRepo, err := postgres.New(pool)
	if err != nil {
		return &healthcheck.Handler{}, err
	}
	hcs, err := hcService.New(pgRepo)
	if err != nil {
		return &healthcheck.Handler{}, err
	}

	hcHandlers, err := healthcheck.New(hcs)
	if err != nil {
		return &healthcheck.Handler{}, err
	}
	return hcHandlers, err
}
