package main

import (
	"log"
	stdhttp "net/http"

	"github.com/hanzc0106/commune/apps/api/internal/config"
	apphttp "github.com/hanzc0106/commune/apps/api/internal/http"
)

func main() {
	cfg := config.Load()
	handler := apphttp.NewHandler(apphttp.Options{
		StaticDir: cfg.StaticDir,
	})

	log.Printf("commune api listening on %s", cfg.HTTPAddr)
	if err := stdhttp.ListenAndServe(cfg.HTTPAddr, handler); err != nil {
		log.Fatal(err)
	}
}
