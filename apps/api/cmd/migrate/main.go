package main

import (
	"context"
	"log"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/config"
	"github.com/hanzc0106/commune/apps/api/internal/db"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	migrations, err := db.LoadMigrations("migrations")
	if err != nil {
		log.Fatal(err)
	}

	if err := db.RunMigrations(ctx, pool, migrations); err != nil {
		log.Fatal(err)
	}
	log.Printf("applied %d available migrations", len(migrations))
}
