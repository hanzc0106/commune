package db

import "testing"

func TestMigrationVersionFromFilename(t *testing.T) {
	version, err := migrationVersion("000002_auth_foundation.sql")
	if err != nil {
		t.Fatalf("migrationVersion returned error: %v", err)
	}
	if version != 2 {
		t.Fatalf("version = %d, want 2", version)
	}
}

func TestMigrationVersionRejectsInvalidFilename(t *testing.T) {
	_, err := migrationVersion("auth_foundation.sql")
	if err == nil {
		t.Fatal("migrationVersion returned nil error for invalid filename")
	}
}
