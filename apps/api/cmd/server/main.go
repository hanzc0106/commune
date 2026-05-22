package main

import (
	"context"
	"log"
	stdhttp "net/http"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/app"
	"github.com/hanzc0106/commune/apps/api/internal/config"
	"github.com/hanzc0106/commune/apps/api/internal/db"
	apphttp "github.com/hanzc0106/commune/apps/api/internal/http"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	service := app.NewService(pool)
	handler := apphttp.NewHandler(apphttp.Options{
		StaticDir:  cfg.StaticDir,
		APIHandler: apphttp.NewAPI(service),
	})

	log.Printf("commune api listening on %s", cfg.HTTPAddr)
	if err := stdhttp.ListenAndServe(cfg.HTTPAddr, handler); err != nil {
		log.Fatal(err)
	}
}
