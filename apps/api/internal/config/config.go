package config

import "os"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	StaticDir   string
}

func Load() Config {
	return Config{
		HTTPAddr:    getEnv("COMMUNE_HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("COMMUNE_DATABASE_URL", "postgres://commune:commune@localhost:5432/commune?sslmode=disable"),
		StaticDir:   getEnv("COMMUNE_STATIC_DIR", "../web/dist"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
