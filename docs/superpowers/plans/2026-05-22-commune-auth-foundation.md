# Commune Auth Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the initialization and PIN login foundation so a self-hosted Commune instance can create its first admin, sign members in, restore sessions, and sign members out.

**Architecture:** The backend adds a small migration command, authentication schema, sqlc queries, auth services, and JSON API routes. The frontend replaces the placeholder shell with a bootstrap flow that chooses between initialization, login, and the authenticated responsive app shell. This phase intentionally stops before member management, categories, transactions, and budgets.

**Tech Stack:** Go, Chi, pgx, sqlc, PostgreSQL, React, Vite, TypeScript, Tailwind CSS, Radix UI, Docker Compose, PowerShell scripts.

---

## Scope

This plan implements:

- Database migration execution.
- `app_settings`, `members`, and `sessions` tables.
- PIN hashing and verification.
- Session token creation and cookie handling.
- Initialization status API.
- First-run initialization API.
- Login member selector API.
- Login API.
- Current session API.
- Logout API.
- Server-side session invalidation.
- Frontend initialization page.
- Frontend login page.
- Frontend logout action.
- Authenticated responsive app shell.

This plan does not implement:

- Admin member management after initialization.
- Category seed or category management.
- Transaction entry.
- Budgets.
- Authorization for ledger APIs.
- Password reset UI.
- CSRF hardening beyond same-site cookies.

## Target File Structure

Create or modify these files:

```text
apps/api/
├─ cmd/
│  ├─ migrate/main.go
│  └─ server/main.go
├─ internal/
│  ├─ app/
│  │  ├─ service.go
│  │  └─ service_test.go
│  ├─ auth/
│  │  ├─ cookie.go
│  │  ├─ cookie_test.go
│  │  ├─ pin.go
│  │  ├─ pin_test.go
│  │  ├─ token.go
│  │  └─ token_test.go
│  ├─ db/
│  │  ├─ migrate.go
│  │  ├─ migrate_test.go
│  │  └─ queries/
│  │     ├─ app.sql.go
│  │     ├─ members.sql.go
│  │     └─ sessions.sql.go
│  ├─ http/
│  │  ├─ api.go
│  │  ├─ api_test.go
│  │  ├─ handler.go
│  │  └─ handler_test.go
│  └─ testutil/
│     └─ db.go
├─ migrations/
│  └─ 000002_auth_foundation.sql
└─ queries/
   ├─ app.sql
   ├─ members.sql
   └─ sessions.sql

apps/web/src/
├─ api.ts
├─ App.tsx
├─ auth.ts
├─ main.tsx
└─ styles.css

scripts/
└─ migrate.ps1
```

---

## API Contract

Use JSON APIs under `/api`.

### `GET /api/bootstrap`

Response when not initialized:

```json
{
  "initialized": false,
  "session": null,
  "householdName": ""
}
```

Response when initialized and not signed in:

```json
{
  "initialized": true,
  "session": null,
  "householdName": "韩家"
}
```

Response when signed in:

```json
{
  "initialized": true,
  "householdName": "韩家",
  "session": {
    "member": {
      "id": "00000000-0000-0000-0000-000000000001",
      "name": "Han",
      "role": "admin"
    }
  }
}
```

### `POST /api/init`

Request:

```json
{
  "householdName": "韩家",
  "adminName": "Han",
  "pin": "123456"
}
```

Response:

```json
{
  "member": {
    "id": "00000000-0000-0000-0000-000000000001",
    "name": "Han",
    "role": "admin"
  }
}
```

The response sets the session cookie.

### `GET /api/login-members`

Response:

```json
{
  "members": [
    {
      "id": "00000000-0000-0000-0000-000000000001",
      "name": "Han"
    }
  ]
}
```

### `POST /api/login`

Request:

```json
{
  "memberId": "00000000-0000-0000-0000-000000000001",
  "pin": "123456"
}
```

Response:

```json
{
  "member": {
    "id": "00000000-0000-0000-0000-000000000001",
    "name": "Han",
    "role": "admin"
  }
}
```

The response sets the session cookie.

### `GET /api/session`

Response when signed in:

```json
{
  "member": {
    "id": "00000000-0000-0000-0000-000000000001",
    "name": "Han",
    "role": "admin"
  }
}
```

Response when not signed in:

```json
{
  "member": null
}
```

### `POST /api/logout`

Response:

```json
{
  "ok": true
}
```

The response clears the session cookie and invalidates the server-side session.

---

## Task 1: Add Migration Runner

**Files:**
- Create: `apps/api/internal/db/migrate.go`
- Create: `apps/api/internal/db/migrate_test.go`
- Create: `apps/api/cmd/migrate/main.go`
- Modify: `scripts/migrate.ps1`
- Modify: `README.md`

- [ ] **Step 1: Write migration parser test**

Create `apps/api/internal/db/migrate_test.go`:

```go
package db

import "testing"

func TestMigrationVersionFromFilename(t *testing.T) {
	version, err := migrationVersion("000002_auth_foundation.sql")
	if err != nil {
		t.Fatalf("migrationVersion returned error: %v", err)
	}
	if version != 2 {
		t.Fatalf("version = %d, want 2", version)
	}
}

func TestMigrationVersionRejectsInvalidFilename(t *testing.T) {
	_, err := migrationVersion("auth_foundation.sql")
	if err == nil {
		t.Fatal("migrationVersion returned nil error for invalid filename")
	}
}
```

- [ ] **Step 2: Run migration test and verify failure**

Run:

```powershell
Set-Location apps\api
go test ./internal/db
```

Expected: FAIL because `migrationVersion` is not defined.

- [ ] **Step 3: Implement migration runner**

Create `apps/api/internal/db/migrate.go`:

```go
package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migration struct {
	Version int64
	Name    string
	SQL     string
}

func LoadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := migrationVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		path := filepath.Join(dir, entry.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, Migration{
			Version: version,
			Name:    entry.Name(),
			SQL:     string(body),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		var exists bool
		err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, migration.Version).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, migration.SQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", migration.Name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, migration.Version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}

func migrationVersion(name string) (int64, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration filename %q must start with a numeric prefix and underscore", name)
	}
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse migration version from %q: %w", name, err)
	}
	return version, nil
}
```

- [ ] **Step 4: Add migration command**

Create `apps/api/cmd/migrate/main.go`:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/config"
	"github.com/hanzc0106/commune/apps/api/internal/db"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	migrations, err := db.LoadMigrations("migrations")
	if err != nil {
		log.Fatal(err)
	}

	if err := db.RunMigrations(ctx, pool, migrations); err != nil {
		log.Fatal(err)
	}
	log.Printf("applied %d available migrations", len(migrations))
}
```

- [ ] **Step 5: Update migration script**

Replace `scripts/migrate.ps1` with:

```powershell
[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot
Set-Location "$root\apps\api"

go run .\cmd\migrate
```

- [ ] **Step 6: Run backend tests**

Run:

```powershell
Set-Location apps\api
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit migration runner**

```powershell
git add apps/api/internal/db/migrate.go apps/api/internal/db/migrate_test.go apps/api/cmd/migrate/main.go scripts/migrate.ps1 README.md
git commit -m "feat: add database migration runner"
```

---

## Task 2: Add Auth Schema and sqlc Queries

**Files:**
- Create: `apps/api/migrations/000002_auth_foundation.sql`
- Create: `apps/api/queries/app.sql`
- Create: `apps/api/queries/members.sql`
- Create: `apps/api/queries/sessions.sql`
- Generated: `apps/api/internal/db/queries/app.sql.go`
- Generated: `apps/api/internal/db/queries/members.sql.go`
- Generated: `apps/api/internal/db/queries/sessions.sql.go`

- [ ] **Step 1: Create auth migration**

Create `apps/api/migrations/000002_auth_foundation.sql`:

```sql
CREATE TABLE IF NOT EXISTS app_settings (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    household_name TEXT NOT NULL,
    default_currency TEXT NOT NULL DEFAULT 'CNY',
    initialized_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT app_settings_singleton CHECK (id = TRUE),
    CONSTRAINT app_settings_household_name_not_blank CHECK (length(trim(household_name)) > 0),
    CONSTRAINT app_settings_default_currency_not_blank CHECK (length(trim(default_currency)) > 0)
);

CREATE TABLE IF NOT EXISTS members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    pin_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT members_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT members_pin_hash_not_blank CHECK (length(trim(pin_hash)) > 0),
    CONSTRAINT members_role_valid CHECK (role IN ('admin', 'member'))
);

CREATE UNIQUE INDEX IF NOT EXISTS members_active_name_unique
ON members (lower(name))
WHERE active = TRUE;

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT sessions_token_hash_not_blank CHECK (length(trim(token_hash)) > 0)
);

CREATE INDEX IF NOT EXISTS sessions_member_id_idx ON sessions (member_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);
```

- [ ] **Step 2: Create app queries**

Create `apps/api/queries/app.sql`:

```sql
-- name: GetAppSettings :one
SELECT id, household_name, default_currency, initialized_at, created_at, updated_at
FROM app_settings
WHERE id = TRUE;

-- name: AppSettingsExist :one
SELECT EXISTS (SELECT 1 FROM app_settings WHERE id = TRUE)::boolean AS exists;

-- name: CreateAppSettings :one
INSERT INTO app_settings (id, household_name, default_currency)
VALUES (TRUE, $1, 'CNY')
RETURNING id, household_name, default_currency, initialized_at, created_at, updated_at;
```

- [ ] **Step 3: Create member queries**

Create `apps/api/queries/members.sql`:

```sql
-- name: CreateMember :one
INSERT INTO members (name, pin_hash, role, active)
VALUES ($1, $2, $3, TRUE)
RETURNING id, name, pin_hash, role, active, created_at, updated_at;

-- name: GetMemberByID :one
SELECT id, name, pin_hash, role, active, created_at, updated_at
FROM members
WHERE id = $1;

-- name: ListActiveLoginMembers :many
SELECT id, name
FROM members
WHERE active = TRUE
ORDER BY lower(name);
```

- [ ] **Step 4: Create session queries**

Create `apps/api/queries/sessions.sql`:

```sql
-- name: CreateSession :one
INSERT INTO sessions (member_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, member_id, token_hash, expires_at, created_at;

-- name: GetSessionByTokenHash :one
SELECT
    sessions.id,
    sessions.member_id,
    sessions.token_hash,
    sessions.expires_at,
    sessions.created_at,
    members.name AS member_name,
    members.role AS member_role,
    members.active AS member_active
FROM sessions
JOIN members ON members.id = sessions.member_id
WHERE sessions.token_hash = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= now();
```

- [ ] **Step 5: Generate sqlc code**

Run:

```powershell
.\scripts\sqlc.ps1
```

Expected: generated files appear under `apps/api/internal/db/queries`.

- [ ] **Step 6: Run migrations locally**

Run:

```powershell
.\scripts\db-up.ps1
.\scripts\migrate.ps1
```

Expected: migration command exits 0 and logs available migrations.

- [ ] **Step 7: Run backend tests**

Run:

```powershell
Set-Location apps\api
go test ./...
```

Expected: PASS.

- [ ] **Step 8: Commit schema and generated queries**

```powershell
git add apps/api/migrations/000002_auth_foundation.sql apps/api/queries apps/api/internal/db/queries
git commit -m "feat: add auth schema queries"
```

---

## Task 3: Add Auth Primitives

**Files:**
- Create: `apps/api/internal/auth/pin.go`
- Create: `apps/api/internal/auth/pin_test.go`
- Create: `apps/api/internal/auth/token.go`
- Create: `apps/api/internal/auth/token_test.go`
- Create: `apps/api/internal/auth/cookie.go`
- Create: `apps/api/internal/auth/cookie_test.go`
- Modify: `apps/api/go.mod`
- Modify: `apps/api/go.sum`

- [ ] **Step 1: Add crypto dependency**

Run:

```powershell
Set-Location apps\api
go get golang.org/x/crypto@latest
```

Expected: `go.mod` includes `golang.org/x/crypto`.

- [ ] **Step 2: Write PIN tests**

Create `apps/api/internal/auth/pin_test.go`:

```go
package auth

import "testing"

func TestHashPINAndVerifyPIN(t *testing.T) {
	hash, err := HashPIN("123456")
	if err != nil {
		t.Fatalf("HashPIN returned error: %v", err)
	}
	if hash == "123456" {
		t.Fatal("HashPIN returned plaintext PIN")
	}
	if !VerifyPIN(hash, "123456") {
		t.Fatal("VerifyPIN returned false for correct PIN")
	}
	if VerifyPIN(hash, "000000") {
		t.Fatal("VerifyPIN returned true for incorrect PIN")
	}
}

func TestHashPINRejectsShortPIN(t *testing.T) {
	_, err := HashPIN("123")
	if err == nil {
		t.Fatal("HashPIN returned nil error for short PIN")
	}
}
```

- [ ] **Step 3: Run PIN tests and verify failure**

Run:

```powershell
go test ./internal/auth
```

Expected: FAIL because `HashPIN` and `VerifyPIN` are not defined.

- [ ] **Step 4: Implement PIN hashing**

Create `apps/api/internal/auth/pin.go`:

```go
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	pinSaltSize = 16
	pinKeySize  = 32
	pinTime     = 1
	pinMemory   = 64 * 1024
	pinThreads  = 4
)

func HashPIN(pin string) (string, error) {
	if len(pin) < 4 {
		return "", errors.New("PIN must be at least 4 characters")
	}
	salt := make([]byte, pinSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pin), salt, pinTime, pinMemory, pinThreads, pinKeySize)
	return fmt.Sprintf(
		"argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		pinMemory,
		pinTime,
		pinThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPIN(encodedHash string, pin string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "argon2id" {
		return false
	}

	var memory uint32
	var time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[2], "v=%d", new(int)); err != nil {
		return false
	}
	for _, param := range strings.Split(parts[3], ",") {
		key, value, ok := strings.Cut(param, "=")
		if !ok {
			return false
		}
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return false
		}
		switch key {
		case "m":
			memory = uint32(parsed)
		case "t":
			time = uint32(parsed)
		case "p":
			threads = uint8(parsed)
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actualKey := argon2.IDKey([]byte(pin), salt, time, memory, threads, uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1
}
```

- [ ] **Step 5: Write token tests**

Create `apps/api/internal/auth/token_test.go`:

```go
package auth

import "testing"

func TestNewSessionTokenAndHash(t *testing.T) {
	token, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken returned error: %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token length = %d, want at least 32", len(token))
	}
	hash := HashSessionToken(token)
	if hash == token {
		t.Fatal("HashSessionToken returned raw token")
	}
	if HashSessionToken(token) != hash {
		t.Fatal("HashSessionToken is not deterministic")
	}
}
```

- [ ] **Step 6: Implement token helpers**

Create `apps/api/internal/auth/token.go`:

```go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func NewSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 7: Write cookie tests**

Create `apps/api/internal/auth/cookie_test.go`:

```go
package auth

import (
	"net/http"
	"testing"
	"time"
)

func TestSessionCookie(t *testing.T) {
	cookie := SessionCookie("token", time.Unix(100, 0))
	if cookie.Name != SessionCookieName {
		t.Fatalf("Name = %q", cookie.Name)
	}
	if cookie.Value != "token" {
		t.Fatalf("Value = %q", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Fatalf("Path = %q", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Fatal("HttpOnly = false")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v", cookie.SameSite)
	}
}

func TestClearSessionCookie(t *testing.T) {
	cookie := ClearSessionCookie()
	if cookie.Name != SessionCookieName {
		t.Fatalf("Name = %q", cookie.Name)
	}
	if cookie.Value != "" {
		t.Fatalf("Value = %q", cookie.Value)
	}
	if cookie.MaxAge != -1 {
		t.Fatalf("MaxAge = %d, want -1", cookie.MaxAge)
	}
}
```

- [ ] **Step 8: Implement cookie helpers**

Create `apps/api/internal/auth/cookie.go`:

```go
package auth

import (
	"net/http"
	"time"
)

const SessionCookieName = "commune_session"

func SessionCookie(token string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
```

- [ ] **Step 9: Run auth tests**

Run:

```powershell
go test ./internal/auth
```

Expected: PASS.

- [ ] **Step 10: Commit auth primitives**

```powershell
git add apps/api/internal/auth apps/api/go.mod apps/api/go.sum
git commit -m "feat: add auth primitives"
```

---

## Task 4: Add Backend App Service

**Files:**
- Create: `apps/api/internal/app/service.go`
- Create: `apps/api/internal/app/service_test.go`
- Create: `apps/api/internal/testutil/db.go`

- [ ] **Step 1: Create database test helper**

Create `apps/api/internal/testutil/db.go`:

```go
package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func OpenTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("COMMUNE_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://commune:commune@localhost:5432/commune?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		t.Skipf("test database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
```

- [ ] **Step 2: Write service test**

Create `apps/api/internal/app/service_test.go`:

```go
package app

import (
	"context"
	"testing"

	"github.com/hanzc0106/commune/apps/api/internal/db"
	"github.com/hanzc0106/commune/apps/api/internal/db/queries"
	"github.com/hanzc0106/commune/apps/api/internal/testutil"
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

	_, _ = pool.Exec(ctx, "TRUNCATE sessions, members, app_settings RESTART IDENTITY CASCADE")

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
```

- [ ] **Step 3: Run service test and verify failure**

Run:

```powershell
go test ./internal/app
```

Expected: FAIL because `NewService`, `Initialize`, and related types are not defined.

- [ ] **Step 4: Implement app service**

Create `apps/api/internal/app/service.go`:

```go
package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/auth"
	"github.com/hanzc0106/commune/apps/api/internal/db/queries"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool    *pgxpool.Pool
	queries *queries.Queries
}

type MemberDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type InitializeInput struct {
	HouseholdName string
	AdminName     string
	PIN           string
}

type InitializeResult struct {
	Member MemberDTO `json:"member"`
}

type BootstrapResult struct {
	Initialized   bool        `json:"initialized"`
	HouseholdName string      `json:"householdName"`
	Session       *SessionDTO `json:"session"`
}

type SessionDTO struct {
	Member MemberDTO `json:"member"`
}

type LoginMemberDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LoginInput struct {
	MemberID string
	PIN      string
}

type LoginResult struct {
	Member MemberDTO `json:"member"`
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:    pool,
		queries: queries.New(pool),
	}
}

func (s *Service) Initialize(ctx context.Context, input InitializeInput) (InitializeResult, string, error) {
	householdName := strings.TrimSpace(input.HouseholdName)
	adminName := strings.TrimSpace(input.AdminName)
	if householdName == "" {
		return InitializeResult{}, "", errors.New("household name is required")
	}
	if adminName == "" {
		return InitializeResult{}, "", errors.New("admin name is required")
	}

	exists, err := s.queries.AppSettingsExist(ctx)
	if err != nil {
		return InitializeResult{}, "", err
	}
	if exists {
		return InitializeResult{}, "", errors.New("application is already initialized")
	}

	pinHash, err := auth.HashPIN(input.PIN)
	if err != nil {
		return InitializeResult{}, "", err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return InitializeResult{}, "", err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)
	if _, err := qtx.CreateAppSettings(ctx, householdName); err != nil {
		return InitializeResult{}, "", err
	}
	member, err := qtx.CreateMember(ctx, queries.CreateMemberParams{
		Name:    adminName,
		PinHash: pinHash,
		Role:    "admin",
	})
	if err != nil {
		return InitializeResult{}, "", err
	}

	token, err := auth.NewSessionToken()
	if err != nil {
		return InitializeResult{}, "", err
	}
	if _, err := qtx.CreateSession(ctx, queries.CreateSessionParams{
		MemberID:  member.ID,
		TokenHash: auth.HashSessionToken(token),
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
	}); err != nil {
		return InitializeResult{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return InitializeResult{}, "", err
	}

	return InitializeResult{
		Member: MemberDTO{
			ID:   member.ID.String(),
			Name: member.Name,
			Role: member.Role,
		},
	}, token, nil
}

func (s *Service) Bootstrap(ctx context.Context, rawToken string) (BootstrapResult, error) {
	exists, err := s.queries.AppSettingsExist(ctx)
	if err != nil {
		return BootstrapResult{}, err
	}
	if !exists {
		return BootstrapResult{
			Initialized:   false,
			HouseholdName: "",
			Session:       nil,
		}, nil
	}
	settings, err := s.queries.GetAppSettings(ctx)
	if err != nil {
		return BootstrapResult{}, err
	}
	result := BootstrapResult{
		Initialized:   true,
		HouseholdName: settings.HouseholdName,
		Session:       nil,
	}
	if rawToken == "" {
		return result, nil
	}
	session, err := s.SessionFromToken(ctx, rawToken)
	if err != nil {
		return result, nil
	}
	result.Session = &SessionDTO{Member: session.Member}
	return result, nil
}

func (s *Service) ListLoginMembers(ctx context.Context) ([]LoginMemberDTO, error) {
	rows, err := s.queries.ListActiveLoginMembers(ctx)
	if err != nil {
		return nil, err
	}
	members := make([]LoginMemberDTO, 0, len(rows))
	for _, row := range rows {
		members = append(members, LoginMemberDTO{
			ID:   row.ID.String(),
			Name: row.Name,
		})
	}
	return members, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, string, error) {
	var memberID pgtype.UUID
	if err := memberID.Scan(input.MemberID); err != nil {
		return LoginResult{}, "", errors.New("invalid member ID")
	}
	member, err := s.queries.GetMemberByID(ctx, memberID)
	if err != nil {
		return LoginResult{}, "", errors.New("invalid member or PIN")
	}
	if !member.Active || !auth.VerifyPIN(member.PinHash, input.PIN) {
		return LoginResult{}, "", errors.New("invalid member or PIN")
	}
	token, err := auth.NewSessionToken()
	if err != nil {
		return LoginResult{}, "", err
	}
	if _, err := s.queries.CreateSession(ctx, queries.CreateSessionParams{
		MemberID:  member.ID,
		TokenHash: auth.HashSessionToken(token),
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
	}); err != nil {
		return LoginResult{}, "", err
	}
	return LoginResult{
		Member: MemberDTO{
			ID:   member.ID.String(),
			Name: member.Name,
			Role: member.Role,
		},
	}, token, nil
}

func (s *Service) SessionFromToken(ctx context.Context, rawToken string) (SessionDTO, error) {
	session, err := s.queries.GetSessionByTokenHash(ctx, auth.HashSessionToken(rawToken))
	if err != nil {
		return SessionDTO{}, err
	}
	if !session.MemberActive || session.ExpiresAt.Time.Before(time.Now()) {
		return SessionDTO{}, errors.New("session expired")
	}
	return SessionDTO{
		Member: MemberDTO{
			ID:   session.MemberID.String(),
			Name: session.MemberName,
			Role: session.MemberRole,
		},
	}, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.queries.DeleteSessionByTokenHash(ctx, auth.HashSessionToken(rawToken))
}
```

- [ ] **Step 5: Add login service test**

Append to `apps/api/internal/app/service_test.go`:

```go
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
	_, _ = pool.Exec(ctx, "TRUNCATE sessions, members, app_settings RESTART IDENTITY CASCADE")

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
```

- [ ] **Step 6: Run service tests**

Run:

```powershell
go test ./internal/app
```

Expected: PASS or SKIP when PostgreSQL is not running.

- [ ] **Step 7: Commit app service**

```powershell
git add apps/api/internal/app apps/api/internal/testutil
git commit -m "feat: add auth application service"
```

---

## Task 5: Add Auth HTTP API

**Files:**
- Create: `apps/api/internal/http/api.go`
- Create: `apps/api/internal/http/api_test.go`
- Modify: `apps/api/internal/http/handler.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: Write handler construction test**

Append to `apps/api/internal/http/handler_test.go`:

```go
func TestAPIPathWithoutAPIHandlerReturnsNotFound(t *testing.T) {
	handler := NewHandler(Options{})
	req := httptest.NewRequest("GET", "/api/bootstrap", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
```

- [ ] **Step 2: Extend handler options**

Modify `apps/api/internal/http/handler.go` so `Options` contains an optional `APIHandler`:

```go
type Options struct {
	StaticDir  string
	APIHandler stdhttp.Handler
}
```

Add this route before the static fallback:

```go
if options.APIHandler != nil {
	r.Mount("/api", options.APIHandler)
}
```

- [ ] **Step 3: Create API handler**

Create `apps/api/internal/http/api.go`:

```go
package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hanzc0106/commune/apps/api/internal/app"
	"github.com/hanzc0106/commune/apps/api/internal/auth"
)

type API struct {
	service *app.Service
}

func NewAPI(service *app.Service) stdhttp.Handler {
	api := &API{service: service}
	r := chi.NewRouter()
	r.Get("/bootstrap", api.bootstrap)
	r.Post("/init", api.init)
	r.Get("/login-members", api.loginMembers)
	r.Post("/login", api.login)
	r.Get("/session", api.session)
	r.Post("/logout", api.logout)
	return r
}

func (api *API) bootstrap(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	token := sessionTokenFromRequest(r)
	result, err := api.service.Bootstrap(r.Context(), token)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "bootstrap failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, result)
}

func (api *API) init(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var input app.InitializeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	result, token, err := api.service.Initialize(r.Context(), input)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	stdhttp.SetCookie(w, auth.SessionCookie(token, sessionExpiresAt()))
	writeJSON(w, stdhttp.StatusCreated, result)
}

func (api *API) loginMembers(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	members, err := api.service.ListLoginMembers(r.Context())
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "load members failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"members": members})
}

func (api *API) login(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var input struct {
		MemberID string `json:"memberId"`
		PIN      string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	result, token, err := api.service.Login(r.Context(), app.LoginInput{
		MemberID: input.MemberID,
		PIN:      input.PIN,
	})
	if err != nil {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "invalid member or PIN"})
		return
	}
	stdhttp.SetCookie(w, auth.SessionCookie(token, sessionExpiresAt()))
	writeJSON(w, stdhttp.StatusOK, result)
}

func (api *API) session(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	token := sessionTokenFromRequest(r)
	if token == "" {
		writeJSON(w, stdhttp.StatusOK, map[string]any{"member": nil})
		return
	}
	session, err := api.service.SessionFromToken(r.Context(), token)
	if err != nil {
		writeJSON(w, stdhttp.StatusOK, map[string]any{"member": nil})
		return
	}
	writeJSON(w, stdhttp.StatusOK, session)
}

func (api *API) logout(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	_ = api.service.Logout(r.Context(), sessionTokenFromRequest(r))
	stdhttp.SetCookie(w, auth.ClearSessionCookie())
	writeJSON(w, stdhttp.StatusOK, map[string]bool{"ok": true})
}

func writeJSON(w stdhttp.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func sessionTokenFromRequest(r *stdhttp.Request) string {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
```

- [ ] **Step 4: Add session expiry helper to API file**

Append to `apps/api/internal/http/api.go`:

```go
func sessionExpiresAt() time.Time {
	return time.Now().Add(30 * 24 * time.Hour)
}
```

Add `time` to the import list.

- [ ] **Step 5: Wire API in server**

Modify `apps/api/cmd/server/main.go`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

pool, err := db.Open(ctx, cfg.DatabaseURL)
if err != nil {
	log.Fatal(err)
}
defer pool.Close()

service := app.NewService(pool)
handler := apphttp.NewHandler(apphttp.Options{
	StaticDir: cfg.StaticDir,
	APIHandler: apphttp.NewAPI(service),
})
```

Add imports:

```go
"context"
"time"

"github.com/hanzc0106/commune/apps/api/internal/app"
"github.com/hanzc0106/commune/apps/api/internal/db"
```

- [ ] **Step 6: Run backend tests**

Run:

```powershell
Set-Location apps\api
go test ./...
```

Expected: PASS or database-dependent app tests SKIP when PostgreSQL is unavailable.

- [ ] **Step 7: Commit API routes**

```powershell
git add apps/api/cmd/server/main.go apps/api/internal/http
git commit -m "feat: add auth api routes"
```

---

## Task 6: Add Frontend Bootstrap, Init, and Login Screens

**Files:**
- Create: `apps/web/src/api.ts`
- Create: `apps/web/src/auth.ts`
- Modify: `apps/web/src/App.tsx`

- [ ] **Step 1: Create frontend API client**

Create `apps/web/src/api.ts`:

```ts
export type Member = {
  id: string;
  name: string;
  role: "admin" | "member";
};

export type BootstrapResponse = {
  initialized: boolean;
  householdName: string;
  session: null | {
    member: Member;
  };
};

export async function getBootstrap(): Promise<BootstrapResponse> {
  return request<BootstrapResponse>("/api/bootstrap");
}

export async function initializeApp(input: {
  householdName: string;
  adminName: string;
  pin: string;
}): Promise<{ member: Member }> {
  return request<{ member: Member }>("/api/init", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function listLoginMembers(): Promise<{ members: Array<{ id: string; name: string }> }> {
  return request<{ members: Array<{ id: string; name: string }> }>("/api/login-members");
}

export async function login(input: { memberId: string; pin: string }): Promise<{ member: Member }> {
  return request<{ member: Member }>("/api/login", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function logout(): Promise<{ ok: true }> {
  return request<{ ok: true }>("/api/logout", {
    method: "POST",
    body: JSON.stringify({})
  });
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers
    }
  });
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error ?? "请求失败");
  }
  return data as T;
}
```

- [ ] **Step 2: Create auth state helpers**

Create `apps/web/src/auth.ts`:

```ts
import type { BootstrapResponse, Member } from "./api";

export type AppState =
  | { status: "loading" }
  | { status: "needs-init" }
  | { status: "needs-login"; householdName: string }
  | { status: "authenticated"; householdName: string; member: Member }
  | { status: "error"; message: string };

export function bootstrapToState(bootstrap: BootstrapResponse): AppState {
  if (!bootstrap.initialized) {
    return { status: "needs-init" };
  }
  if (bootstrap.session?.member) {
    return {
      status: "authenticated",
      householdName: bootstrap.householdName,
      member: bootstrap.session.member
    };
  }
  return {
    status: "needs-login",
    householdName: bootstrap.householdName
  };
}
```

- [ ] **Step 3: Replace App with bootstrap flow**

Modify `apps/web/src/App.tsx` so:

- It calls `getBootstrap()` in `useEffect`.
- It renders a loading screen while loading.
- It renders an initialization form when `status === "needs-init"`.
- It renders the existing responsive shell when `status === "authenticated"`.
- It renders a login form when `status === "needs-login"`.

Use these component names:

```tsx
function LoadingScreen()
function InitScreen()
function LoginScreen({ householdName, onLogin }: { householdName: string; onLogin: (member: Member) => void })
function AuthenticatedShell({ icon, member, householdName, onLogout }: { icon: ReactNode; member: Member; householdName: string; onLogout: () => void })
```

The initialization form must call `initializeApp`, then set state to authenticated using the returned member.
The login form must call `listLoginMembers` on mount, submit `memberId` and `pin` to `login`, then set state to authenticated using the returned member.
The authenticated shell must expose a logout button that calls `logout`, then moves the state back to `needs-login`.

- [ ] **Step 4: Build frontend**

Run:

```powershell
pnpm --dir apps/web build
```

Expected: PASS.

- [ ] **Step 5: Commit frontend bootstrap**

```powershell
git add apps/web/src
git commit -m "feat: add frontend auth bootstrap"
```

---

## Task 7: Final Verification

**Files:**
- Modify if needed: files created in previous tasks.

- [ ] **Step 1: Start database**

Run:

```powershell
.\scripts\db-up.ps1
```

Expected: `commune-db` is running.

- [ ] **Step 2: Run migrations**

Run:

```powershell
.\scripts\migrate.ps1
```

Expected: exits 0.

- [ ] **Step 3: Run full build**

Run:

```powershell
.\scripts\build.ps1
```

Expected: frontend build passes, Go tests pass, and `dist\commune-server.exe` is created.

- [ ] **Step 4: Run API manually**

Run:

```powershell
Set-Location apps\api
go run .\cmd\server
```

Expected: API listens on `:8090`.

- [ ] **Step 5: Test bootstrap endpoint**

In another terminal:

```powershell
Invoke-WebRequest http://localhost:8090/api/bootstrap
```

Expected: JSON response with `initialized`.

- [ ] **Step 6: Test initialization endpoint manually**

In another terminal:

```powershell
Invoke-WebRequest http://localhost:8090/api/init -Method POST -ContentType "application/json" -Body '{"householdName":"Han Home","adminName":"Han","pin":"123456"}'
```

Expected: JSON response includes member name `Han` and role `admin`.

- [ ] **Step 7: Test login members endpoint manually**

Run:

```powershell
Invoke-WebRequest http://localhost:8090/api/login-members
```

Expected: JSON response includes member `Han`.

- [ ] **Step 8: Test login endpoint manually**

Use the member ID from the previous response:

```powershell
Invoke-WebRequest http://localhost:8090/api/login -Method POST -ContentType "application/json" -Body '{"memberId":"<member-id-from-login-members>","pin":"123456"}'
```

Expected: JSON response includes member role `admin`.

- [ ] **Step 9: Push branch**

```powershell
git push -u origin feature/commune-auth-foundation
```

Expected: branch is pushed.

---

## Self-Review

Spec coverage:

- Initialization status is covered by `GET /api/bootstrap`.
- First admin creation is covered by `POST /api/init`.
- PIN hashing is covered by `internal/auth/pin.go`.
- Session cookie and server-side session table are covered by `sessions` schema and cookie helpers.
- Frontend initialization flow is covered by Task 6.
- Login member selector is covered by `GET /api/login-members` and the login form.
- PIN login is covered by `POST /api/login`.
- Session restore is covered by `GET /api/bootstrap` and `GET /api/session`.
- Logout invalidation is covered by `POST /api/logout`.

Deferred by design:

- Admin member management.
- Ledger features.

Follow-up plan:

- Write `2026-05-22-commune-ledger-core.md` to implement default categories, member management, and transaction CRUD after the auth foundation is merged.
