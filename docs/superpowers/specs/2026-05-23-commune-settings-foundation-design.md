# Commune 设置与管理基础设计

日期：2026-05-23

## 背景

Commune 当前已经具备初始化、PIN 登录、默认分类、交易流水和月度概览。下一步需要让家庭管理员可以维护成员和分类，让普通成员可以维护自己的 PIN。否则应用只能依赖初始化时创建的管理员和默认分类，无法支持真实家庭长期使用。

本阶段聚焦设置基础能力，不实现预算，不实现账户，也不做复杂后台管理。

## 目标

实现第一版 Settings 页面可用能力：成员管理、分类管理和个人 PIN 修改。

## 范围

本阶段包含：

- Admin 查看成员列表。
- Admin 新增成员。
- Admin 停用成员。
- Admin 重置成员 PIN。
- Admin 查看分类列表。
- Admin 新增分类。
- Admin 重命名分类。
- Admin 停用分类。
- 当前成员修改自己的 PIN。
- Settings 页面按角色展示不同内容。

本阶段不包含：

- 删除成员。
- 删除分类。
- 成员头像。
- 邀请链接。
- 邮箱、手机号、OAuth。
- 分类拖拽排序。
- 预算设置。
- 多家庭或多租户。

## 产品规则

- 只有 Admin 可以管理成员和分类。
- Member 只能修改自己的 PIN。
- 成员停用后不能登录。
- 成员停用后，其历史交易仍可展示。
- 分类停用后不能用于新交易。
- 分类停用后，其历史交易仍可展示。
- 分类类型创建后不在本阶段提供修改，避免影响历史交易语义。
- PIN 不返回明文，后端只保存 hash。

## 数据模型影响

现有表已经支持本阶段主要能力：

- `members`
  - `name`
  - `pin_hash`
  - `role`
  - `active`
  - timestamps

- `categories`
  - `name`
  - `type`
  - `icon_key`
  - `color_key`
  - `sort_order`
  - `active`
  - `system_default`
  - timestamps

本阶段不新增数据库表。只补充 sqlc 查询和 service 方法。

## API 形态

新增接口：

- `GET /api/members`
- `POST /api/members`
- `POST /api/members/{id}/disable`
- `POST /api/members/{id}/reset-pin`
- `POST /api/me/change-pin`
- `POST /api/categories`
- `PATCH /api/categories/{id}`
- `POST /api/categories/{id}/disable`

已有 `GET /api/categories` 继续复用，但需要确认 Admin 设置中能看到 active 分类。历史停用分类展示由交易列表继续通过交易 join 支持。

## 权限

服务端必须校验权限，不依赖前端隐藏按钮。

Admin-only：

- 成员列表。
- 新增成员。
- 停用成员。
- 重置任意成员 PIN。
- 新增分类。
- 更新分类名称、图标、颜色和排序。
- 停用分类。

Member 可用：

- 修改自己的 PIN。
- 退出登录。

## UI 行为

### Admin Settings

Admin Settings 显示两个区域：

- 成员管理。
- 分类管理。

成员管理：

- 展示成员名、角色、状态。
- 新增成员：姓名、角色、初始 PIN。
- 停用成员：二次确认可以后续补，本阶段先使用明确按钮。
- 重置 PIN：输入新 PIN 后提交。

分类管理：

- 展示分类名、类型、状态。
- 新增分类：名称、类型。
- 重命名分类。
- 停用分类。

### Member Settings

普通成员 Settings 只显示：

- 当前成员信息。
- 修改自己的 PIN。
- 退出登录。

## 错误处理

- 权限不足返回 403。
- 未登录返回 401。
- 输入不合法返回 400。
- 成员名、分类名不能为空。
- PIN 复用现有 PIN 校验规则。
- 不能停用最后一个 active admin，避免系统失去管理员。

## 测试范围

后端必须覆盖：

- Member 不能访问 Admin-only API。
- Admin 可以新增 member。
- 停用 member 后不能登录。
- 不能停用最后一个 active admin。
- Admin 可以重置 member PIN。
- 当前成员可以修改自己的 PIN。
- Admin 可以新增分类。
- Admin 可以更新分类名称。
- 停用分类后分类不再出现在 active 分类列表。

前端验证：

- Admin Settings 能看到成员管理和分类管理。
- Member Settings 不能看到管理区。
- Admin 可以通过 Settings 新增成员和分类。
- 当前成员可以修改 PIN。

## 构建顺序

1. 成员和分类 sqlc 查询。
2. 后端 Settings service 方法和权限测试。
3. HTTP API。
4. 前端 API 类型和函数。
5. Settings 页面接入成员管理、分类管理和修改 PIN。
6. 完整测试和本地手动验证。
