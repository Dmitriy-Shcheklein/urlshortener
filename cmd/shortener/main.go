package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/service"
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
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middlewares.WithGzip)

	handlers := handler.New(service.New(repository.New(cfg)), cfg)
	router.Post("/", handlers.CreateShort)
	router.Get("/{id}", handlers.GetByd)
	router.Post("/api/shorten", handlers.CreateFromJSONBody)

	err = http.ListenAndServe(cfg.GetNetAddress(), router)
	if err != nil {
		log.Fatalf("error while start server: %s", err)
	}
}
