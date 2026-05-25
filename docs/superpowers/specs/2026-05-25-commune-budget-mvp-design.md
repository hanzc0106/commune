# Commune 预算 MVP 设计

## 背景

Commune 当前已经具备家庭初始化、成员登录、成员管理、分类管理、记账、流水和月度概览能力。预算是 MVP 的下一块核心能力，用来帮助家庭按月观察支出是否接近或超过预期。

本设计只覆盖预算 MVP，不包含年度预算、收入预算、账户预算、预算模板自动继承、提醒通知或复杂报表。

## 目标

- 支持按月给支出分类设置预算。
- 成员可以查看预算执行情况。
- 管理员可以设置、更新预算，并从上个月复制预算到当前月份。
- 预算状态固定为正常、接近预算、已超支和未设置。
- 预算页在移动端单列展示，在 PC 端使用更宽的主列表和汇总布局。

## 非目标

- 不做总预算和分类预算并存。
- 不给收入分类设置预算。
- 不做预算删除。需要清空预算时，可以在后续版本增加停用或删除能力。
- 不做自动沿用上月预算。
- 不做预算阈值配置。
- 不做通知推送。
- 不做多人审批或预算调整历史。

## 产品规则

### 预算维度

预算维度是 `month + category_id`。

- `month` 使用 `YYYY-MM` 格式。
- `category_id` 必须指向支出分类。
- 每个分类每个月最多一条预算。
- 预算金额用整数分存储，避免浮点误差。
- 预算金额必须大于 0。

### 权限

- 所有已登录成员都可以查看预算。
- 只有管理员可以设置或更新预算。
- 只有管理员可以复制上月预算。
- 权限必须由服务端校验，前端隐藏按钮不是可信边界。

### 预算状态

预算状态由本月该分类支出金额和预算金额计算。

- `unset`：该分类本月没有预算。
- `normal`：已花金额小于预算的 80%。
- `near`：已花金额大于等于预算的 80%，且小于预算。
- `over`：已花金额大于等于预算。

百分比按 `spent_cents / budget_cents * 100` 计算。未设置预算时百分比为 0。

### 复制上月预算

管理员在预算页可以点击“复制上月预算”。

规则：

- 来源月份是当前选择月份的上一个月。
- 只复制仍然 active 的支出分类预算。
- 如果当前月某分类已经有预算，不覆盖。
- 如果上个月没有可复制预算，返回成功但复制数量为 0，前端显示明确反馈。
- 复制后的预算金额与上月一致。

这个规则避免自动魔法，也避免误覆盖管理员已经录入的当前月预算。

## 数据模型

新增表：`budgets`

字段：

- `id UUID PRIMARY KEY`
- `month TEXT NOT NULL`
- `category_id UUID NOT NULL REFERENCES categories(id)`
- `amount_cents BIGINT NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

约束：

- `UNIQUE(month, category_id)`
- `amount_cents > 0`
- `month` 需要符合 `YYYY-MM` 格式

索引：

- `(month)`
- `(category_id)`

分类是否为支出分类由 service 层校验。PostgreSQL check 约束不能直接跨表校验分类类型，因此不在数据库层强制 `category.type = 'expense'`。

## 后端设计

### sqlc 查询

新增 `apps/api/queries/budgets.sql`。

需要支持：

- 按月列出预算。
- upsert 某月某分类预算。
- 查询上月预算。
- 批量插入复制预算，跳过当前月已有预算。
- 计算某月每个支出分类的已花金额。

预算展示建议由 service 组合查询完成：先取 active 支出分类，再取预算，再取支出汇总，最后合成 DTO。这样逻辑清晰，也能保证未设置预算的分类仍然返回。

### Service

新增输入和 DTO：

- `BudgetStatusDTO`
- `BudgetSummaryDTO`
- `SetBudgetInput`
- `CopyPreviousBudgetsResult`

建议返回结构：

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
    Month              string            `json:"month"`
    TotalBudgetCents   int64             `json:"totalBudgetCents"`
    TotalSpentCents    int64             `json:"totalSpentCents"`
    TotalRemainingCents int64            `json:"totalRemainingCents"`
    OverCount          int               `json:"overCount"`
    NearCount          int               `json:"nearCount"`
    Items              []BudgetStatusDTO `json:"items"`
}
```

`remainingCents` 可以为负数，表示已超支。

Service 方法：

- `ListBudgets(ctx, month string) (BudgetSummaryDTO, error)`
- `SetBudget(ctx, actor MemberDTO, month string, categoryID string, input SetBudgetInput) (BudgetStatusDTO, error)`
- `CopyPreviousBudgets(ctx, actor MemberDTO, month string) (CopyPreviousBudgetsResult, error)`

校验：

- `month` 必须是 `YYYY-MM`。
- `amount_cents` 必须大于 0。
- 设置预算的分类必须存在、active，并且 type 是 `expense`。
- Admin-only 方法必须校验 `actor.Role == "admin"`。

### HTTP API

新增接口：

- `GET /api/budgets?month=YYYY-MM`
- `PUT /api/budgets/{month}/{categoryId}`
- `POST /api/budgets/{month}/copy-previous`

返回规则：

- 未登录返回 `401`。
- 权限不足返回 `403`。
- 输入错误返回 `400`。
- 服务端错误返回 `500`。

### 月度概览关系

预算 MVP 不修改现有 `GET /api/overview/monthly` 的返回结构。预算页独立调用 `GET /api/budgets`。

后续如果要在首页显示预算摘要，可以在 Budget API 稳定后再把总预算、接近预算数量、超支数量接入首页概览。

## 前端设计

当前 `budgets` tab 是占位页。预算 MVP 会替换为真实预算页。

### 页面结构

移动端：

- 顶部月份选择。
- Admin 可见“复制上月预算”按钮。
- 预算分类列表单列展示。
- 每个分类显示名称、预算、已花、剩余、进度条和状态。
- Admin 可直接输入预算金额并保存。
- Member 只读。

PC 端：

- 顶部月份选择和操作按钮。
- 左侧为预算分类列表。
- 右侧为本月预算汇总：
  - 总预算。
  - 总支出。
  - 剩余金额。
  - 接近预算分类数量。
  - 超支分类数量。

### 交互规则

- 切换月份后重新加载预算。
- 保存单个分类预算后刷新预算数据。
- 复制上月预算后刷新预算数据，并显示复制数量。
- 未设置预算的分类展示为“未设置”，进度条为空。
- 接近预算和超支状态要视觉优先，但仍保持当前产品的克制样式。

## 测试策略

后端测试：

- 预算只能设置到支出分类。
- 普通成员不能设置预算。
- 同月同分类重复设置会更新金额。
- 未设置预算的支出分类仍在预算列表中返回。
- 状态计算覆盖 `unset`、`normal`、`near`、`over`。
- 复制上月预算不覆盖当前月已有预算。
- 复制上月预算只复制 active 支出分类。

HTTP 测试：

- 未登录访问预算 API 返回 `401`。
- member 编辑预算返回 `403`。
- admin 设置预算返回成功。
- admin 复制上月预算返回复制数量。

前端验证：

- `pnpm --dir apps/web build`
- `task test`
- 本地手动验证 Admin 设置预算、复制上月预算、普通成员只读、状态显示正确。

## 实施顺序

1. 新增 `budgets` 迁移、sqlc 查询和生成代码。
2. 新增 service DTO、预算计算和权限测试。
3. 新增 HTTP API 和测试。
4. 新增前端 API 封装。
5. 替换 Budgets tab 占位页。
6. 跑全量测试和构建。

## 风险和取舍

- `app.Service` 文件会继续变大。预算 MVP 先延续现有结构，避免在功能开发中同时做大规模拆包。后续可以单独规划 service 分包。
- 预算只针对 active 支出分类。历史停用分类不会出现在新月份预算设置中。
- 当前月已有预算时复制上月预算不覆盖，减少误操作；如果需要覆盖，后续可以增加明确的“覆盖复制”能力。
