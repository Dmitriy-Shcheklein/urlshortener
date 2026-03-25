package main

import (
	"context"
	"net/http"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/service"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stoolap/stoolap-go"
)

func main() {
	db := initDB()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	handlers := handler.New(service.New(repository.New(db)))
	router.Post("/", handlers.CreateShort)
	router.Get("/{id}", handlers.GetByd)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}

func initDB() *stoolap.DB {
	db, err := stoolap.Open("memory://")
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	_, err = db.Exec(
		ctx,
		"CREATE TABLE links (id INTEGER PRIMARY KEY AUTO_INCREMENT, url TEXT NOT NULL, short TEXT NOT NULL UNIQUE)",
	)
	db.Exec(ctx, "INSERT INTO links (url, short) VALUES ('long_url', 'EwHXdJfB')")
	if err != nil {
		panic(err)
	}
	return db
}
