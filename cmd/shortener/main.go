package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/service"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stoolap/stoolap-go"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("error while getting config: %s", err)
	}

	db, err := initDB()
	if err != nil {
		log.Fatalf("error while getting db: %s", err)
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	handlers := handler.New(service.New(repository.New(db)), cfg)
	router.Post("/", handlers.CreateShort)
	router.Get("/{id}", handlers.GetByd)

	err = http.ListenAndServe(cfg.GetNetAddress(), router)
	if err != nil {
		log.Fatalf("error while start server: %s", err)
	}
}

func initDB() (*stoolap.DB, error) {
	db, err := stoolap.Open("memory://")
	if err != nil {
		return db, err
	}

	ctx := context.Background()

	_, err = db.Exec(
		ctx,
		"CREATE TABLE links (id INTEGER PRIMARY KEY AUTO_INCREMENT, url TEXT NOT NULL, short TEXT NOT NULL UNIQUE)",
	)
	db.Exec(ctx, "INSERT INTO links (url, short) VALUES ('long_url', 'EwHXdJfB')")
	if err != nil {
		return db, err
	}
	return db, nil
}
