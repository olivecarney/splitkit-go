package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olivecarney/splitkit-go/internal/app"
	"github.com/olivecarney/splitkit-go/internal/db"
	"github.com/olivecarney/splitkit-go/internal/models"
	"github.com/olivecarney/splitkit-go/internal/store"
)

func main() {
	ctx := context.Background()
	port := getenv("PORT", "8080")
	databaseURL := getenv("DATABASE_URL", "postgres://splitkit:splitkit@localhost:5432/splitkit?sslmode=disable")
	devUserID := getenv("DEV_USER_ID", "00000000-0000-0000-0000-000000000001")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatal(err)
	}

	server := app.NewServer(app.Config{
		Addr:          ":" + port,
		Store:         store.NewPostgresStore(pool),
		DevUser:       models.User{ID: devUserID, Email: "dev@splitkit.local"},
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
