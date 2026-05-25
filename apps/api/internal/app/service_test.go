package app

import (
	"context"
	"testing"

	"github.com/hanzc0106/commune/apps/api/internal/db"
	"github.com/hanzc0106/commune/apps/api/internal/db/queries"
	"github.com/hanzc0106/commune/apps/api/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
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

	if _, err := pool.Exec(ctx, "TRUNCATE budgets, transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, "TRUNCATE budgets, transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, "TRUNCATE budgets, transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"); err != nil {
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

func TestTransactionPermissions(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	service := newInitializedTestService(t, ctx, pool)

	admin, member := createTestMember(t, ctx, pool)
	categories, err := service.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories returned error: %v", err)
	}
	expenseCategory := findTestCategory(t, categories, "expense")

	adminTransaction, err := service.CreateTransaction(ctx, admin, CreateTransactionInput{
		Type:            "expense",
		AmountCents:     1280,
		CategoryID:      expenseCategory.ID,
		TransactionDate: "2026-05-22",
		Note:            "lunch",
	})
	if err != nil {
		t.Fatalf("CreateTransaction admin returned error: %v", err)
	}
	memberTransaction, err := service.CreateTransaction(ctx, member, CreateTransactionInput{
		Type:            "expense",
		AmountCents:     990,
		CategoryID:      expenseCategory.ID,
		TransactionDate: "2026-05-22",
		Note:            "snack",
	})
	if err != nil {
		t.Fatalf("CreateTransaction member returned error: %v", err)
	}

	if err := service.DeleteTransaction(ctx, member, adminTransaction.ID); err == nil {
		t.Fatal("member deleted another member transaction, want error")
	}
	if _, err := service.UpdateTransaction(ctx, member, adminTransaction.ID, UpdateTransactionInput{
		Type:            "expense",
		AmountCents:     1800,
		CategoryID:      expenseCategory.ID,
		TransactionDate: "2026-05-22",
		Note:            "member edit",
	}); err == nil {
		t.Fatal("member updated another member transaction, want error")
	}
	if _, err := service.UpdateTransaction(ctx, admin, memberTransaction.ID, UpdateTransactionInput{
		Type:            "expense",
		AmountCents:     1880,
		CategoryID:      expenseCategory.ID,
		TransactionDate: "2026-05-23",
		Note:            "admin edit",
	}); err != nil {
		t.Fatalf("admin UpdateTransaction returned error: %v", err)
	}
	if err := service.DeleteTransaction(ctx, admin, memberTransaction.ID); err != nil {
		t.Fatalf("admin DeleteTransaction returned error: %v", err)
	}
}

func TestCreateTransactionRejectsCategoryTypeMismatch(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	service := newInitializedTestService(t, ctx, pool)

	admin := firstTestMember(t, ctx, pool)
	categories, err := service.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories returned error: %v", err)
	}
	expenseCategory := findTestCategory(t, categories, "expense")
	incomeCategory := findTestCategory(t, categories, "income")

	_, err = service.CreateTransaction(ctx, admin, CreateTransactionInput{
		Type:            "expense",
		AmountCents:     100,
		CategoryID:      incomeCategory.ID,
		TransactionDate: "2026-05-22",
	})
	if err == nil {
		t.Fatal("expense transaction with income category succeeded, want error")
	}

	_, err = service.CreateTransaction(ctx, admin, CreateTransactionInput{
		Type:            "income",
		AmountCents:     100,
		CategoryID:      expenseCategory.ID,
		TransactionDate: "2026-05-22",
	})
	if err == nil {
		t.Fatal("income transaction with expense category succeeded, want error")
	}
}

func TestMonthlyOverviewSummarizesTransactions(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	service := newInitializedTestService(t, ctx, pool)

	admin := firstTestMember(t, ctx, pool)
	categories, err := service.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories returned error: %v", err)
	}
	expenseCategory := findTestCategory(t, categories, "expense")
	incomeCategory := findTestCategory(t, categories, "income")

	_, err = service.CreateTransaction(ctx, admin, CreateTransactionInput{
		Type:            "expense",
		AmountCents:     1200,
		CategoryID:      expenseCategory.ID,
		TransactionDate: "2026-05-10",
		Note:            "groceries",
	})
	if err != nil {
		t.Fatalf("CreateTransaction expense returned error: %v", err)
	}
	_, err = service.CreateTransaction(ctx, admin, CreateTransactionInput{
		Type:            "income",
		AmountCents:     5000,
		CategoryID:      incomeCategory.ID,
		TransactionDate: "2026-05-11",
		Note:            "salary",
	})
	if err != nil {
		t.Fatalf("CreateTransaction income returned error: %v", err)
	}
	_, err = service.CreateTransaction(ctx, admin, CreateTransactionInput{
		Type:            "expense",
		AmountCents:     700,
		CategoryID:      expenseCategory.ID,
		TransactionDate: "2026-06-01",
		Note:            "next month",
	})
	if err != nil {
		t.Fatalf("CreateTransaction next month returned error: %v", err)
	}

	overview, err := service.MonthlyOverview(ctx, "2026-05")
	if err != nil {
		t.Fatalf("MonthlyOverview returned error: %v", err)
	}
	if overview.IncomeCents != 5000 {
		t.Fatalf("income cents = %d, want 5000", overview.IncomeCents)
	}
	if overview.ExpenseCents != 1200 {
		t.Fatalf("expense cents = %d, want 1200", overview.ExpenseCents)
	}
	if overview.BalanceCents != 3800 {
		t.Fatalf("balance cents = %d, want 3800", overview.BalanceCents)
	}
	if len(overview.CategoryTotals) != 1 {
		t.Fatalf("category totals count = %d, want 1", len(overview.CategoryTotals))
	}
	if len(overview.Recent) != 2 {
		t.Fatalf("recent count = %d, want 2", len(overview.Recent))
	}
	if !overview.Recent[0].Category.SystemDefault {
		t.Fatalf("recent transaction category did not preserve system default flag: %+v", overview.Recent[0].Category)
	}
	if overview.Recent[0].Category.SortOrder == 0 {
		t.Fatalf("recent transaction category sort order = 0, want populated value")
	}
}

func TestSettingsAdminMemberManagement(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	service := newInitializedTestService(t, ctx, pool)
	admin := firstTestMember(t, ctx, pool)

	created, err := service.CreateMember(ctx, admin, CreateMemberInput{
		Name: "Li",
		Role: "member",
		PIN:  "654321",
	})
	if err != nil {
		t.Fatalf("CreateMember returned error: %v", err)
	}
	if created.Name != "Li" || created.Role != "member" || !created.Active {
		t.Fatalf("created member = %+v", created)
	}

	members, err := service.ListMembers(ctx, admin)
	if err != nil {
		t.Fatalf("ListMembers returned error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("member count = %d, want 2", len(members))
	}

	if err := service.ResetMemberPIN(ctx, admin, created.ID, ResetMemberPINInput{PIN: "111111"}); err != nil {
		t.Fatalf("ResetMemberPIN returned error: %v", err)
	}
	if _, _, err := service.Login(ctx, LoginInput{MemberID: created.ID, PIN: "111111"}); err != nil {
		t.Fatalf("Login with reset PIN returned error: %v", err)
	}

	disabled, err := service.DisableMember(ctx, admin, created.ID)
	if err != nil {
		t.Fatalf("DisableMember returned error: %v", err)
	}
	if disabled.Active {
		t.Fatalf("disabled member is active: %+v", disabled)
	}
	if _, _, err := service.Login(ctx, LoginInput{MemberID: created.ID, PIN: "111111"}); err == nil {
		t.Fatal("disabled member logged in, want error")
	}
}

func TestSettingsRejectsMemberAdminActions(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	service := newInitializedTestService(t, ctx, pool)
	_, member := createTestMember(t, ctx, pool)

	if _, err := service.ListMembers(ctx, member); err == nil {
		t.Fatal("member listed members, want error")
	}
	if _, err := service.CreateCategory(ctx, member, CreateCategoryInput{Name: "测试", Type: "expense"}); err == nil {
		t.Fatal("member created category, want error")
	}
}

func TestSettingsCannotDisableLastAdmin(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	service := newInitializedTestService(t, ctx, pool)
	admin := firstTestMember(t, ctx, pool)

	if _, err := service.DisableMember(ctx, admin, admin.ID); err == nil {
		t.Fatal("disabled last admin, want error")
	}
}

func TestChangeOwnPIN(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	service := newInitializedTestService(t, ctx, pool)
	admin := firstTestMember(t, ctx, pool)

	if err := service.ChangeOwnPIN(ctx, admin, ChangeOwnPINInput{
		CurrentPIN: "wrong",
		NewPIN:     "222222",
	}); err == nil {
		t.Fatal("ChangeOwnPIN accepted wrong current PIN")
	}
	if err := service.ChangeOwnPIN(ctx, admin, ChangeOwnPINInput{
		CurrentPIN: "123456",
		NewPIN:     "222222",
	}); err != nil {
		t.Fatalf("ChangeOwnPIN returned error: %v", err)
	}
	if _, _, err := service.Login(ctx, LoginInput{MemberID: admin.ID, PIN: "222222"}); err != nil {
		t.Fatalf("Login with changed PIN returned error: %v", err)
	}
}

func TestSettingsCategoryManagement(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	ctx := context.Background()
	service := newInitializedTestService(t, ctx, pool)
	admin := firstTestMember(t, ctx, pool)

	category, err := service.CreateCategory(ctx, admin, CreateCategoryInput{
		Name:     "宠物",
		Type:     "expense",
		IconKey:  "paw-print",
		ColorKey: "orange",
	})
	if err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if category.Name != "宠物" || category.Type != "expense" {
		t.Fatalf("category = %+v", category)
	}

	updated, err := service.UpdateCategory(ctx, admin, category.ID, UpdateCategoryInput{
		Name:      "宠物用品",
		IconKey:   "bone",
		ColorKey:  "amber",
		SortOrder: 120,
	})
	if err != nil {
		t.Fatalf("UpdateCategory returned error: %v", err)
	}
	if updated.Name != "宠物用品" || updated.SortOrder != 120 {
		t.Fatalf("updated category = %+v", updated)
	}

	if _, err := service.DisableCategory(ctx, admin, category.ID); err != nil {
		t.Fatalf("DisableCategory returned error: %v", err)
	}
	activeCategories, err := service.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories returned error: %v", err)
	}
	for _, active := range activeCategories {
		if active.ID == category.ID {
			t.Fatalf("disabled category returned in active list: %+v", active)
		}
	}
}

func newInitializedTestService(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *Service {
	t.Helper()
	migrations, err := db.LoadMigrations("../../migrations")
	if err != nil {
		t.Fatalf("LoadMigrations returned error: %v", err)
	}
	if err := db.RunMigrations(ctx, pool, migrations); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE budgets, transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"); err != nil {
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
	return service
}

func firstTestMember(t *testing.T, ctx context.Context, pool *pgxpool.Pool) MemberDTO {
	t.Helper()
	rows, err := queries.New(pool).ListActiveLoginMembers(ctx)
	if err != nil {
		t.Fatalf("ListActiveLoginMembers returned error: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("no test members")
	}
	return MemberDTO{
		ID:   rows[0].ID.String(),
		Name: rows[0].Name,
		Role: "admin",
	}
}

func createTestMember(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (MemberDTO, MemberDTO) {
	t.Helper()
	admin := firstTestMember(t, ctx, pool)
	member, err := queries.New(pool).CreateMember(ctx, queries.CreateMemberParams{
		Name:    "Member",
		PinHash: "test-pin-hash",
		Role:    "member",
	})
	if err != nil {
		t.Fatalf("CreateMember returned error: %v", err)
	}
	return admin, memberDTO(member.ID, member.Name, member.Role)
}

func findTestCategory(t *testing.T, categories []CategoryDTO, categoryType string) CategoryDTO {
	t.Helper()
	for _, category := range categories {
		if category.Type == categoryType {
			return category
		}
	}
	t.Fatalf("missing %s category", categoryType)
	return CategoryDTO{}
}
