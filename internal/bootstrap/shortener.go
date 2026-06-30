// Package bootstrap provides application initialization and dependency wiring.
// It assembles the URL shortener service by creating repositories, services,
// handlers, and registering HTTP routes.
package bootstrap

import (
	"context"
	"net/http"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/shortener"
	pool "github.com/Dmitriy-Shcheklein/urlshortener/internal/infrastructure/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/fs"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/services/auditor"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/services/auditor/fsobserver"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/services/auditor/httpobserver"
	shService "github.com/Dmitriy-Shcheklein/urlshortener/internal/services/shortener"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/workers/deletelinks"
	"github.com/go-chi/chi"
)

// InitResult holds the results of shortener initialization, including an error
// channel for background worker errors and shutdown functions for cleanup.
type InitResult struct {
	// ErrChannel receives errors from background workers.
	ErrChannel chan error
	// Shutdowns contains cleanup functions to call on application shutdown.
	Shutdowns []func()
}

// InitShortener initializes the URL shortener service and registers all HTTP
// routes on the given router. It uses the PostgreSQL pool if provided; otherwise,
// it falls back to filesystem storage.
//
// Registered routes:
//   - POST /                     - Create short URL (text/plain)
//   - GET /{id}                  - Redirect to original URL
//   - POST /api/shorten          - Create short URL (JSON)
//   - POST /api/shorten/batch    - Batch create short URLs (JSON)
//   - GET /api/user/urls         - List user's URLs
//   - DELETE /api/user/urls      - Delete user's URLs
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
	observers := make([]auditor.Observer, 0)

	if cfg.GetAuditFilePath() != "" {
		fsObs, err := fsobserver.New(logger.Logger, cfg.GetAuditFilePath())
		if err != nil {
			return nil, err
		}
		observers = append(observers, fsObs)
	}
	if cfg.GetAuditUrl() != "" {
		observers = append(
			observers, httpobserver.New(
				logger.Logger, &http.Client{Timeout: time.Second * 10}, cfg.GetAuditUrl(),
			),
		)
	}
	appAuditor := auditor.NewAuditor(logger.Logger, observers...)

	svc := shService.New(repository)
	deleteWorker := deletelinks.New(svc)

	handler, err := shortener.New(svc, cfg, deleteWorker, middlewares.NewAuthService(), appAuditor)
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

	return &InitResult{ErrChannel: errChan, Shutdowns: []func(){deleteWorker.Stop, appAuditor.Shutdown}}, nil
}
