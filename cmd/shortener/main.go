package main

import (
	"context"
	"net/http"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/service"
	"github.com/stoolap/stoolap-go"
)

func main() {
	db := initDB()

	handlers := handler.New(service.New(repository.New(db)))

	mux := http.NewServeMux()

	mux.HandleFunc("POST /", handlers.CreateShort)
	mux.HandleFunc("GET /{id}", handlers.GetByd)

	err := http.ListenAndServe(":8080", mux)
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
