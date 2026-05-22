package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("COMMUNE_HTTP_ADDR", "")
	t.Setenv("COMMUNE_DATABASE_URL", "")
	t.Setenv("COMMUNE_STATIC_DIR", "")

	cfg := Load()

	if cfg.HTTPAddr != ":8090" {
		t.Fatalf("HTTPAddr = %q, want :8090", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://commune:commune@localhost:5432/commune?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.StaticDir != "../web/dist" {
		t.Fatalf("StaticDir = %q, want ../web/dist", cfg.StaticDir)
	}
}

func TestLoadUsesEnvironment(t *testing.T) {
	t.Setenv("COMMUNE_HTTP_ADDR", ":9090")
	t.Setenv("COMMUNE_DATABASE_URL", "postgres://example")
	t.Setenv("COMMUNE_STATIC_DIR", "public")

	cfg := Load()

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q, want :9090", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("DatabaseURL = %q, want postgres://example", cfg.DatabaseURL)
	}
	if cfg.StaticDir != "public" {
		t.Fatalf("StaticDir = %q, want public", cfg.StaticDir)
	}
}
