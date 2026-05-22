# Commune Ledger Core Design

Date: 2026-05-22

## Background

Commune is a self-hosted family bookkeeping application for one household, one deployment, and a small server. Auth and dev tooling are already in place. The next step is to make daily bookkeeping actually useful by adding the core ledger data model.

This phase intentionally excludes account tracking. A transaction records what happened, who recorded it, when it happened, and which category it belongs to.

## Goal

Build the first usable ledger core: categories, transactions, and monthly overview data that power the Add and Transactions flows.

## Scope

This phase includes:

- Category storage and listing.
- Transaction creation, update, deletion, and filtering.
- Monthly income, expense, and balance summary.
- Category totals for the current month.
- Household-wide visibility with role-based edit permissions.
- Frontend wiring for Add and Transactions screens.

This phase does not include:

- Accounts such as WeChat, Alipay, banks, or cash boxes.
- Budget editing UI.
- Advanced reporting.
- Transfer transactions.
- Recurring transactions.
- Receipt attachments.
- Import/export.

## Product Rules

- All amounts are stored as integer cents.
- Transactions belong to one category and one member.
- Transactions are household-visible by default.
- Admins can edit or delete any transaction.
- Members can only edit or delete transactions they created.
- Disabled categories remain visible on historical transactions.
- Disabled members remain visible on historical transactions.

## Data Model

### Category

Fields:

- ID
- Name
- Type: `expense` or `income`
- Icon key
- Color key
- Sort order
- Active flag
- System default flag
- Created and updated timestamps

System default categories are inserted during initialization and can be disabled, renamed, or reordered later.

### Transaction

Fields:

- ID
- Type: `expense` or `income`
- Amount in cents
- Category ID
- Member ID
- Transaction date
- Note
- Created and updated timestamps

Transactions are month-filtered on the server by `YYYY-MM`.

### Monthly Overview

Computed data returned by the API:

- Total income
- Total expense
- Net balance
- Per-category expense totals
- Recent transactions

This is derived from transaction rows and is not stored as a separate table.

## API Shape

New endpoints:

- `GET /api/categories`
- `GET /api/transactions?month=YYYY-MM`
- `POST /api/transactions`
- `PATCH /api/transactions/{id}`
- `DELETE /api/transactions/{id}`
- `GET /api/overview/monthly?month=YYYY-MM`

The server performs permission checks and returns concise JSON errors.

## UI Behavior

### Add Screen

The Add screen is the fastest path to record a transaction.

Primary fields:

- Amount
- Type toggle
- Category selector
- Date
- Note

After submit, the form resets amount and note, keeps the last category if it is still valid, and refreshes monthly overview data.

### Transactions Screen

The Transactions screen shows the current month by default.

It supports:

- Month filter
- Type filter
- Category filter
- Member filter

Each row shows amount, category, member, date, and note.

### Monthly Overview

The overview shows:

- This month income
- This month expense
- Balance
- Top expense categories

## Backend Structure

The existing `apps/api` layout is extended with focused modules:

- `internal/categories` for category listing and maintenance helpers.
- `internal/transactions` for transaction CRUD, authorization, and monthly filtering.
- `internal/overview` or a small extension of the app service for monthly summary queries.
- `internal/db/queries` for sqlc-generated queries.

The implementation should keep SQL explicit and keep authorization checks close to the service layer.

## Testing

Required coverage:

- Default categories are created during initialization.
- Category listing returns active categories.
- Transaction create stores cents, category, member, and date correctly.
- Transaction edit and delete respect ownership rules.
- Admins can edit and delete any transaction.
- Monthly overview totals match inserted transactions.
- Filtered transaction listing respects the selected month.

Frontend coverage can stay narrow:

- Add form submit path.
- Transactions list rendering.
- Overview summary rendering.

## Build Order

1. Database schema and sqlc queries.
2. Backend category and transaction services.
3. Monthly overview query and API.
4. Frontend API wiring.
5. Add screen transaction entry.
6. Transactions list and monthly summary.
7. Verification and cleanup.
