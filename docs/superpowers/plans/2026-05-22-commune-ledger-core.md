# Commune 账本核心实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 实现无账户版本的账本核心能力：默认分类、交易流水 CRUD、月度概览，以及 Add/Transactions 页面的真实数据接入。

**架构：** 延续当前 monorepo 和 `app.Service` 结构，先不引入额外领域包拆分。数据库使用 PostgreSQL migration + sqlc 查询，服务层负责输入校验和权限判断，HTTP 层只做请求解析和响应编码，前端通过 `apps/web/src/api.ts` 统一访问 API。

**技术栈：** Go、Chi、pgx、sqlc、PostgreSQL、React、Vite、Tailwind CSS、Radix Tabs、PowerShell、Taskfile。

---

## 范围锁定

本计划实现：

- `categories` 表和默认分类种子数据。
- `transactions` 表。
- 分类列表 API。
- 交易创建、列表、更新、删除 API。
- 月度概览 API。
- Add 页面快速记账表单。
- Transactions 页面当前月流水列表。
- Budgets 页面保留空状态，不实现预算编辑。

本计划不实现：

- 账户管理。
- 转账。
- 预算表和预算编辑。
- 成员管理 UI。
- 导入导出。
- 附件。

## 文件结构

创建：

- `apps/api/migrations/000003_ledger_core.sql`：分类和交易 schema。
- `apps/api/queries/categories.sql`：分类 sqlc 查询。
- `apps/api/queries/transactions.sql`：交易 sqlc 查询。
- `apps/api/queries/overview.sql`：月度概览 sqlc 查询。

修改：

- `apps/api/internal/app/service.go`：增加分类、交易、月度概览 DTO、输入结构和 service 方法。
- `apps/api/internal/app/service_test.go`：增加默认分类、交易权限和月度汇总测试。
- `apps/api/internal/http/api.go`：增加 ledger API 路由和 handler。
- `apps/api/internal/http/handler_test.go` 或新增 API 测试：覆盖基本 HTTP 行为。
- `apps/web/src/api.ts`：增加分类、交易、概览 API 类型和函数。
- `apps/web/src/App.tsx`：替换占位面板为 Add 表单、Transactions 列表和 Overview。
- `scripts/reset-dev-db.ps1` 与 `Taskfile.yml`：重置开发数据时包含新表。

生成：

- `apps/api/internal/db/queries/*.sql.go`：由 `sqlc generate` 生成，不能手写。

---

## Task 1：数据库 schema 和 sqlc 查询

**文件：**

- 创建：`apps/api/migrations/000003_ledger_core.sql`
- 创建：`apps/api/queries/categories.sql`
- 创建：`apps/api/queries/transactions.sql`
- 创建：`apps/api/queries/overview.sql`
- 修改：`scripts/reset-dev-db.ps1`
- 修改：`Taskfile.yml`
- 生成：`apps/api/internal/db/queries/*.sql.go`

- [ ] **Step 1：新增 migration**

创建 `apps/api/migrations/000003_ledger_core.sql`：

```sql
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    icon_key TEXT NOT NULL DEFAULT 'circle',
    color_key TEXT NOT NULL DEFAULT 'slate',
    sort_order INTEGER NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    system_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT categories_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT categories_type_valid CHECK (type IN ('expense', 'income'))
);

CREATE INDEX IF NOT EXISTS categories_active_sort_idx ON categories (active, type, sort_order, name);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id),
    member_id UUID NOT NULL REFERENCES members(id),
    transaction_date DATE NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transactions_type_valid CHECK (type IN ('expense', 'income')),
    CONSTRAINT transactions_amount_positive CHECK (amount_cents > 0)
);

CREATE INDEX IF NOT EXISTS transactions_month_idx ON transactions (transaction_date DESC);
CREATE INDEX IF NOT EXISTS transactions_category_id_idx ON transactions (category_id);
CREATE INDEX IF NOT EXISTS transactions_member_id_idx ON transactions (member_id);
```

- [ ] **Step 2：新增 sqlc 查询**

`apps/api/queries/categories.sql`：

```sql
-- name: CreateCategory :one
INSERT INTO categories (name, type, icon_key, color_key, sort_order, active, system_default)
VALUES ($1, $2, $3, $4, $5, TRUE, $6)
RETURNING *;

-- name: ListActiveCategories :many
SELECT *
FROM categories
WHERE active = TRUE
ORDER BY type, sort_order, name;
```

`apps/api/queries/transactions.sql`：

```sql
-- name: CreateTransaction :one
INSERT INTO transactions (type, amount_cents, category_id, member_id, transaction_date, note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTransactionByID :one
SELECT *
FROM transactions
WHERE id = $1;

-- name: ListTransactionsByMonth :many
SELECT
    t.id,
    t.type,
    t.amount_cents,
    t.transaction_date,
    t.note,
    t.created_at,
    t.updated_at,
    c.id AS category_id,
    c.name AS category_name,
    c.type AS category_type,
    c.icon_key AS category_icon_key,
    c.color_key AS category_color_key,
    m.id AS member_id,
    m.name AS member_name,
    m.role AS member_role
FROM transactions t
JOIN categories c ON c.id = t.category_id
JOIN members m ON m.id = t.member_id
WHERE t.transaction_date >= $1 AND t.transaction_date < $2
ORDER BY t.transaction_date DESC, t.created_at DESC;

-- name: UpdateTransaction :one
UPDATE transactions
SET
    type = $2,
    amount_cents = $3,
    category_id = $4,
    transaction_date = $5,
    note = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM transactions
WHERE id = $1;
```

`apps/api/queries/overview.sql`：

```sql
-- name: GetMonthlyTotals :one
SELECT
    COALESCE(SUM(amount_cents) FILTER (WHERE type = 'income'), 0)::bigint AS income_cents,
    COALESCE(SUM(amount_cents) FILTER (WHERE type = 'expense'), 0)::bigint AS expense_cents
FROM transactions
WHERE transaction_date >= $1 AND transaction_date < $2;

-- name: ListMonthlyExpenseCategoryTotals :many
SELECT
    c.id AS category_id,
    c.name AS category_name,
    c.icon_key AS category_icon_key,
    c.color_key AS category_color_key,
    COALESCE(SUM(t.amount_cents), 0)::bigint AS expense_cents
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.type = 'expense' AND t.transaction_date >= $1 AND t.transaction_date < $2
GROUP BY c.id, c.name, c.icon_key, c.color_key
ORDER BY expense_cents DESC, c.name;
```

- [ ] **Step 3：生成 sqlc 代码**

运行：

```powershell
.\scripts\sqlc.ps1
```

预期：`apps/api/internal/db/queries` 下生成或更新 `categories.sql.go`、`transactions.sql.go`、`overview.sql.go`。

- [ ] **Step 4：更新重置脚本**

把 `scripts/reset-dev-db.ps1` 和 `Taskfile.yml` 中的 TRUNCATE 表列表改为：

```text
transactions, categories, sessions, members, app_settings
```

- [ ] **Step 5：验证迁移和生成代码**

运行：

```powershell
Set-Location apps\api
go test ./internal/db -count=1
go test ./internal/db/queries -count=1
```

预期：PASS。此任务只提交 schema、SQL 查询和生成代码，不提交依赖未实现 service 方法的测试。

- [ ] **Step 6：提交**

运行：

```powershell
git add apps/api/migrations apps/api/queries apps/api/internal/db/queries scripts/reset-dev-db.ps1 Taskfile.yml
git commit -m "feat: add ledger schema and queries"
```

---

## Task 2：分类 service 和默认分类种子

**文件：**

- 修改：`apps/api/internal/app/service.go`
- 修改：`apps/api/internal/app/service_test.go`

- [ ] **Step 1：先写失败测试，验证初始化会创建默认分类**

在 `apps/api/internal/app/service_test.go` 增加测试：

```go
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
	_, _ = pool.Exec(ctx, "TRUNCATE transactions, categories, sessions, members, app_settings RESTART IDENTITY CASCADE")

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
	if len(categories) < 10 {
		t.Fatalf("category count = %d, want at least 10", len(categories))
	}
	if categories[0].Name == "" || categories[0].Type == "" {
		t.Fatalf("first category is incomplete: %+v", categories[0])
	}
}
```

- [ ] **Step 2：运行测试确认失败**

运行：

```powershell
Set-Location apps\api
go test ./internal/app -run TestInitializeCreatesDefaultCategories -count=1
```

预期：编译失败，提示 `service.ListCategories undefined`。

- [ ] **Step 3：实现 DTO 和默认分类**

在 `service.go` 增加：

```go
type CategoryDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	IconKey       string `json:"iconKey"`
	ColorKey      string `json:"colorKey"`
	SortOrder     int32  `json:"sortOrder"`
	SystemDefault bool   `json:"systemDefault"`
}

var defaultCategories = []struct {
	name      string
	kind      string
	iconKey   string
	colorKey  string
	sortOrder int32
}{
	{"餐饮", "expense", "utensils", "emerald", 10},
	{"日用", "expense", "shopping-bag", "sky", 20},
	{"交通", "expense", "bus", "amber", 30},
	{"住房", "expense", "home", "slate", 40},
	{"医疗", "expense", "heart-pulse", "rose", 50},
	{"娱乐", "expense", "gamepad-2", "violet", 60},
	{"孩子", "expense", "baby", "pink", 70},
	{"其他支出", "expense", "circle", "zinc", 990},
	{"工资", "income", "wallet", "emerald", 10},
	{"其他收入", "income", "plus-circle", "teal", 990},
}
```

- [ ] **Step 4：实现 `ListCategories` 和初始化种子**

在 `Initialize` 的事务内，创建 admin member 后、创建 session 前，循环调用 `qtx.CreateCategory` 插入默认分类。

增加：

```go
func (s *Service) ListCategories(ctx context.Context) ([]CategoryDTO, error) {
	rows, err := s.queries.ListActiveCategories(ctx)
	if err != nil {
		return nil, err
	}
	categories := make([]CategoryDTO, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, categoryDTO(row))
	}
	return categories, nil
}
```

`categoryDTO` 使用 sqlc 生成的 `queries.Category` 转换。

- [ ] **Step 5：运行测试**

运行：

```powershell
Set-Location apps\api
go test ./internal/app -run TestInitializeCreatesDefaultCategories -count=1
```

预期：PASS。

- [ ] **Step 6：提交**

```powershell
git add apps/api/internal/app/service.go apps/api/internal/app/service_test.go
git commit -m "feat: seed default ledger categories"
```

---

## Task 3：交易 service、权限和月度概览

**文件：**

- 修改：`apps/api/internal/app/service.go`
- 修改：`apps/api/internal/app/service_test.go`

- [ ] **Step 1：写失败测试：成员只能删除自己的交易，Admin 可以删除任意交易**

在 `service_test.go` 增加测试，使用 `Initialize` 创建 admin，通过 sqlc 直接创建一个 member，再分别创建交易，断言：

- 普通成员删除 admin 的交易返回错误。
- Admin 删除普通成员交易成功。

运行：

```powershell
Set-Location apps\api
go test ./internal/app -run TestTransactionPermissions -count=1
```

预期：编译失败，提示交易方法未定义。

- [ ] **Step 2：增加输入和 DTO**

在 `service.go` 增加：

```go
type CreateTransactionInput struct {
	Type            string `json:"type"`
	AmountCents     int64  `json:"amountCents"`
	CategoryID      string `json:"categoryId"`
	TransactionDate string `json:"transactionDate"`
	Note            string `json:"note"`
}

type UpdateTransactionInput struct {
	Type            string `json:"type"`
	AmountCents     int64  `json:"amountCents"`
	CategoryID      string `json:"categoryId"`
	TransactionDate string `json:"transactionDate"`
	Note            string `json:"note"`
}

type TransactionDTO struct {
	ID              string      `json:"id"`
	Type            string      `json:"type"`
	AmountCents     int64       `json:"amountCents"`
	Category        CategoryDTO `json:"category"`
	Member          MemberDTO   `json:"member"`
	TransactionDate string      `json:"transactionDate"`
	Note            string      `json:"note"`
	CreatedAt       string      `json:"createdAt"`
	UpdatedAt       string      `json:"updatedAt"`
}

type MonthlyOverviewDTO struct {
	Month          string                     `json:"month"`
	IncomeCents    int64                      `json:"incomeCents"`
	ExpenseCents   int64                      `json:"expenseCents"`
	BalanceCents   int64                      `json:"balanceCents"`
	CategoryTotals []MonthlyCategoryTotalDTO  `json:"categoryTotals"`
	Recent         []TransactionDTO           `json:"recent"`
}
```

- [ ] **Step 3：实现 service 方法**

增加方法：

- `CreateTransaction(ctx, actor MemberDTO, input CreateTransactionInput) (TransactionDTO, error)`
- `ListTransactions(ctx, month string) ([]TransactionDTO, error)`
- `UpdateTransaction(ctx, actor MemberDTO, id string, input UpdateTransactionInput) (TransactionDTO, error)`
- `DeleteTransaction(ctx, actor MemberDTO, id string) error`
- `MonthlyOverview(ctx, month string) (MonthlyOverviewDTO, error)`

规则：

- `type` 只允许 `expense` 或 `income`。
- `amountCents` 必须大于 0。
- `transactionDate` 使用 `YYYY-MM-DD`。
- `month` 使用 `YYYY-MM`，空值默认为当前月。
- 更新和删除前调用 `GetTransactionByID`。
- `actor.Role == "admin"` 可以操作任意交易。
- 非 admin 只能操作 `member_id == actor.ID` 的交易。

- [ ] **Step 4：运行 app 测试**

```powershell
Set-Location apps\api
go test ./internal/app -count=1
```

预期：PASS。

- [ ] **Step 5：提交**

```powershell
git add apps/api/internal/app/service.go apps/api/internal/app/service_test.go
git commit -m "feat: add ledger transaction service"
```

---

## Task 4：HTTP API

**文件：**

- 修改：`apps/api/internal/http/api.go`
- 修改：`apps/api/internal/http/handler_test.go`

- [ ] **Step 1：写失败测试**

新增 HTTP 测试覆盖未登录访问：

```go
func TestLedgerAPIRequiresSession(t *testing.T) {
	handler := NewAPI(app.NewService(nil))
	req := httptest.NewRequest("GET", "/categories", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
```

如果 `app.NewService(nil)` 不适合当前结构，则改用一个可注入测试 service 的小接口；不要引入全局变量。

- [ ] **Step 2：增加路由**

在 `NewAPI` 增加：

```go
r.Get("/categories", api.categories)
r.Get("/transactions", api.transactions)
r.Post("/transactions", api.createTransaction)
r.Patch("/transactions/{id}", api.updateTransaction)
r.Delete("/transactions/{id}", api.deleteTransaction)
r.Get("/overview/monthly", api.monthlyOverview)
```

- [ ] **Step 3：增加当前会话解析 helper**

新增 helper：

```go
func (api *API) requireSession(w stdhttp.ResponseWriter, r *stdhttp.Request) (app.MemberDTO, bool) {
	session, err := api.service.SessionFromToken(r.Context(), sessionTokenFromRequest(r))
	if err != nil {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "login required"})
		return app.MemberDTO{}, false
	}
	return session.Member, true
}
```

- [ ] **Step 4：实现 handlers**

每个 handler 先调用 `requireSession`。列表和概览读取 `month` query。创建和更新 decode JSON body。删除读取 `chi.URLParam(r, "id")`。

- [ ] **Step 5：运行测试**

```powershell
Set-Location apps\api
go test ./internal/http -count=1
go test ./...
```

预期：PASS。

- [ ] **Step 6：提交**

```powershell
git add apps/api/internal/http/api.go apps/api/internal/http/handler_test.go
git commit -m "feat: expose ledger api"
```

---

## Task 5：前端 API 类型和 Add 页面

**文件：**

- 修改：`apps/web/src/api.ts`
- 修改：`apps/web/src/App.tsx`

- [ ] **Step 1：增加前端 API 类型**

在 `apps/web/src/api.ts` 增加：

```ts
export type Category = {
  id: string;
  name: string;
  type: "expense" | "income";
  iconKey: string;
  colorKey: string;
  sortOrder: number;
  systemDefault: boolean;
};

export type Transaction = {
  id: string;
  type: "expense" | "income";
  amountCents: number;
  category: Category;
  member: Member;
  transactionDate: string;
  note: string;
  createdAt: string;
  updatedAt: string;
};
```

增加函数：

- `listCategories()`
- `listTransactions(month: string)`
- `createTransaction(input)`
- `updateTransaction(id, input)`
- `deleteTransaction(id)`
- `getMonthlyOverview(month: string)`

- [ ] **Step 2：实现 Add 页面状态和提交**

在 `AuthenticatedShell` 内加载 categories 和 overview。替换 Add tab 占位内容为：

- 金额输入。
- 收入 / 支出切换。
- 分类按钮网格。
- 日期输入。
- 备注输入。
- 提交按钮。

金额输入前端先转换为分：

```ts
function yuanToCents(value: string): number {
  const normalized = value.trim();
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) {
    return 0;
  }
  const [yuan, cents = ""] = normalized.split(".");
  return Number(yuan) * 100 + Number(cents.padEnd(2, "0"));
}
```

- [ ] **Step 3：运行前端构建**

```powershell
pnpm --dir apps/web build
```

预期：TypeScript 和 Vite build 通过。

- [ ] **Step 4：提交**

```powershell
git add apps/web/src/api.ts apps/web/src/App.tsx
git commit -m "feat: add transaction entry UI"
```

---

## Task 6：Transactions 页面和月度概览

**文件：**

- 修改：`apps/web/src/App.tsx`

- [ ] **Step 1：实现月度概览组件**

展示：

- 本月收入。
- 本月支出。
- 本月结余。
- 支出分类排行前三。

金额格式：

```ts
function formatMoney(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}
```

- [ ] **Step 2：实现 Transactions 列表**

展示当前月交易，字段：

- 日期。
- 分类名。
- 成员名。
- 备注。
- 金额。

支出显示为负数样式，收入显示为正数样式。

- [ ] **Step 3：增加月份切换**

使用 `<input type="month">` 控制当前月份。切换月份后重新加载 overview 和 transactions。

- [ ] **Step 4：运行验证**

```powershell
pnpm --dir apps/web build
```

预期：PASS。

- [ ] **Step 5：提交**

```powershell
git add apps/web/src/App.tsx
git commit -m "feat: show ledger monthly activity"
```

---

## Task 7：最终验证和推送

**文件：**

- 可能修改：`README.md`，仅当启动或测试说明需要更新。

- [ ] **Step 1：运行完整测试**

```powershell
task test
```

预期：

- 后端 `go test ./...` 通过。
- 前端 TypeScript 和 Vite build 通过。

- [ ] **Step 2：运行完整构建**

```powershell
.\scripts\build.ps1
```

预期：生成 `dist\commune-server.exe`。

- [ ] **Step 3：手动启动检查**

```powershell
.\scripts\start-dev.ps1
```

检查：

- 初始化后默认分类出现。
- Add 页面可以创建一笔支出。
- Transactions 页面能看到刚创建的流水。
- 月度概览金额更新。

- [ ] **Step 4：推送分支**

```powershell
git status --short --branch
git push -u origin feature/ledger-core
```

预期：工作区干净，远端分支创建成功。

## 自检

- 设计范围已覆盖：分类、交易、月度概览、Add 页面、Transactions 页面。
- 不包含账户、预算编辑、转账、导入导出。
- 后端权限在 service 层校验，前端只负责展示和调用。
- 计划中的命令均使用当前项目已有入口：`task test`、`scripts/build.ps1`、`scripts/sqlc.ps1`。
