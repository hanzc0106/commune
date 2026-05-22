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

	if _, err := pool.Exec(ctx, "TRUNCATE transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE returned error: %v", err)
	}

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

func TestInitializeCreatesDefaultCategories(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	migrations, err := db.LoadMigrations("../../migrations")
	if err != nil {
		t.Fatalf("LoadMigrations returned error: %v", err)
	}
	if err := db.RunMigrations(ctx, pool, migrations); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE returned error: %v", err)
	}

	service := NewService(pool)
	if _, _, err := service.Initialize(ctx, InitializeInput{
		HouseholdName: "Han Home",
		AdminName:     "Han",
		PIN:           "123456",
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	categories, err := service.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories returned error: %v", err)
	}
	if len(categories) != 10 {
		t.Fatalf("category count = %d, want 10", len(categories))
	}

	expected := map[string]bool{
		"expense/餐饮":   false,
		"expense/其他支出": false,
		"income/工资":    false,
		"income/其他收入":  false,
	}
	seen := make(map[string]bool, len(categories))
	for _, category := range categories {
		key := category.Type + "/" + category.Name
		if seen[key] {
			t.Fatalf("duplicate category: %s", key)
		}
		seen[key] = true

		if !category.SystemDefault {
			t.Fatalf("category is not system default: %+v", category)
		}
		if category.Name == "" || category.Type == "" || category.IconKey == "" || category.ColorKey == "" {
			t.Fatalf("category is incomplete: %+v", category)
		}
		if category.Type != "expense" && category.Type != "income" {
			t.Fatalf("category type = %q, want expense or income", category.Type)
		}
		if _, ok := expected[key]; ok {
			expected[key] = true
		}
	}
	for key, found := range expected {
		if !found {
			t.Fatalf("missing expected category: %s", key)
		}
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
	if _, err := pool.Exec(ctx, "TRUNCATE transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE returned error: %v", err)
	}

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
