# Commune Budget MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现按月、按支出分类的预算 MVP，让成员查看预算执行情况，让管理员设置预算并复制上月预算。

**Architecture:** 延续当前 monorepo 结构：PostgreSQL 迁移定义 `budgets` 表，sqlc 生成数据访问代码，`app.Service` 聚合预算业务规则和权限校验，HTTP 层只做 session、JSON 和状态码，React 单页应用替换 Budgets tab 占位页。预算页独立调用预算 API，不改现有月度概览 API。

**Tech Stack:** Go、Chi、pgx、sqlc、PostgreSQL、React、Vite、Tailwind CSS、PowerShell、Taskfile。

---

## 文件结构

- 创建：`apps/api/migrations/000004_budget_mvp.sql`  
  定义 `budgets` 表、唯一约束、金额约束、月份格式约束和索引。
- 创建：`apps/api/queries/budgets.sql`  
  定义预算查询、upsert、复制上月预算和分类支出汇总查询。
- 生成：`apps/api/internal/db/queries/budgets.sql.go`  
  由 sqlc 生成，不手写。
- 修改：`apps/api/internal/app/service.go`  
  增加预算 DTO、输入类型、service 方法和预算状态计算。
- 修改：`apps/api/internal/app/service_test.go`  
  增加预算 service 测试。
- 修改：`apps/api/internal/http/api.go`  
  增加预算路由和 handler。
- 修改：`apps/api/internal/http/api_test.go`  
  增加预算 HTTP 测试。
- 修改：`apps/web/src/api.ts`  
  增加预算类型和 API 函数。
- 修改：`apps/web/src/App.tsx`  
  替换 Budgets tab 占位页为预算管理页面。

## Task 1：数据库迁移和 sqlc 查询

**Files:**

- Create: `apps/api/migrations/000004_budget_mvp.sql`
- Create: `apps/api/queries/budgets.sql`
- Generate: `apps/api/internal/db/queries/budgets.sql.go`

- [ ] **Step 1: 写迁移文件**

创建 `apps/api/migrations/000004_budget_mvp.sql`：

```sql
CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    month TEXT NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id),
    amount_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT budgets_month_category_unique UNIQUE (month, category_id),
    CONSTRAINT budgets_month_format CHECK (month ~ '^[0-9]{4}-[0-9]{2}$'),
    CONSTRAINT budgets_amount_positive CHECK (amount_cents > 0)
);

CREATE INDEX IF NOT EXISTS budgets_month_idx ON budgets (month);
CREATE INDEX IF NOT EXISTS budgets_category_id_idx ON budgets (category_id);
```

- [ ] **Step 2: 写预算查询**

创建 `apps/api/queries/budgets.sql`：

```sql
-- name: ListBudgetsByMonth :many
SELECT id, month, category_id, amount_cents, created_at, updated_at
FROM budgets
WHERE month = $1
ORDER BY category_id;

-- name: UpsertBudget :one
INSERT INTO budgets (month, category_id, amount_cents)
VALUES ($1, $2, $3)
ON CONFLICT (month, category_id)
DO UPDATE SET amount_cents = EXCLUDED.amount_cents, updated_at = now()
RETURNING id, month, category_id, amount_cents, created_at, updated_at;

-- name: CopyPreviousBudgets :one
WITH inserted AS (
    INSERT INTO budgets (month, category_id, amount_cents)
    SELECT sqlc.arg(target_month)::text, b.category_id, b.amount_cents
    FROM budgets b
    JOIN categories c ON c.id = b.category_id
    WHERE b.month = sqlc.arg(source_month)
      AND c.active = TRUE
      AND c.type = 'expense'
    ON CONFLICT (month, category_id) DO NOTHING
    RETURNING id
)
SELECT count(*)::bigint AS copied_count
FROM inserted;

-- name: ListMonthlyBudgetSpending :many
SELECT
    t.category_id,
    COALESCE(SUM(t.amount_cents), 0)::bigint AS spent_cents
FROM transactions t
WHERE t.type = 'expense'
  AND t.transaction_date >= sqlc.arg(start_date)
  AND t.transaction_date < sqlc.arg(end_date)
GROUP BY t.category_id;
```

- [ ] **Step 3: 生成 sqlc 代码**

运行：

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
.\scripts\sqlc.ps1
```

预期：

- 生成 `apps/api/internal/db/queries/budgets.sql.go`
- 生成代码里存在 `ListBudgetsByMonth`、`UpsertBudget`、`CopyPreviousBudgets`、`ListMonthlyBudgetSpending`

- [ ] **Step 4: 更新测试清理语句**

在 `apps/api/internal/app/service_test.go` 和 `apps/api/internal/http/api_test.go` 中，把所有测试初始化里的：

```go
"TRUNCATE transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"
```

替换为：

```go
"TRUNCATE budgets, transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE"
```

这样新表不会污染包内测试。

- [ ] **Step 5: 验证迁移和 sqlc 编译**

运行：

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
Set-Location apps\api
go test ./internal/db/queries -count=1
go test ./internal/db -count=1
```

预期：两个命令退出码为 0。

- [ ] **Step 6: 提交**

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
git add apps/api/migrations/000004_budget_mvp.sql apps/api/queries/budgets.sql apps/api/internal/db/queries/budgets.sql.go apps/api/internal/app/service_test.go apps/api/internal/http/api_test.go
git commit -m "feat: add budget schema and queries"
```

## Task 2：Service 预算逻辑和测试

**Files:**

- Modify: `apps/api/internal/app/service.go`
- Modify: `apps/api/internal/app/service_test.go`

- [ ] **Step 1: 先写失败测试**

在 `apps/api/internal/app/service_test.go` 中新增测试。测试应覆盖：

```go
func TestBudgetSummaryIncludesUnsetExpenseCategories(t *testing.T) {
    pool := testutil.OpenTestDB(t)
    ctx := context.Background()
    service := newInitializedTestService(t, ctx, pool)

    summary, err := service.ListBudgets(ctx, "2026-05")
    if err != nil {
        t.Fatalf("ListBudgets returned error: %v", err)
    }
    if summary.Month != "2026-05" {
        t.Fatalf("month = %q, want 2026-05", summary.Month)
    }
    if len(summary.Items) != 8 {
        t.Fatalf("item count = %d, want 8 expense categories", len(summary.Items))
    }
    for _, item := range summary.Items {
        if item.Category.Type != "expense" {
            t.Fatalf("category type = %q, want expense", item.Category.Type)
        }
        if item.Status != "unset" {
            t.Fatalf("status = %q, want unset", item.Status)
        }
    }
}
```

```go
func TestBudgetSetAndStatusCalculation(t *testing.T) {
    pool := testutil.OpenTestDB(t)
    ctx := context.Background()
    service := newInitializedTestService(t, ctx, pool)
    admin := firstTestMember(t, ctx, pool)
    categories, err := service.ListCategories(ctx)
    if err != nil {
        t.Fatalf("ListCategories returned error: %v", err)
    }
    expenseCategory := findTestCategory(t, categories, "expense")

    if _, err := service.SetBudget(ctx, admin, "2026-05", expenseCategory.ID, SetBudgetInput{AmountCents: 10000}); err != nil {
        t.Fatalf("SetBudget returned error: %v", err)
    }
    if _, err := service.CreateTransaction(ctx, admin, CreateTransactionInput{
        Type: "expense", AmountCents: 8500, CategoryID: expenseCategory.ID, TransactionDate: "2026-05-10",
    }); err != nil {
        t.Fatalf("CreateTransaction returned error: %v", err)
    }

    summary, err := service.ListBudgets(ctx, "2026-05")
    if err != nil {
        t.Fatalf("ListBudgets returned error: %v", err)
    }
    item := findBudgetItem(t, summary.Items, expenseCategory.ID)
    if item.BudgetCents != 10000 || item.SpentCents != 8500 || item.RemainingCents != 1500 {
        t.Fatalf("budget item = %+v", item)
    }
    if item.UsagePercent != 85 {
        t.Fatalf("usage percent = %d, want 85", item.UsagePercent)
    }
    if item.Status != "near" {
        t.Fatalf("status = %q, want near", item.Status)
    }
}
```

```go
func TestBudgetRejectsIncomeCategoryAndMemberWrites(t *testing.T) {
    pool := testutil.OpenTestDB(t)
    ctx := context.Background()
    service := newInitializedTestService(t, ctx, pool)
    admin := firstTestMember(t, ctx, pool)
    member, err := service.CreateMember(ctx, admin, CreateMemberInput{Name: "Li", Role: "member", PIN: "234567"})
    if err != nil {
        t.Fatalf("CreateMember returned error: %v", err)
    }
    categories, err := service.ListCategories(ctx)
    if err != nil {
        t.Fatalf("ListCategories returned error: %v", err)
    }
    incomeCategory := findTestCategory(t, categories, "income")
    expenseCategory := findTestCategory(t, categories, "expense")

    if _, err := service.SetBudget(ctx, admin, "2026-05", incomeCategory.ID, SetBudgetInput{AmountCents: 10000}); err == nil {
        t.Fatal("SetBudget accepted income category, want error")
    }
    if _, err := service.SetBudget(ctx, MemberDTO{ID: member.ID, Name: member.Name, Role: member.Role}, "2026-05", expenseCategory.ID, SetBudgetInput{AmountCents: 10000}); err == nil {
        t.Fatal("SetBudget accepted member actor, want error")
    }
}
```

```go
func TestBudgetCopyPreviousDoesNotOverwriteCurrentMonth(t *testing.T) {
    pool := testutil.OpenTestDB(t)
    ctx := context.Background()
    service := newInitializedTestService(t, ctx, pool)
    admin := firstTestMember(t, ctx, pool)
    categories, err := service.ListCategories(ctx)
    if err != nil {
        t.Fatalf("ListCategories returned error: %v", err)
    }
    expenseCategory := findTestCategory(t, categories, "expense")

    if _, err := service.SetBudget(ctx, admin, "2026-04", expenseCategory.ID, SetBudgetInput{AmountCents: 10000}); err != nil {
        t.Fatalf("SetBudget previous returned error: %v", err)
    }
    if _, err := service.SetBudget(ctx, admin, "2026-05", expenseCategory.ID, SetBudgetInput{AmountCents: 5000}); err != nil {
        t.Fatalf("SetBudget current returned error: %v", err)
    }
    copied, err := service.CopyPreviousBudgets(ctx, admin, "2026-05")
    if err != nil {
        t.Fatalf("CopyPreviousBudgets returned error: %v", err)
    }
    if copied.CopiedCount != 0 {
        t.Fatalf("copied count = %d, want 0", copied.CopiedCount)
    }
    summary, err := service.ListBudgets(ctx, "2026-05")
    if err != nil {
        t.Fatalf("ListBudgets returned error: %v", err)
    }
    item := findBudgetItem(t, summary.Items, expenseCategory.ID)
    if item.BudgetCents != 5000 {
        t.Fatalf("budget cents = %d, want 5000", item.BudgetCents)
    }
}
```

同时新增 helper：

```go
func findBudgetItem(t *testing.T, items []BudgetStatusDTO, categoryID string) BudgetStatusDTO {
    t.Helper()
    for _, item := range items {
        if item.Category.ID == categoryID {
            return item
        }
    }
    t.Fatalf("missing budget item for category %s", categoryID)
    return BudgetStatusDTO{}
}
```

- [ ] **Step 2: 运行测试确认失败**

运行：

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
Set-Location apps\api
go test ./internal/app -run "TestBudget" -count=1
```

预期：编译失败，缺少 `ListBudgets`、`SetBudget`、`CopyPreviousBudgets`、`SetBudgetInput`、`BudgetStatusDTO` 等类型或方法。

- [ ] **Step 3: 增加 DTO 和输入类型**

在 `apps/api/internal/app/service.go` 的 DTO 区域新增：

```go
type BudgetStatusDTO struct {
    Category       CategoryDTO `json:"category"`
    BudgetCents    int64       `json:"budgetCents"`
    SpentCents     int64       `json:"spentCents"`
    RemainingCents int64       `json:"remainingCents"`
    UsagePercent   int         `json:"usagePercent"`
    Status         string      `json:"status"`
}

type BudgetSummaryDTO struct {
    Month                string            `json:"month"`
    TotalBudgetCents     int64             `json:"totalBudgetCents"`
    TotalSpentCents      int64             `json:"totalSpentCents"`
    TotalRemainingCents  int64             `json:"totalRemainingCents"`
    OverCount            int               `json:"overCount"`
    NearCount            int               `json:"nearCount"`
    Items                []BudgetStatusDTO `json:"items"`
}

type SetBudgetInput struct {
    AmountCents int64 `json:"amountCents"`
}

type CopyPreviousBudgetsResult struct {
    CopiedCount int64 `json:"copiedCount"`
}
```

- [ ] **Step 4: 实现 service 方法**

在 `apps/api/internal/app/service.go` 设置管理方法附近新增：

```go
func (s *Service) ListBudgets(ctx context.Context, month string) (BudgetSummaryDTO, error) {
    monthRange, err := parseMonthRange(month)
    if err != nil {
        return BudgetSummaryDTO{}, err
    }
    categories, err := s.queries.ListActiveCategories(ctx)
    if err != nil {
        return BudgetSummaryDTO{}, err
    }
    budgets, err := s.queries.ListBudgetsByMonth(ctx, monthRange.month)
    if err != nil {
        return BudgetSummaryDTO{}, err
    }
    spending, err := s.queries.ListMonthlyBudgetSpending(ctx, queries.ListMonthlyBudgetSpendingParams{
        StartDate: dateValue(monthRange.start),
        EndDate:   dateValue(monthRange.end),
    })
    if err != nil {
        return BudgetSummaryDTO{}, err
    }

    budgetByCategory := make(map[string]int64, len(budgets))
    for _, budget := range budgets {
        budgetByCategory[budget.CategoryID.String()] = budget.AmountCents
    }
    spentByCategory := make(map[string]int64, len(spending))
    for _, row := range spending {
        spentByCategory[row.CategoryID.String()] = row.SpentCents
    }

    summary := BudgetSummaryDTO{Month: monthRange.month}
    for _, category := range categories {
        if category.Type != "expense" {
            continue
        }
        item := budgetStatusDTO(categoryDTO(category), budgetByCategory[category.ID.String()], spentByCategory[category.ID.String()])
        summary.Items = append(summary.Items, item)
        summary.TotalBudgetCents += item.BudgetCents
        summary.TotalSpentCents += item.SpentCents
        summary.TotalRemainingCents += item.RemainingCents
        if item.Status == "over" {
            summary.OverCount++
        }
        if item.Status == "near" {
            summary.NearCount++
        }
    }
    return summary, nil
}

func (s *Service) SetBudget(ctx context.Context, actor MemberDTO, month string, categoryID string, input SetBudgetInput) (BudgetStatusDTO, error) {
    if err := requireAdmin(actor); err != nil {
        return BudgetStatusDTO{}, err
    }
    monthRange, err := parseMonthRange(month)
    if err != nil {
        return BudgetStatusDTO{}, err
    }
    if input.AmountCents <= 0 {
        return BudgetStatusDTO{}, errors.New("budget amount must be greater than zero")
    }
    categoryUUID, err := uuidFromString(categoryID)
    if err != nil {
        return BudgetStatusDTO{}, errors.New("invalid category ID")
    }
    category, err := s.queries.GetCategoryByID(ctx, categoryUUID)
    if err != nil {
        return BudgetStatusDTO{}, err
    }
    if !category.Active || category.Type != "expense" {
        return BudgetStatusDTO{}, errors.New("budget category must be an active expense category")
    }
    budget, err := s.queries.UpsertBudget(ctx, queries.UpsertBudgetParams{
        Month:       monthRange.month,
        CategoryID:  categoryUUID,
        AmountCents: input.AmountCents,
    })
    if err != nil {
        return BudgetStatusDTO{}, err
    }
    summary, err := s.ListBudgets(ctx, budget.Month)
    if err != nil {
        return BudgetStatusDTO{}, err
    }
    for _, item := range summary.Items {
        if item.Category.ID == categoryID {
            return item, nil
        }
    }
    return budgetStatusDTO(categoryDTO(category), budget.AmountCents, 0), nil
}

func (s *Service) CopyPreviousBudgets(ctx context.Context, actor MemberDTO, month string) (CopyPreviousBudgetsResult, error) {
    if err := requireAdmin(actor); err != nil {
        return CopyPreviousBudgetsResult{}, err
    }
    monthRange, err := parseMonthRange(month)
    if err != nil {
        return CopyPreviousBudgetsResult{}, err
    }
    copiedCount, err := s.queries.CopyPreviousBudgets(ctx, queries.CopyPreviousBudgetsParams{
        TargetMonth: monthRange.month,
        SourceMonth: monthRange.start.AddDate(0, -1, 0).Format("2006-01"),
    })
    if err != nil {
        return CopyPreviousBudgetsResult{}, err
    }
    return CopyPreviousBudgetsResult{CopiedCount: copiedCount}, nil
}
```

新增 helper：

```go
func budgetStatusDTO(category CategoryDTO, budgetCents int64, spentCents int64) BudgetStatusDTO {
    item := BudgetStatusDTO{
        Category:    category,
        BudgetCents: budgetCents,
        SpentCents:  spentCents,
        Status:      "unset",
    }
    if budgetCents <= 0 {
        return item
    }
    item.RemainingCents = budgetCents - spentCents
    item.UsagePercent = int((spentCents * 100) / budgetCents)
    switch {
    case spentCents >= budgetCents:
        item.Status = "over"
    case spentCents*100 >= budgetCents*80:
        item.Status = "near"
    default:
        item.Status = "normal"
    }
    return item
}
```

- [ ] **Step 5: 格式化并验证**

运行：

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
gofmt -w apps\api\internal\app\service.go apps\api\internal\app\service_test.go
Set-Location apps\api
go test ./internal/app -run "TestBudget" -count=1
go test ./... -count=1
```

预期：全部通过。

- [ ] **Step 6: 提交**

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
git add apps/api/internal/app/service.go apps/api/internal/app/service_test.go
git commit -m "feat: add budget service"
```

## Task 3：HTTP API 和测试

**Files:**

- Modify: `apps/api/internal/http/api.go`
- Modify: `apps/api/internal/http/api_test.go`

- [ ] **Step 1: 写失败测试**

在 `apps/api/internal/http/api_test.go` 新增：

```go
func TestBudgetAPIRequiresSession(t *testing.T) {
    handler := newInitializedAPI(t)
    req := httptest.NewRequest("GET", "/budgets?month=2026-05", nil)
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("status = %d, want 401", rec.Code)
    }
}
```

```go
func TestBudgetAPIAdminSetsBudget(t *testing.T) {
    handler, cookie := newInitializedAPIWithCookie(t)
    categoryID := firstCategoryID(t, handler, cookie, "expense")
    req := httptest.NewRequest("PUT", "/budgets/2026-05/"+categoryID, bytes.NewReader([]byte(`{"amountCents":120000}`)))
    req.AddCookie(cookie)
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
    }
}
```

```go
func TestBudgetAPIMemberCannotSetBudget(t *testing.T) {
    handler, adminCookie := newInitializedAPIWithCookie(t)
    member := createMemberViaAPI(t, handler, adminCookie, "Li", "member", "234567")
    memberCookie := loginViaAPI(t, handler, member.ID, "234567")
    categoryID := firstCategoryID(t, handler, adminCookie, "expense")
    req := httptest.NewRequest("PUT", "/budgets/2026-05/"+categoryID, bytes.NewReader([]byte(`{"amountCents":120000}`)))
    req.AddCookie(memberCookie)
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusForbidden {
        t.Fatalf("status = %d, want 403, body = %s", rec.Code, rec.Body.String())
    }
}
```

```go
func TestBudgetAPICopiesPreviousBudgets(t *testing.T) {
    handler, cookie := newInitializedAPIWithCookie(t)
    categoryID := firstCategoryID(t, handler, cookie, "expense")
    setReq := httptest.NewRequest("PUT", "/budgets/2026-04/"+categoryID, bytes.NewReader([]byte(`{"amountCents":120000}`)))
    setReq.AddCookie(cookie)
    setRec := httptest.NewRecorder()
    handler.ServeHTTP(setRec, setReq)
    if setRec.Code != http.StatusOK {
        t.Fatalf("set status = %d, want 200, body = %s", setRec.Code, setRec.Body.String())
    }

    copyReq := httptest.NewRequest("POST", "/budgets/2026-05/copy-previous", nil)
    copyReq.AddCookie(cookie)
    copyRec := httptest.NewRecorder()
    handler.ServeHTTP(copyRec, copyReq)
    if copyRec.Code != http.StatusOK {
        t.Fatalf("copy status = %d, want 200, body = %s", copyRec.Code, copyRec.Body.String())
    }
    var response struct {
        CopiedCount int64 `json:"copiedCount"`
    }
    if err := json.NewDecoder(copyRec.Body).Decode(&response); err != nil {
        t.Fatalf("Decode returned error: %v", err)
    }
    if response.CopiedCount != 1 {
        t.Fatalf("copied count = %d, want 1", response.CopiedCount)
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
Set-Location apps\api
go test ./internal/http -run "TestBudget" -count=1
```

预期：路由未注册，返回 404。

- [ ] **Step 3: 增加路由和 handlers**

在 `NewAPI` 增加：

```go
r.Get("/budgets", api.budgets)
r.Put("/budgets/{month}/{categoryId}", api.setBudget)
r.Post("/budgets/{month}/copy-previous", api.copyPreviousBudgets)
```

新增 handler：

```go
func (api *API) budgets(w stdhttp.ResponseWriter, r *stdhttp.Request) {
    if _, ok := api.requireSession(w, r); !ok {
        return
    }
    summary, err := api.service.ListBudgets(r.Context(), r.URL.Query().Get("month"))
    if err != nil {
        writeServiceError(w, err)
        return
    }
    writeJSON(w, stdhttp.StatusOK, summary)
}

func (api *API) setBudget(w stdhttp.ResponseWriter, r *stdhttp.Request) {
    member, ok := api.requireSession(w, r)
    if !ok {
        return
    }
    var input app.SetBudgetInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
        return
    }
    item, err := api.service.SetBudget(r.Context(), member, chi.URLParam(r, "month"), chi.URLParam(r, "categoryId"), input)
    if err != nil {
        writeServiceError(w, err)
        return
    }
    writeJSON(w, stdhttp.StatusOK, item)
}

func (api *API) copyPreviousBudgets(w stdhttp.ResponseWriter, r *stdhttp.Request) {
    member, ok := api.requireSession(w, r)
    if !ok {
        return
    }
    result, err := api.service.CopyPreviousBudgets(r.Context(), member, chi.URLParam(r, "month"))
    if err != nil {
        writeServiceError(w, err)
        return
    }
    writeJSON(w, stdhttp.StatusOK, result)
}
```

- [ ] **Step 4: 格式化并验证**

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
gofmt -w apps\api\internal\http\api.go apps\api\internal\http\api_test.go
Set-Location apps\api
go test ./internal/http -run "TestBudget" -count=1
go test ./... -count=1
```

预期：全部通过。

- [ ] **Step 5: 提交**

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
git add apps/api/internal/http/api.go apps/api/internal/http/api_test.go
git commit -m "feat: expose budget api"
```

## Task 4：前端 API 和预算页面

**Files:**

- Modify: `apps/web/src/api.ts`
- Modify: `apps/web/src/App.tsx`

- [ ] **Step 1: 增加前端预算类型和 API**

在 `apps/web/src/api.ts` 增加：

```ts
export type BudgetStatus = "unset" | "normal" | "near" | "over";

export type BudgetItem = {
  category: Category;
  budgetCents: number;
  spentCents: number;
  remainingCents: number;
  usagePercent: number;
  status: BudgetStatus;
};

export type BudgetSummary = {
  month: string;
  totalBudgetCents: number;
  totalSpentCents: number;
  totalRemainingCents: number;
  overCount: number;
  nearCount: number;
  items: BudgetItem[];
};

export async function getBudgets(month: string): Promise<BudgetSummary> {
  return request<BudgetSummary>(`/api/budgets?month=${encodeURIComponent(month)}`);
}

export async function setBudget(month: string, categoryId: string, amountCents: number): Promise<BudgetItem> {
  return request<BudgetItem>(`/api/budgets/${encodeURIComponent(month)}/${encodeURIComponent(categoryId)}`, {
    method: "PUT",
    body: JSON.stringify({ amountCents })
  });
}

export async function copyPreviousBudgets(month: string): Promise<{ copiedCount: number }> {
  return request<{ copiedCount: number }>(`/api/budgets/${encodeURIComponent(month)}/copy-previous`, {
    method: "POST",
    body: JSON.stringify({})
  });
}
```

- [ ] **Step 2: 替换 Budgets tab 占位页**

在 `apps/web/src/App.tsx` import 增加：

```ts
  copyPreviousBudgets,
  getBudgets,
  setBudget,
  type BudgetItem,
  type BudgetSummary,
```

在 `AuthenticatedShell` 中增加 state：

```ts
const [budgetSummary, setBudgetSummary] = useState<BudgetSummary | null>(null);
const [budgetError, setBudgetError] = useState("");
const [loadingBudgets, setLoadingBudgets] = useState(true);
```

增加刷新函数：

```ts
async function refreshBudgets(targetMonth = month) {
  setBudgetError("");
  setLoadingBudgets(true);
  try {
    const result = await getBudgets(targetMonth);
    setBudgetSummary(result);
  } catch (error) {
    setBudgetError(error instanceof Error ? error.message : "加载预算失败");
  } finally {
    setLoadingBudgets(false);
  }
}
```

在 `useEffect` 中和账本数据一起加载预算：

```ts
Promise.all([listCategories(), listTransactions(month), getMonthlyOverview(month), getBudgets(month)])
  .then(([categoryResult, transactionResult, overviewResult, budgetResult]) => {
    if (cancelled) {
      return;
    }
    setCategories(categoryResult.categories);
    setTransactions(transactionResult.transactions);
    setOverview(overviewResult);
    setBudgetSummary(budgetResult);
  })
```

把 Budgets tab 替换为：

```tsx
<Tabs.Content value="budgets">
  <BudgetsPanel
    member={member}
    month={month}
    onMonthChange={setMonth}
    summary={budgetSummary}
    loading={loadingBudgets}
    error={budgetError}
    onChanged={() => refreshBudgets(month)}
  />
</Tabs.Content>
```

- [ ] **Step 3: 增加 BudgetsPanel 组件**

在 `TransactionsPanel` 后新增：

```tsx
function BudgetsPanel({
  member,
  month,
  onMonthChange,
  summary,
  loading,
  error,
  onChanged
}: {
  member: Member;
  month: string;
  onMonthChange: (month: string) => void;
  summary: BudgetSummary | null;
  loading: boolean;
  error: string;
  onChanged: () => Promise<void>;
}) {
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const [submitError, setSubmitError] = useState("");

  useEffect(() => {
    if (!summary) {
      return;
    }
    setAmounts((current) => {
      const next = { ...current };
      for (const item of summary.items) {
        if (next[item.category.id] === undefined) {
          next[item.category.id] = item.budgetCents > 0 ? centsToYuanInput(item.budgetCents) : "";
        }
      }
      return next;
    });
  }, [summary]);

  async function handleSave(item: BudgetItem) {
    setMessage("");
    setSubmitError("");
    const amountCents = yuanToCents(amounts[item.category.id] ?? "");
    if (amountCents <= 0) {
      setSubmitError("请输入有效预算金额");
      return;
    }
    try {
      await setBudget(month, item.category.id, amountCents);
      await onChanged();
      setMessage("预算已保存");
    } catch (saveError) {
      setSubmitError(saveError instanceof Error ? saveError.message : "保存预算失败");
    }
  }

  async function handleCopyPrevious() {
    setMessage("");
    setSubmitError("");
    try {
      const result = await copyPreviousBudgets(month);
      await onChanged();
      setMessage(`已复制 ${result.copiedCount} 个预算`);
    } catch (copyError) {
      setSubmitError(copyError instanceof Error ? copyError.message : "复制预算失败");
    }
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-sm font-medium text-slate-600">预算</p>
            <h2 className="mt-1 text-2xl font-semibold">月度分类预算</h2>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <input
              type="month"
              value={month}
              onChange={(event) => onMonthChange(event.target.value)}
              className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700"
            />
            {member.role === "admin" ? (
              <button
                type="button"
                onClick={handleCopyPrevious}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700"
              >
                复制上月预算
              </button>
            ) : null}
          </div>
        </div>

        {loading ? <p className="mt-5 text-sm text-slate-500">正在加载...</p> : null}
        {error ? <p className="mt-5 text-sm text-red-600">{error}</p> : null}
        {submitError ? <p className="mt-5 text-sm text-red-600">{submitError}</p> : null}
        {message ? <p className="mt-5 text-sm text-emerald-700">{message}</p> : null}

        <div className="mt-5 space-y-3">
          {summary?.items.map((item) => (
            <div key={item.category.id} className="rounded-md border border-slate-200 p-3">
              <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div className="min-w-0">
                  <p className="text-sm font-medium">{item.category.name}</p>
                  <p className="mt-1 text-xs text-slate-500">
                    已花 {formatMoney(item.spentCents)} · {budgetStatusLabel(item.status)}
                  </p>
                </div>
                <div className="grid gap-2 sm:grid-cols-[10rem_auto]">
                  <input
                    value={amounts[item.category.id] ?? ""}
                    onChange={(event) => setAmounts((current) => ({ ...current, [item.category.id]: event.target.value }))}
                    disabled={member.role !== "admin"}
                    inputMode="decimal"
                    placeholder="预算金额"
                    className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700 disabled:bg-slate-50"
                  />
                  {member.role === "admin" ? (
                    <button
                      type="button"
                      onClick={() => handleSave(item)}
                      className="rounded-md bg-emerald-700 px-3 py-2 text-sm font-medium text-white"
                    >
                      保存
                    </button>
                  ) : null}
                </div>
              </div>
              <div className="mt-3 h-2 rounded-full bg-slate-100">
                <div
                  className={`h-2 rounded-full ${budgetStatusBarClass(item.status)}`}
                  style={{ width: `${Math.min(item.usagePercent, 100)}%` }}
                />
              </div>
              <div className="mt-2 grid grid-cols-3 gap-2 text-xs text-slate-500">
                <span>预算 {item.budgetCents > 0 ? formatMoney(item.budgetCents) : "未设置"}</span>
                <span>剩余 {item.budgetCents > 0 ? formatMoney(item.remainingCents) : "-"}</span>
                <span>{item.budgetCents > 0 ? `${item.usagePercent}%` : "0%"}</span>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
        <p className="text-sm font-medium text-slate-600">本月汇总</p>
        <div className="mt-4 grid gap-2">
          <Metric label="总预算" value={formatMoney(summary?.totalBudgetCents ?? 0)} />
          <Metric label="总支出" value={formatMoney(summary?.totalSpentCents ?? 0)} />
          <Metric label="剩余" value={formatMoney(summary?.totalRemainingCents ?? 0)} />
          <Metric label="接近预算" value={`${summary?.nearCount ?? 0} 个分类`} />
          <Metric label="已超支" value={`${summary?.overCount ?? 0} 个分类`} />
        </div>
      </section>
    </div>
  );
}
```

新增 helper：

```tsx
function centsToYuanInput(cents: number): string {
  return (cents / 100).toFixed(2);
}

function budgetStatusLabel(status: string): string {
  if (status === "normal") {
    return "正常";
  }
  if (status === "near") {
    return "接近预算";
  }
  if (status === "over") {
    return "已超支";
  }
  return "未设置";
}

function budgetStatusBarClass(status: string): string {
  if (status === "over") {
    return "bg-red-600";
  }
  if (status === "near") {
    return "bg-amber-500";
  }
  if (status === "normal") {
    return "bg-emerald-700";
  }
  return "bg-slate-200";
}
```

- [ ] **Step 4: 构建验证**

运行：

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
pnpm --dir apps/web build
task test
```

预期：两个命令退出码为 0。

- [ ] **Step 5: 提交**

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
git add apps/web/src/api.ts apps/web/src/App.tsx
git commit -m "feat: add budget management UI"
```

## Task 5：最终验证和推送

**Files:**

- No source changes expected.

- [ ] **Step 1: 全量测试**

运行：

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
task test
.\scripts\build.ps1
```

预期：

- Go 测试全部通过。
- Vite build 通过。
- 输出 `Build complete: dist\commune-server.exe`。

- [ ] **Step 2: 本地手动检查**

运行：

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
.\scripts\start-dev.ps1
```

检查：

- Admin 可以打开预算页。
- Admin 可以给支出分类设置预算。
- 预算页显示已花、剩余、百分比和状态。
- 花费达到 80% 显示接近预算。
- 花费达到 100% 显示已超支。
- Admin 可以复制上月预算。
- 普通成员可以查看预算，但看不到保存和复制按钮。

- [ ] **Step 3: 推送分支**

如果在 `feature/budget-mvp` 分支上执行：

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8; [Console]::OutputEncoding = [Text.Encoding]::UTF8; chcp 65001 > $null;
git status --short --branch
git push -u origin feature/budget-mvp
```

预期：工作区干净，远程分支推送成功。

## 自检

- 计划覆盖设计文档中的按月分类预算、复制上月、固定状态、权限、预算页 UI。
- 预算只支持支出分类。
- 未设置预算的分类仍然显示。
- 复制上月预算不覆盖当前月已有预算。
- 不实现删除预算、总预算、收入预算、阈值配置或通知。
