package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/app"
	"github.com/hanzc0106/commune/apps/api/internal/auth"
	"github.com/hanzc0106/commune/apps/api/internal/db"
	"github.com/hanzc0106/commune/apps/api/internal/testutil"
)

func TestLedgerAPIRequiresSession(t *testing.T) {
	handler := newInitializedAPI(t)
	req := httptest.NewRequest("GET", "/categories", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestCategoriesAPIReturnsCategoriesForSession(t *testing.T) {
	handler, cookie := newInitializedAPIWithCookie(t)
	req := httptest.NewRequest("GET", "/categories", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTransactionAndMonthlyOverviewAPI(t *testing.T) {
	handler, cookie := newInitializedAPIWithCookie(t)
	categoryID := firstCategoryID(t, handler, cookie, "expense")
	body := []byte(`{"type":"expense","amountCents":2300,"categoryId":"` + categoryID + `","transactionDate":"2026-05-23","note":"dinner"}`)
	createReq := httptest.NewRequest("POST", "/transactions", bytes.NewReader(body))
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()

	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body = %s", createRec.Code, createRec.Body.String())
	}

	overviewReq := httptest.NewRequest("GET", "/overview/monthly?month=2026-05", nil)
	overviewReq.AddCookie(cookie)
	overviewRec := httptest.NewRecorder()

	handler.ServeHTTP(overviewRec, overviewReq)

	if overviewRec.Code != http.StatusOK {
		t.Fatalf("overview status = %d, want 200, body = %s", overviewRec.Code, overviewRec.Body.String())
	}
	var overview struct {
		ExpenseCents int64 `json:"expenseCents"`
	}
	if err := json.NewDecoder(overviewRec.Body).Decode(&overview); err != nil {
		t.Fatalf("Decode overview returned error: %v", err)
	}
	if overview.ExpenseCents != 2300 {
		t.Fatalf("expense cents = %d, want 2300", overview.ExpenseCents)
	}
}

func newInitializedAPI(t *testing.T) http.Handler {
	t.Helper()
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
	service := app.NewService(pool)
	if _, _, err := service.Initialize(ctx, app.InitializeInput{
		HouseholdName: "Han Home",
		AdminName:     "Han",
		PIN:           "123456",
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	return NewAPI(service)
}

func newInitializedAPIWithCookie(t *testing.T) (http.Handler, *http.Cookie) {
	t.Helper()
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
	service := app.NewService(pool)
	_, token, err := service.Initialize(ctx, app.InitializeInput{
		HouseholdName: "Han Home",
		AdminName:     "Han",
		PIN:           "123456",
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	return NewAPI(service), auth.SessionCookie(token, time.Now().Add(30*24*time.Hour))
}

func firstCategoryID(t *testing.T, handler http.Handler, cookie *http.Cookie, categoryType string) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/categories", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("categories status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Categories []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"categories"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Decode categories returned error: %v", err)
	}
	for _, category := range response.Categories {
		if category.Type == categoryType {
			return category.ID
		}
	}
	t.Fatalf("missing %s category", categoryType)
	return ""
}
