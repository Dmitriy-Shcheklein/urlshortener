package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/bootstrap"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	pool "github.com/Dmitriy-Shcheklein/urlshortener/internal/config/db/postgres"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("error while getting config: %s", err)
	}

	logger.InitLogger(zerolog.InfoLevel)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middlewares.WithLogging)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(time.Minute))
	router.Use(middlewares.WithGzip)

	var dbPool *pool.Pool

	if cfg.DbDSN.IsValid {
		dbPool, err = pool.NewPool(cfg.DbDSN.Value)
		if err != nil && cfg.DbDSN.IsValid {
			log.Fatalf("error while creating pool: %s", err)
		}
		defer func() {
			if dbPool != nil {
				dbPool.Stop()
			}
		}()
		if err = bootstrap.RunMigration(cfg.DbDSN.Value); err != nil {
			log.Fatalf("error while execute migrations: %s", err)
		}
		if err = bootstrap.InitHealthcheck(cfg, dbPool, router); err != nil {
			log.Fatalf("error while bootstrap healthcheck handler: %s", err)
		}
	}
	if err = bootstrap.InitShortener(cfg, dbPool, router); err != nil {
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
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		if err = server.Close(); err != nil {
			log.Printf("Server close error: %v", err)
		}
	} else {
		log.Println("Server stopped gracefully")
	}

	log.Println("Server exiting")
}
