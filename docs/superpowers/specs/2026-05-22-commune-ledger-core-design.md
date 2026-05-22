# Commune 账本核心设计

日期：2026-05-22

## 背景

Commune 是一个面向单个家庭、单个部署实例和小型服务器的自托管家庭账本应用。当前项目已经完成认证基础和开发工程化。下一步需要让应用具备真正可用的日常记账能力，因此本阶段聚焦账本核心数据模型。

本阶段明确不引入“账户”概念。也就是说，不记录微信、支付宝、银行卡、现金等资金账户。每笔交易只记录发生了什么、由谁记录、何时发生、归属哪个分类。

## 目标

构建第一版可用的账本核心能力：分类、交易流水和月度概览数据，为 Add 页面和 Transactions 页面提供真实业务数据。

## 范围

本阶段包含：

- 分类的存储和列表查询。
- 交易流水的创建、更新、删除和过滤。
- 月度收入、支出和结余汇总。
- 当前月份的分类支出统计。
- 家庭共享可见性，以及基于角色的编辑权限。
- 前端 Add 页面和 Transactions 页面的 API 接入。

本阶段不包含：

- 微信、支付宝、银行卡、现金等账户管理。
- 预算编辑 UI。
- 高级报表。
- 转账交易。
- 周期性交易。
- 小票或附件。
- 导入导出。

## 产品规则

- 所有金额都使用整数分存储。
- 每笔交易属于一个分类和一个成员。
- 交易默认对整个家庭可见。
- Admin 可以编辑或删除任意交易。
- Member 只能编辑或删除自己创建的交易。
- 已停用分类仍然保留在历史交易中并可展示。
- 已停用成员仍然保留在历史交易中并可展示。

## 数据模型

### Category

字段：

- `id`
- `name`
- `type`：`expense` 或 `income`
- `icon_key`
- `color_key`
- `sort_order`
- `active`
- `system_default`
- `created_at` 和 `updated_at`

系统默认分类在应用初始化时插入。后续管理员可以停用、重命名或调整排序，但本阶段先实现默认分类和列表能力。

### Transaction

字段：

- `id`
- `type`：`expense` 或 `income`
- `amount_cents`
- `category_id`
- `member_id`
- `transaction_date`
- `note`
- `created_at` 和 `updated_at`

交易列表按服务端解析的 `YYYY-MM` 月份过滤。

### Monthly Overview

月度概览是由交易流水实时计算出的 API 返回数据，不单独建表存储。

返回内容：

- 本月总收入。
- 本月总支出。
- 本月结余。
- 按分类聚合的支出金额。
- 最近交易。

## API 形态

新增接口：

- `GET /api/categories`
- `GET /api/transactions?month=YYYY-MM`
- `POST /api/transactions`
- `PATCH /api/transactions/{id}`
- `DELETE /api/transactions/{id}`
- `GET /api/overview/monthly?month=YYYY-MM`

权限校验由服务端完成。错误响应保持简洁的 JSON 格式。

## UI 行为

### Add 页面

Add 页面是最快的记账入口。

主要字段：

- 金额。
- 收入 / 支出切换。
- 分类选择。
- 日期。
- 备注。

提交成功后，清空金额和备注。如果上一次选择的分类仍然可用，则保留该分类，并刷新月度概览。

### Transactions 页面

Transactions 页面默认展示当前月份的流水。

支持过滤：

- 月份。
- 类型。
- 分类。
- 成员。

每行展示金额、分类、成员、日期和备注。

### Monthly Overview

月度概览展示：

- 本月收入。
- 本月支出。
- 本月结余。
- 支出分类排行。

## 后端结构

在现有 `apps/api` 结构上扩展聚焦模块：

- `internal/categories`：分类列表和默认分类维护辅助逻辑。
- `internal/transactions`：交易 CRUD、权限判断和月度过滤。
- `internal/overview` 或在现有 app service 中小范围扩展：月度汇总查询。
- `internal/db/queries`：由 sqlc 生成的查询代码。

实现时保持 SQL 显式、可审查；权限检查放在 service 层附近，避免只依赖前端控制。

## 测试范围

后端必须覆盖：

- 初始化应用时创建默认分类。
- 分类列表只返回可用分类。
- 创建交易时正确存储金额分、分类、成员和日期。
- 编辑和删除交易时遵守所有权规则。
- Admin 可以编辑和删除任意交易。
- 月度概览汇总与插入的交易一致。
- 交易列表按指定月份过滤。

前端测试可以保持较窄：

- Add 表单提交路径。
- Transactions 列表渲染。
- 月度概览渲染。

## 构建顺序

1. 数据库 schema 和 sqlc 查询。
2. 后端分类和交易 service。
3. 月度概览查询和 API。
4. 前端 API 接入。
5. Add 页面交易录入。
6. Transactions 列表和月度概览。
7. 验证和清理。
