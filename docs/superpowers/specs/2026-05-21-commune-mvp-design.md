# Commune MVP Design

Date: 2026-05-21

## Background

Commune is a self-hosted family bookkeeping application. The name uses the English word "commune" to reflect a shared household ledger rather than a public financial SaaS product.

The first version targets one family, one deployment instance, and a small local or cloud server. The product should solve three primary problems:

- Fast daily bookkeeping.
- Shared family visibility.
- Monthly category budget review.

The MVP should avoid public registration, multi-tenant SaaS concerns, asset management, investment tracking, and complex financial reports.

## Product Shape

The MVP is a single-family, single-instance, mobile-first web application.

There is no public registration flow. An administrator initializes the application, creates family members, and assigns each member a PIN. Members sign in by selecting their name and entering their PIN.

The first screen after sign-in is the fast bookkeeping screen. The user should be able to add a transaction in a few seconds by entering an amount, choosing income or expense, selecting a category, and submitting.

The application has four main areas:

- Add: fast transaction entry and a small monthly overview.
- Transactions: monthly ledger list, filters, edit, and delete.
- Budgets: monthly category budget status.
- Settings: administrator controls for members, categories, and basic household settings.

## Explicit Non-Goals

The MVP will not include:

- Public registration.
- Multiple households.
- Tenant isolation.
- Invite links.
- Email, SMS, or third-party OAuth.
- Bank account balance tracking.
- Credit cards, debts, loans, or investments.
- Multi-currency accounting.
- Invoice, reimbursement, or approval workflows.
- Offline sync.
- Real-time multi-user collaboration.
- Complex annual reports.
- Full data import/export platform.

An administrator backup or CSV export feature can be considered later, but it is not required for the first MVP.

## Users and Permissions

There are two roles:

- Admin
- Member

Admins can:

- Manage members.
- Reset member PINs.
- Disable members.
- Manage categories.
- Set monthly category budgets.
- Create, edit, and delete transactions.
- View all household data.

Members can:

- Sign in with their PIN.
- Add transactions.
- View household transactions and budgets.
- Edit or delete only transactions they created.
- Change their own PIN.

Disabled members cannot sign in. Historical transactions created by disabled members remain visible.

## Core Data Model

### AppSetting

Stores application-level configuration.

Fields:

- Household name.
- Initialization status.
- Default currency, initially fixed to `CNY`.
- Created and updated timestamps.

### Member

Represents a household member.

Fields:

- ID.
- Name.
- PIN hash.
- Role: `admin` or `member`.
- Active status.
- Created and updated timestamps.

PINs are never stored in plaintext.

### Category

Represents income or expense categories.

Fields:

- ID.
- Name.
- Type: `expense`, `income`, or `general`.
- Icon key.
- Color key.
- Sort order.
- Active status.
- System default flag.
- Created and updated timestamps.

The application ships with default categories such as food, groceries, transportation, housing, children, healthcare, entertainment, and other. Admins can rename, disable, add, and reorder categories.

Disabled categories remain linked to historical transactions.

### Transaction

Represents one income or expense record.

Fields:

- ID.
- Type: `expense` or `income`.
- Amount in cents.
- Category ID.
- Member ID.
- Transaction date.
- Note.
- Created and updated timestamps.

Amounts are stored as integer cents to avoid floating-point errors.

### MonthlyBudget

Represents a monthly budget for one expense category.

Fields:

- ID.
- Month, stored as a month date or `YYYY-MM`.
- Category ID.
- Budget amount in cents.
- Created and updated timestamps.

Budgets only apply to expense categories. Categories without a budget are still included in statistics but do not trigger budget warnings.

There must be at most one budget per month and category.

### Session

Represents a signed-in member session.

Fields:

- ID.
- Member ID.
- Token hash.
- Expiration timestamp.
- Created timestamp.

Sessions can be implemented with signed cookies, server-side session storage, or a hybrid approach. The MVP should prefer a simple server-side session table to support logout and future session invalidation.

## Main Flows

### Initialization

When the application has not been initialized, it shows an initialization screen.

The administrator enters:

- Household name.
- Admin name.
- Admin PIN.

After initialization:

- The admin member is created.
- Default categories are inserted.
- The application moves to normal sign-in mode.

There is no public registration screen after initialization.

### Sign-In

The member selects their name and enters a PIN.

On success:

- A session is created.
- The user is redirected to the Add screen.

On failure:

- The app shows a concise error.
- No account enumeration beyond the member selector is needed because this is a single-family deployment.

### Add Transaction

Defaults:

- Type defaults to expense.
- Date defaults to today.
- Member is the signed-in member.
- Category is required.
- Note is optional.

After a successful submission:

- Amount and note are cleared.
- The user remains on the Add screen.
- Monthly overview and recent transaction data update.
- The app shows a lightweight success message.

### View and Manage Transactions

The Transactions screen defaults to the current month.

The user can filter by:

- Month.
- Type.
- Category.
- Member.

Transactions are grouped by date. Each row shows amount, category, member, note, and date.

Editing opens a detail screen or bottom drawer. The editable fields are amount, type, category, date, and note.

Members can edit or delete their own transactions. Admins can edit or delete all transactions.

### Budgets

The Budgets screen defaults to the current month.

For each expense category, the app shows:

- Category name.
- Amount spent.
- Budget amount, if configured.
- Remaining or over-budget amount.
- Usage ratio.
- Status: normal, near limit, or over budget.

Over-budget and near-limit categories should be visually prioritized.

Admins can set or update monthly category budgets. Members can view budgets but cannot edit them.

### Settings

Admin settings include:

- Household name.
- Member management.
- Category management.
- PIN reset.
- Member disablement.

Member settings include:

- Current member name.
- Change own PIN.
- Logout.

## UI Direction

The UI is mobile-first.

The main navigation uses a bottom tab bar:

- Add.
- Transactions.
- Budgets.
- Settings.

Desktop layout can use a centered application shell with a maximum width. A complex desktop dashboard is not required for the MVP.

The Add screen should optimize for fast one-handed entry:

- Large amount input.
- Clear income/expense segmented control.
- Category grid.
- Optional note and date fields placed below primary controls.
- Submit button reachable near the bottom.

The application should feel like a small mobile app rather than a traditional admin panel.

## Technical Architecture

The project will use a monorepo structure.

Recommended stack:

- Backend: Go.
- HTTP router: Chi.
- Database: PostgreSQL.
- Database driver: pgx.
- SQL code generation: sqlc.
- Frontend: React + Vite.
- Styling: Tailwind CSS.
- UI primitives: Radix UI.
- Icons: Lucide React.
- Deployment: Docker Compose.

This keeps the server runtime small while allowing a smooth SPA-style frontend.

The React frontend is built into static assets. Production does not require a Node.js server. The Go service serves the frontend static files and exposes JSON APIs.

## Monorepo Layout

Proposed repository layout:

```text
commune/
├─ apps/
│  ├─ api/
│  │  ├─ cmd/server/
│  │  ├─ internal/
│  │  │  ├─ auth/
│  │  │  ├─ budgets/
│  │  │  ├─ categories/
│  │  │  ├─ config/
│  │  │  ├─ db/
│  │  │  ├─ http/
│  │  │  ├─ members/
│  │  │  └─ transactions/
│  │  ├─ migrations/
│  │  ├─ queries/
│  │  ├─ sqlc.yaml
│  │  └─ go.mod
│  │
│  └─ web/
│     ├─ src/
│     ├─ public/
│     ├─ package.json
│     ├─ vite.config.ts
│     └─ tailwind.config.ts
│
├─ packages/
│  └─ shared/
│
├─ deploy/
│  ├─ compose.yml
│  ├─ Caddyfile
│  └─ systemd/
│
├─ docs/
│  └─ superpowers/specs/
│
├─ scripts/
│  ├─ dev.ps1
│  ├─ build.ps1
│  └─ migrate.ps1
│
├─ .env.example
├─ AGENTS.md
└─ README.md
```

`packages/shared` is optional for the MVP. It can later hold generated TypeScript API types or OpenAPI artifacts. Shared business logic should not be introduced prematurely.

## Backend Modules

### `cmd/server`

Application entry point. It loads configuration, opens the database connection pool, registers HTTP routes, serves static frontend assets, and starts the HTTP server.

### `internal/config`

Loads environment variables and validates required configuration such as:

- HTTP port.
- Database URL.
- Session secret.
- Static asset path or embedded asset mode.

### `internal/db`

Owns PostgreSQL connection setup, migration integration, transaction helpers, and sqlc generated query wiring.

### `internal/auth`

Owns PIN verification, password hashing, session creation, session lookup, logout, and authorization helpers.

### `internal/members`

Owns member creation, update, disablement, PIN reset, and member listing.

### `internal/categories`

Owns default category seeding, category management, sorting, disabling, and category lookup.

### `internal/transactions`

Owns transaction creation, editing, deletion, filtering, and permission checks around transaction ownership.

### `internal/budgets`

Owns monthly category budgets and budget usage calculations.

### `internal/http`

Owns route registration, request decoding, response encoding, middleware, API error format, and static frontend fallback.

## API Shape

The MVP can start with JSON APIs:

- `POST /api/init`
- `GET /api/session`
- `POST /api/login`
- `POST /api/logout`
- `GET /api/members`
- `POST /api/members`
- `PATCH /api/members/{id}`
- `POST /api/members/{id}/reset-pin`
- `GET /api/categories`
- `POST /api/categories`
- `PATCH /api/categories/{id}`
- `GET /api/transactions`
- `POST /api/transactions`
- `PATCH /api/transactions/{id}`
- `DELETE /api/transactions/{id}`
- `GET /api/budgets`
- `PUT /api/budgets/{month}/{categoryId}`
- `GET /api/overview/monthly`

The exact API can be refined during implementation, but it should keep authorization checks server-side.

## Database and SQL Approach

PostgreSQL is used instead of SQLite because the project should use a standard server database with strong constraints, mature backup tools, better concurrency, and easier future expansion.

The backend should use pgx for PostgreSQL connectivity and sqlc for type-safe generated Go code from handwritten SQL.

Reasons:

- SQL remains explicit and reviewable.
- Generated Go methods reduce repetitive row scanning code.
- Compile-time types catch many data access mistakes.
- Complex budget and monthly aggregation queries remain easy to express.
- The project avoids a heavy ORM while keeping maintainable data access.

## Deployment

The default deployment target is Docker Compose:

```text
commune-app
commune-db
```

`commune-app`:

- Runs the Go server.
- Serves the React static build.
- Exposes JSON APIs.

`commune-db`:

- Runs PostgreSQL.
- Stores data in a persistent volume.

A reverse proxy such as Caddy can be provided in `deploy/`, but it is optional for local network deployment.

For very small servers, PostgreSQL settings should be conservative:

- Low connection limit.
- Conservative shared buffers.
- Conservative work memory.

Production should not require a Node.js process. Node or pnpm is only needed to build the frontend.

## Testing Scope

The MVP should include focused tests around money, permissions, and aggregation.

Required backend test coverage:

- PIN login succeeds and fails correctly.
- Disabled members cannot sign in.
- Amount parsing and storage use integer cents.
- Members cannot edit or delete transactions created by others.
- Admins can edit and delete any transaction.
- Creating a transaction updates monthly totals.
- Editing a transaction updates monthly totals.
- Deleting a transaction updates monthly totals.
- Budget usage is calculated per month and category.
- Budget data is isolated by month.
- Disabled categories remain visible on historical transactions.
- Disabled members remain visible on historical transactions.

Frontend tests can start smaller:

- Login form.
- Add transaction form.
- Transaction list filtering.
- Budget status rendering.
- Permission-sensitive settings visibility.

End-to-end tests should cover:

- Initialize app.
- Sign in as admin.
- Create a member.
- Add a transaction.
- Set a budget.
- Verify budget overview updates.
- Sign in as member.
- Verify member cannot access admin-only controls.

## Open Decisions

The following details can be finalized during implementation planning:

- Whether frontend API types are generated from OpenAPI or manually maintained at first.
- Whether Go serves embedded static assets or reads `apps/web/dist` from disk.
- Whether database migrations use goose, tern, atlas, or another lightweight migration tool.
- Whether budget warning thresholds are fixed, such as near limit at 80 percent, or configurable.
- Whether Add screen remembers the last selected category after submission.

## Recommended MVP Build Order

1. Monorepo scaffold.
2. PostgreSQL and migration setup.
3. Go server, config, health endpoint.
4. Initialization flow.
5. Auth and session flow.
6. Members and categories.
7. Transactions.
8. Monthly overview and budget calculations.
9. React mobile shell and bottom navigation.
10. Add transaction screen.
11. Transactions screen.
12. Budgets screen.
13. Settings screen.
14. Docker Compose deployment.
15. Focused tests and final verification.
