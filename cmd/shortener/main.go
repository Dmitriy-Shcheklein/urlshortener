package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof" // nolint:gosec  // Только для отладки в dev-окружении
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/bootstrap"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/infrastructure/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
)

func main() {
	baseCancelCtx, cancelFunc := context.WithCancel(context.Background())

	cfg, err := config.New()
	if err != nil {
		log.Fatalf("error while getting config: %s", err)
	}

	logger.InitLogger(zerolog.InfoLevel)

	appMiddleware := middlewares.NewAppMiddleware(logger.Logger, cfg)

	router := chi.NewRouter()
	router.Use(middleware.Timeout(time.Minute))
	router.Use(appMiddleware.WithGzip)
	router.Use(appMiddleware.Auth)
	router.Use(appMiddleware.WithLogging)
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Mount("/debug/pprof", http.DefaultServeMux)

	var dbPool *postgres.Pool

	dsnValue := cfg.GetDSN()
	if dsnValue != "" {
		dbPool, err = postgres.NewPool(dsnValue)
		if err != nil {
			log.Fatalf("error while creating pool: %s", err)
		}
		defer func() {
			if dbPool != nil {
				dbPool.Stop()
			}
		}()
		if err = bootstrap.RunMigration(dsnValue); err != nil {
			log.Fatalf("error while execute migrations: %s", err)
		}
		if err = bootstrap.InitHealthcheck(cfg, dbPool, router); err != nil {
			log.Fatalf("error while bootstrap healthcheck handler: %s", err)
		}
	}
	initResult, err := bootstrap.InitShortener(baseCancelCtx, cfg, dbPool, router)
	if err != nil {
		log.Fatalf("error while create handlers: %s", err)
	}

	server := &http.Server{
		Addr:              cfg.GetNetAddress(),
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server stopped: %v", err)
		}
	}()

	log.Println("Server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case <-initResult.ErrChannel:
	}
	log.Println("Shutting down server...")
	cancelFunc()
	shutdown(server, initResult)

	log.Println("Server exiting")
}

func shutdown(server *http.Server, initResult *bootstrap.InitResult) {
	wg := sync.WaitGroup{}
	if len(initResult.Shutdowns) != 0 {
		for _, fn := range initResult.Shutdowns {
			wg.Add(1)
			go func(fn func()) {
				defer wg.Done()
				fn()
			}(fn)
		}
	}
	wg.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		if err = server.Close(); err != nil {
			log.Printf("Server close error: %v", err)
		}
	} else {
		log.Println("Server stopped gracefully")
	}
}
