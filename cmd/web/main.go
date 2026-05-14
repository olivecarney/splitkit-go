package main

import (
	"log"
	"net/http"
	"os"

	"github.com/olivercarney/splitkit-go/internal/app"
	"github.com/olivercarney/splitkit-go/internal/models"
	"github.com/olivercarney/splitkit-go/internal/store"
)

func main() {
	port := getenv("PORT", "8080")

	server := app.NewServer(app.Config{
		Addr:          ":" + port,
		Store:         store.NewMemoryStore(),
		DevUser:       models.User{ID: "local-user"},
		StaticDir:     "static",
		TemplatesGlob: "internal/views/*.html",
	})

	log.Printf("splitkit listening on http://localhost:%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
