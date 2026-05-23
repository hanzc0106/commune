# Commune 设置与管理基础实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 实现 Settings 页面基础管理能力：Admin 管理成员和分类，成员修改自己的 PIN。

**架构：** 延续现有 `app.Service` 聚合业务逻辑，sqlc 负责数据访问，HTTP 层做 session/权限入口和 JSON 编解码，前端继续在 `App.tsx` 内按现有轻量结构迭代 Settings 页面。服务端权限是唯一可信边界。

**技术栈：** Go、Chi、pgx、sqlc、PostgreSQL、React、Vite、Tailwind CSS、PowerShell、Taskfile。

---

## 范围

实现：

- Admin 成员列表、新增成员、停用成员、重置 PIN。
- 当前成员修改自己的 PIN。
- Admin 分类新增、更新名称/图标/颜色/排序、停用分类。
- Settings 页面按角色展示管理区和个人设置区。

不实现：

- 删除成员。
- 删除分类。
- 预算。
- 分类拖拽排序。
- 邀请、邮箱、手机号、OAuth。

## Task 1：sqlc 查询

**文件：**

- 修改：`apps/api/queries/members.sql`
- 修改：`apps/api/queries/categories.sql`
- 生成：`apps/api/internal/db/queries/*.sql.go`

- [ ] 增加成员查询：

```sql
-- name: ListMembers :many
SELECT id, name, role, active, created_at, updated_at
FROM members
ORDER BY active DESC, lower(name);

-- name: DisableMember :one
UPDATE members
SET active = FALSE, updated_at = now()
WHERE id = $1
RETURNING id, name, pin_hash, role, active, created_at, updated_at;

-- name: UpdateMemberPIN :one
UPDATE members
SET pin_hash = $2, updated_at = now()
WHERE id = $1
RETURNING id, name, pin_hash, role, active, created_at, updated_at;

-- name: CountActiveAdmins :one
SELECT count(*)::bigint
FROM members
WHERE active = TRUE AND role = 'admin';
```

- [ ] 增加分类查询：

```sql
-- name: ListCategories :many
SELECT *
FROM categories
ORDER BY active DESC, type, sort_order, name;

-- name: UpdateCategory :one
UPDATE categories
SET
    name = $2,
    icon_key = $3,
    color_key = $4,
    sort_order = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DisableCategory :one
UPDATE categories
SET active = FALSE, updated_at = now()
WHERE id = $1
RETURNING *;
```

- [ ] 运行：

```powershell
.\scripts\sqlc.ps1
Set-Location apps\api
go test ./internal/db/queries -count=1
```

- [ ] 提交：

```powershell
git add apps/api/queries apps/api/internal/db/queries
git commit -m "feat: add settings sql queries"
```

## Task 2：Service 和权限测试

**文件：**

- 修改：`apps/api/internal/app/service.go`
- 修改：`apps/api/internal/app/service_test.go`

- [ ] 先写失败测试：

测试内容：

- Member 调用 Admin-only 方法返回错误。
- Admin 新增 member 成功。
- 停用 member 后不能登录。
- 不能停用最后一个 active admin。
- Admin 重置 member PIN 后，新 PIN 可登录。
- 当前成员修改自己的 PIN 后，新 PIN 可登录。
- Admin 新增分类成功。
- Admin 更新分类名称成功。
- 停用分类后不出现在 `ListCategories` active 列表。

运行：

```powershell
Set-Location apps\api
go test ./internal/app -run "TestSettings|TestMember|TestCategory|TestChangePIN" -count=1
```

预期：编译失败，因为方法未实现。

- [ ] 增加 DTO 和输入：

```go
type MemberAdminDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

type CreateMemberInput struct {
	Name string `json:"name"`
	Role string `json:"role"`
	PIN  string `json:"pin"`
}

type ResetMemberPINInput struct {
	PIN string `json:"pin"`
}

type ChangeOwnPINInput struct {
	CurrentPIN string `json:"currentPin"`
	NewPIN     string `json:"newPin"`
}

type CreateCategoryInput struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	IconKey  string `json:"iconKey"`
	ColorKey string `json:"colorKey"`
}

type UpdateCategoryInput struct {
	Name      string `json:"name"`
	IconKey   string `json:"iconKey"`
	ColorKey  string `json:"colorKey"`
	SortOrder int32  `json:"sortOrder"`
}
```

- [ ] 增加 service 方法：

```go
ListMembers(ctx, actor MemberDTO) ([]MemberAdminDTO, error)
CreateMember(ctx, actor MemberDTO, input CreateMemberInput) (MemberAdminDTO, error)
DisableMember(ctx, actor MemberDTO, id string) (MemberAdminDTO, error)
ResetMemberPIN(ctx, actor MemberDTO, id string, input ResetMemberPINInput) error
ChangeOwnPIN(ctx, actor MemberDTO, input ChangeOwnPINInput) error
ListAllCategories(ctx, actor MemberDTO) ([]CategoryDTO, error)
CreateCategory(ctx, actor MemberDTO, input CreateCategoryInput) (CategoryDTO, error)
UpdateCategory(ctx, actor MemberDTO, id string, input UpdateCategoryInput) (CategoryDTO, error)
DisableCategory(ctx, actor MemberDTO, id string) (CategoryDTO, error)
```

规则：

- `actor.Role != "admin"` 的 Admin-only 方法返回权限错误。
- 不能停用最后一个 active admin。
- 新增成员 role 只允许 `admin` 或 `member`。
- 新增/更新分类名称不能为空。
- 新增分类 type 只允许 `expense` 或 `income`。
- 修改自己的 PIN 时必须验证当前 PIN。

- [ ] 运行：

```powershell
Set-Location apps\api
go test ./internal/app -count=1
go test ./... -count=1
```

- [ ] 提交：

```powershell
git add apps/api/internal/app/service.go apps/api/internal/app/service_test.go
git commit -m "feat: add settings service"
```

## Task 3：HTTP API

**文件：**

- 修改：`apps/api/internal/http/api.go`
- 修改：`apps/api/internal/http/api_test.go`

- [ ] 增加路由：

```go
r.Get("/members", api.members)
r.Post("/members", api.createMember)
r.Post("/members/{id}/disable", api.disableMember)
r.Post("/members/{id}/reset-pin", api.resetMemberPIN)
r.Post("/me/change-pin", api.changeOwnPIN)
r.Post("/categories", api.createCategory)
r.Patch("/categories/{id}", api.updateCategory)
r.Post("/categories/{id}/disable", api.disableCategory)
```

- [ ] 每个 handler 先 `requireSession`，再调用 service。
- [ ] HTTP 测试覆盖：

  - 未登录访问设置 API 返回 401。
  - member 访问 Admin-only API 返回 403 或 400 中带权限错误。
  - admin 新增成员返回 201。
  - 当前成员修改 PIN 返回 200。

- [ ] 运行：

```powershell
Set-Location apps\api
go test ./internal/http -count=1
go test ./... -count=1
```

- [ ] 提交：

```powershell
git add apps/api/internal/http/api.go apps/api/internal/http/api_test.go
git commit -m "feat: expose settings api"
```

## Task 4：前端 API 和 Settings 页面

**文件：**

- 修改：`apps/web/src/api.ts`
- 修改：`apps/web/src/App.tsx`

- [ ] 在 `api.ts` 增加成员、分类管理和修改 PIN 的类型与函数。
- [ ] Settings 页面按角色渲染：

  - Admin：成员管理、分类管理、修改自己的 PIN。
  - Member：当前成员信息、修改自己的 PIN。

- [ ] 成员管理支持：

  - 展示成员列表。
  - 新增成员。
  - 停用成员。
  - 重置 PIN。

- [ ] 分类管理支持：

  - 展示分类列表。
  - 新增分类。
  - 重命名分类。
  - 停用分类。

- [ ] 运行：

```powershell
pnpm --dir apps/web build
task test
```

- [ ] 提交：

```powershell
git add apps/web/src/api.ts apps/web/src/App.tsx
git commit -m "feat: add settings management UI"
```

## Task 5：最终验证

- [ ] 运行：

```powershell
task test
.\scripts\build.ps1
```

- [ ] 本地手动检查：

```powershell
.\scripts\start-dev.ps1
```

检查：

- Admin 可以新增成员。
- 新成员可以登录。
- Admin 可以重置成员 PIN。
- 停用成员后不能登录。
- Admin 可以新增分类。
- 停用分类后 Add 页面不再显示该分类。
- 普通成员 Settings 不显示管理区。
- 普通成员可以修改自己的 PIN。

- [ ] 推送分支：

```powershell
git status --short --branch
git push -u origin feature/settings-foundation
```

## 自检

- 本计划不新增数据库表。
- 所有权限由后端 service 校验。
- 删除操作统一用停用实现。
- 不包含预算和账户功能。
