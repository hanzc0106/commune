package app

import (
	"context"
	"testing"

	"github.com/hanzc0106/commune/apps/api/internal/db"
	"github.com/hanzc0106/commune/apps/api/internal/db/queries"
	"github.com/hanzc0106/commune/apps/api/internal/testutil"
)

func TestInitializeCreatesSettingsAdminAndSession(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()

	migrations, err := db.LoadMigrations("../../migrations")
	if err != nil {
		t.Fatalf("LoadMigrations returned error: %v", err)
	}
	if err := db.RunMigrations(ctx, pool, migrations); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	_, _ = pool.Exec(ctx, "TRUNCATE sessions, members, app_settings RESTART IDENTITY CASCADE")

	service := NewService(pool)
	result, token, err := service.Initialize(ctx, InitializeInput{
		HouseholdName: "Han Home",
		AdminName:     "Han",
		PIN:           "123456",
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	if result.Member.Name != "Han" {
		t.Fatalf("member name = %q", result.Member.Name)
	}
	if result.Member.Role != "admin" {
		t.Fatalf("member role = %q", result.Member.Role)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	settings, err := queries.New(pool).GetAppSettings(ctx)
	if err != nil {
		t.Fatalf("GetAppSettings returned error: %v", err)
	}
	if settings.HouseholdName != "Han Home" {
		t.Fatalf("household name = %q", settings.HouseholdName)
	}
}

func TestLoginCreatesSessionForCorrectPIN(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()

	migrations, err := db.LoadMigrations("../../migrations")
	if err != nil {
		t.Fatalf("LoadMigrations returned error: %v", err)
	}
	if err := db.RunMigrations(ctx, pool, migrations); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}
	_, _ = pool.Exec(ctx, "TRUNCATE sessions, members, app_settings RESTART IDENTITY CASCADE")

	service := NewService(pool)
	initResult, _, err := service.Initialize(ctx, InitializeInput{
		HouseholdName: "Han Home",
		AdminName:     "Han",
		PIN:           "123456",
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	loginResult, token, err := service.Login(ctx, LoginInput{
		MemberID: initResult.Member.ID,
		PIN:      "123456",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if loginResult.Member.ID != initResult.Member.ID {
		t.Fatalf("login member ID = %q, want %q", loginResult.Member.ID, initResult.Member.ID)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
}
