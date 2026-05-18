# Tenant Filter Leaks — Audit & Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Audit every GORM call site across Atlas services against the F1–F10 tenant-filter threat model, close every classified leak, harden the tenant Create callback to inject `tenant_id` instead of merely warning, and add two-tenant regression tests so future regressions fail loudly.

**Architecture:** Lock in design.md's Option A — the `tenantQueryCallback` / `tenantCreateCallback` callbacks registered by `libs/atlas-database/connection.go:123` are the invariant. This plan (a) hardens the Create callback so it injects `tenant_id` when missing, (b) produces `audit.md` classifying every call site as PASS-CB / PASS-EXPLICIT / PASS-CROSS-TENANT / LEAK-F<n> / UNCLEAR, (c) applies per-F-class fix templates to every LEAK row, and (d) adds two-tenant sqlite-backed provider tests for each fix plus one read + one write per tenant-scoped service. No defense-in-depth duplicate WHERE clauses; no testcontainers.

**Tech Stack:** Go 1.25, GORM, in-memory SQLite (`gorm.io/driver/sqlite`), `libs/atlas-database` (callbacks, transaction, provider helpers), `libs/atlas-tenant` (context plumbing), `libs/atlas-model` (provider pattern), testify, logrus null logger.

**Artifact paths (this worktree):**
- Plan: `docs/tasks/task-041-tenant-filter-leaks/plan.md`
- Context: `docs/tasks/task-041-tenant-filter-leaks/context.md`
- Audit output: `docs/tasks/task-041-tenant-filter-leaks/audit.md`
- All code paths below are relative to the worktree root `<WORKTREE>/`.

---

## File Structure

**Created files:**
- `libs/atlas-database/testdb.go` — shared in-memory tenant DB helper (`NewInMemoryTenantDB`, `TenantContext`).
- `libs/atlas-database/testdb_test.go` — smoke test for the helper itself.
- `docs/tasks/task-041-tenant-filter-leaks/audit.md` — classified call-site inventory.
- One `provider_test.go` (or `administrator_test.go`) next to every fixed call site and every tenant-scoped provider audited (~30 services × ~2 tests).

**Modified files:**
- `libs/atlas-database/tenant_scope.go:80-120` — `tenantCreateCallback` rewritten to inject `tenant_id` from context onto zero-valued struct/slice rows.
- `libs/atlas-database/tenant_scope_test.go` — add F6 regression tests covering single-row inject, explicit override preserved, batched slice with mixed rows.
- Any service file flagged LEAK-F<n> by audit.md — fix applied per the per-class template in Task 6.

**Out of scope (do not touch):**
- `libs/atlas-database/connection.go` (callback wiring already correct).
- `EntityProvider` / `database.Query` / `database.SliceQuery` signatures.
- Removing redundant `TenantId: t.Id()` assignments at Create call sites (mechanical follow-up; explicitly deferred by design §5).
- `services/atlas-ui`, `services/atlas-assets`, `services/atlas-data` (read-only WZ), `services/atlas-tenants`, `services/atlas-wz-extractor`, `services/atlas-pr-bootstrap`, `services/atlas-runtime-orchestrator` — no Go GORM tenanted use.

---

## Task 1: Shared In-Memory Tenant DB Helper

Lift the sqlite setup pattern from `libs/atlas-database/tenant_scope_test.go:38-50` into a reusable helper so every service test can stand up a tenanted DB in 2 lines instead of 20. Three+ services will use it; design §6 #2 authorizes the lift.

**Files:**
- Create: `libs/atlas-database/testdb.go`
- Create: `libs/atlas-database/testdb_test.go`

- [ ] **Step 1: Write the failing test**

```go
// libs/atlas-database/testdb_test.go
package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type helperEntity struct {
	ID       uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId uuid.UUID `gorm:"not null"`
	Name     string    `gorm:"not null"`
}

func (helperEntity) TableName() string { return "helper_entities" }

func helperMigration(db *gorm.DB) error { return db.AutoMigrate(&helperEntity{}) }

func TestNewInMemoryTenantDB_RegistersCallbacksAndMigrates(t *testing.T) {
	db := NewInMemoryTenantDB(t, helperMigration)
	tid := uuid.New()
	require.NoError(t, db.Create(&helperEntity{TenantId: tid, Name: "x"}).Error)

	other := uuid.New()
	require.NoError(t, db.Create(&helperEntity{TenantId: other, Name: "y"}).Error)

	var rows []helperEntity
	require.NoError(t, db.WithContext(TenantContext(tid)).Find(&rows).Error)
	assert.Len(t, rows, 1)
	assert.Equal(t, "x", rows[0].Name)
}

func TestTenantContext_CarriesTenant(t *testing.T) {
	tid := uuid.New()
	ctx := TenantContext(tid)
	require.NotNil(t, ctx)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-database && go test -run 'TestNewInMemoryTenantDB|TestTenantContext' -v ./...
```

Expected: FAIL — `undefined: NewInMemoryTenantDB`, `undefined: TenantContext`.

- [ ] **Step 3: Write minimal implementation**

```go
// libs/atlas-database/testdb.go
package database

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewInMemoryTenantDB returns a fresh sqlite-in-memory *gorm.DB with the tenant
// callbacks registered and every supplied Migrator applied. Use this in provider
// tests instead of hand-rolling sqlite setup. A discard logger is attached so
// callback warnings do not spam test output.
func NewInMemoryTenantDB(t *testing.T, migrations ...Migrator) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	registerTenantCallbacks(l, db)
	for _, m := range migrations {
		require.NoError(t, m(db))
	}
	return db
}

// TenantContext returns a context carrying a tenant with the supplied id (GMS / v83 / region 1).
// Use this in tests to scope a query to a specific tenant.
func TenantContext(id uuid.UUID) context.Context {
	t, _ := tenant.Create(id, "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-database && go test -run 'TestNewInMemoryTenantDB|TestTenantContext' -v ./...
```

Expected: PASS.

- [ ] **Step 5: Run the rest of the package's tests to confirm no regression**

```bash
cd libs/atlas-database && go test -race ./...
```

Expected: PASS (including the existing `TestQuery*`, `TestUpdate*`, `TestDelete*`, `TestDoubleWhereIsHarmless`, `TestCreateDoesNotInjectWhere`).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-database/testdb.go libs/atlas-database/testdb_test.go
git commit -m "feat(atlas-database): NewInMemoryTenantDB test helper for tenant-aware sqlite

Shared helper for service provider tests so each service does not re-roll the
sqlite + RegisterTenantCallbacks + AutoMigrate boilerplate. Used by the
two-tenant regression tests added under task-041."
```

---

## Task 2: Harden `tenantCreateCallback` to Inject `tenant_id` (F6)

Replace the warn-only behavior at `libs/atlas-database/tenant_scope.go:100-119` with a call to `field.Set` so that a `Create` issued under a tenanted context produces a row whose `tenant_id` is the context tenant, even when the caller forgot to set it. Existing redundant assignments at call sites become belt-and-suspenders; do **not** remove them (out of scope per design §7).

**Files:**
- Modify: `libs/atlas-database/tenant_scope.go:80-120`
- Modify: `libs/atlas-database/tenant_scope_test.go` (add F6 regression tests)

- [ ] **Step 1: Add the three failing regression tests**

Append to `libs/atlas-database/tenant_scope_test.go`:

```go
func TestCreateInjectsTenantIdWhenMissing(t *testing.T) {
	db, _ := setupTestDB(t)
	tid := uuid.New()

	e := tenantEntity{Name: "no-tid"} // TenantId left zero
	require.NoError(t, db.WithContext(tenantContext(tid)).Create(&e).Error)

	var result tenantEntity
	require.NoError(t, db.Unscoped().Where("name = ?", "no-tid").First(&result).Error)
	assert.Equal(t, tid, result.TenantId, "callback must inject context tenant_id when struct value is zero")
	assert.Equal(t, tid, e.TenantId, "callback must mutate the caller's struct so subsequent reads see the injected value")
}

func TestCreateDoesNotOverrideExplicitTenantId(t *testing.T) {
	db, _ := setupTestDB(t)
	ctxTid := uuid.New()
	structTid := uuid.New()

	e := tenantEntity{TenantId: structTid, Name: "explicit"}
	require.NoError(t, db.WithContext(tenantContext(ctxTid)).Create(&e).Error)

	var result tenantEntity
	require.NoError(t, db.Unscoped().Where("name = ?", "explicit").First(&result).Error)
	assert.Equal(t, structTid, result.TenantId, "callback must not overwrite an explicitly-set non-zero tenant_id")
}

func TestCreateBatchInjectsTenantIdForZeroEntries(t *testing.T) {
	db, _ := setupTestDB(t)
	tid := uuid.New()
	explicitTid := uuid.New()

	rows := []tenantEntity{
		{Name: "row-a"},                            // zero -> inject tid
		{TenantId: explicitTid, Name: "row-b"},     // non-zero -> preserved
		{Name: "row-c"},                            // zero -> inject tid
	}
	require.NoError(t, db.WithContext(tenantContext(tid)).Create(&rows).Error)

	var all []tenantEntity
	require.NoError(t, db.Unscoped().Order("name").Find(&all).Error)
	require.Len(t, all, 3)

	got := map[string]uuid.UUID{}
	for _, r := range all {
		got[r.Name] = r.TenantId
	}
	assert.Equal(t, tid, got["row-a"])
	assert.Equal(t, explicitTid, got["row-b"])
	assert.Equal(t, tid, got["row-c"])
}
```

- [ ] **Step 2: Run new tests to verify they fail**

```bash
cd libs/atlas-database && go test -run 'TestCreateInjectsTenantIdWhenMissing|TestCreateDoesNotOverrideExplicitTenantId|TestCreateBatchInjectsTenantIdForZeroEntries' -v ./...
```

Expected: FAIL — the persisted row has `tenant_id = 00000000-0000-0000-0000-000000000000` because the callback only warns.

- [ ] **Step 3: Rewrite `tenantCreateCallback`**

Replace lines 80–120 of `libs/atlas-database/tenant_scope.go` with:

```go
func tenantCreateCallback(l logrus.FieldLogger) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Error != nil {
			return
		}

		if !hasTenantColumn(db) {
			return
		}

		ctx := db.Statement.Context
		if shouldSkipTenantFilter(ctx) {
			return
		}

		t, err := tenant.FromContext(ctx)()
		if err != nil {
			return
		}

		field := db.Statement.Schema.FieldsByDBName["tenant_id"]
		if !db.Statement.ReflectValue.IsValid() {
			return
		}

		rv := db.Statement.ReflectValue
		switch rv.Kind() {
		case reflect.Struct:
			injectTenantIdIfZero(ctx, l, db.Statement.Schema.Table, field, rv, t.Id())
		case reflect.Slice, reflect.Array:
			for i := 0; i < rv.Len(); i++ {
				injectTenantIdIfZero(ctx, l, db.Statement.Schema.Table, field, rv.Index(i), t.Id())
			}
		}
	}
}

// injectTenantIdIfZero sets tenant_id from context onto a single reflected row
// when its existing value is the zero UUID. Non-zero values are preserved. A
// callback-set failure is logged at warn level and the row is left untouched —
// the query will then proceed with whatever zero-value the caller supplied,
// matching pre-task-041 behavior.
func injectTenantIdIfZero(ctx context.Context, l logrus.FieldLogger, table string, field *schema.Field, rv reflect.Value, tenantId uuid.UUID) {
	_, isZero := field.ValueOf(ctx, rv)
	if !isZero {
		return
	}
	if err := field.Set(ctx, rv, tenantId); err != nil {
		l.WithError(err).Warnf("tenant:create: failed to inject tenant_id for %s; row will retain its zero value.", table)
	}
}
```

Then update the imports at the top of `tenant_scope.go`:

```go
import (
	"context"
	"reflect"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)
```

- [ ] **Step 4: Run the F6 tests to verify they pass**

```bash
cd libs/atlas-database && go test -run 'TestCreateInjectsTenantIdWhenMissing|TestCreateDoesNotOverrideExplicitTenantId|TestCreateBatchInjectsTenantIdForZeroEntries' -v ./...
```

Expected: PASS.

- [ ] **Step 5: Run the full package suite to confirm no regression**

```bash
cd libs/atlas-database && go test -race ./...
```

Expected: PASS on every existing case (`TestQuery*`, `TestUpdate*`, `TestDelete*`, `TestFirstWithTenantContext_*`, `TestDoubleWhereIsHarmless`, `TestCreateDoesNotInjectWhere`, and the three new F6 cases).

- [ ] **Step 6: Run `go vet` on the lib**

```bash
cd libs/atlas-database && go vet ./...
```

Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-database/tenant_scope.go libs/atlas-database/tenant_scope_test.go
git commit -m "fix(atlas-database): inject tenant_id on Create when struct value is zero (F6)

The tenantCreateCallback previously only warned when a Create-bound struct
had a zero TenantId; the row was then written with the zero UUID and became
invisible to any real tenant query. With the query callback already injecting
tenant_id on reads/updates/deletes, the Create path is now the only place a
caller can produce orphaned rows under a tenanted context.

Behavior: when (a) the entity has a tenant_id column, (b) the context carries
a tenant, (c) the skip flag is not set, and (d) the struct value is the zero
UUID, the callback sets tenant_id to the context tenant before the INSERT.
Explicitly non-zero values are preserved (caller wins). Batched Create over a
slice handles each row independently.

Tests cover single-row injection, explicit-override preservation, and mixed
batched slices."
```

---

## Task 3: Audit Enumeration — Produce `audit.md`

Run the static enumeration pipeline from design §4 and classify every GORM call site that touches a tenant-scoped entity. This is the work item that scopes the rest of the plan: every LEAK-F<n> row in audit.md becomes a fix sub-task in Task 6; every PASS-* row informs which services need smoke tests in Task 8.

**Files:**
- Create: `docs/tasks/task-041-tenant-filter-leaks/audit.md`

- [ ] **Step 1: Enumerate raw call sites**

```bash
cd <WORKTREE>
rg --type go -n \
  -e '\bp?\.?db\.(Where|Find|First|Take|Scan|Create|Save|Updates|UpdateColumn|UpdateColumns|Delete|Exec|Raw|Preload|Joins)\(' \
  -e '\btx\.(Where|Find|First|Take|Scan|Create|Save|Updates|UpdateColumn|UpdateColumns|Delete|Exec|Raw|Preload|Joins)\(' \
  -e 'database\.Query\(' -e 'database\.SliceQuery\(' \
  -e 'WithoutTenantFilter' \
  services/ \
  | rg -v '_test\.go' \
  > /tmp/task-041-callsites.txt
wc -l /tmp/task-041-callsites.txt
```

Expected: a non-empty file listing one call per line. Keep `/tmp/task-041-callsites.txt` as a working scratch.

- [ ] **Step 2: Inventory entities that declare `TenantId`**

```bash
rg --type go -l -e '\bTenantId\b\s+uuid\.UUID' services/ \
  | sort -u > /tmp/task-041-tenanted-entities.txt
wc -l /tmp/task-041-tenanted-entities.txt
```

Then list entity files that DO NOT declare `TenantId` for inspection:

```bash
find services -path "*/atlas-*/*entity.go" \
  | xargs rg -L '\bTenantId\b' \
  | sort -u > /tmp/task-041-tenantless-entities.txt
wc -l /tmp/task-041-tenantless-entities.txt
```

Use these two lists when classifying each row.

- [ ] **Step 3: Background-context audit**

Find places where a goroutine builds its own context (rather than inheriting a tenanted one) so each one can be classified F2 if appropriate:

```bash
rg --type go -n -B1 -A4 \
  -e 'context\.Background\(\)' -e 'context\.TODO\(\)' \
  services/ | rg -v '_test\.go' > /tmp/task-041-bg-ctx.txt
```

Also find tasks that already use `tenant.WithContext` correctly:

```bash
rg --type go -n 'tenant\.WithContext' services/ > /tmp/task-041-good-bg-ctx.txt
```

- [ ] **Step 4: Raw SQL audit**

```bash
rg --type go -n -e '\.Raw\(' -e '\.Exec\(' services/ \
  | rg -v '_test\.go' > /tmp/task-041-raw-sql.txt
```

Every entry here is candidate F4 unless it is a global/registry table.

- [ ] **Step 5: Preload audit**

```bash
rg --type go -n '\bPreload\("' services/ | rg -v '_test\.go' > /tmp/task-041-preloads.txt
```

For every preload target table, check whether that target's entity declares `TenantId`. If not → F3/F8 candidate.

- [ ] **Step 6: Existing `WithoutTenantFilter` sites — F10 audit**

```bash
rg --type go -n -B2 -A4 'WithoutTenantFilter' services/ \
  | rg -v '_test\.go' > /tmp/task-041-bypass.txt
```

For each, verify (a) there is a comment justifying the bypass, (b) the bypass scope ends before any tenant-aware downstream call. Flag missing comments as fix-during-this-task in audit.md.

- [ ] **Step 7: Classify every call site and write `audit.md`**

Create `docs/tasks/task-041-tenant-filter-leaks/audit.md` with this exact structure:

````markdown
# Tenant Filter Leak Audit — task-041

Date: <YYYY-MM-DD>
Methodology: design.md §4 (static enumeration + entity inventory + WithContext + raw SQL + preload + WithoutTenantFilter passes).
Threat model: F1–F10 from design.md §2.

## Summary

| Class | Count |
|---|---|
| PASS-CB | <n> |
| PASS-EXPLICIT | <n> |
| PASS-CROSS-TENANT | <n> |
| LEAK-F1 | <n> |
| LEAK-F2 | <n> |
| LEAK-F3 | <n> |
| LEAK-F4 | <n> |
| LEAK-F5 | <n> |
| LEAK-F6 | <n> (resolved by Task 2; sites listed for historic record) |
| LEAK-F7 | <n> |
| LEAK-F8 | <n> |
| LEAK-F9 | <n> |
| LEAK-F10 | <n> |
| UNCLEAR | <n> |

## Call sites

| Service | File:Line | Function | Op | Class | Fix | Notes |
|---|---|---|---|---|---|---|
| atlas-character | services/atlas-character/atlas.com/character/character/provider.go:11 | getById | R | PASS-CB | none | callback covers; entity has tenant_id; processor uses WithContext(p.ctx) |
| ... | ... | ... | ... | ... | ... | ... |

## Tenant-scoped services (drives Task 8 test coverage)

- atlas-account
- atlas-asset-expiration
- ...
(every service that has at least one row classified PASS-CB / PASS-EXPLICIT / LEAK-* in the table above)

## Intentional cross-tenant sites (PASS-CROSS-TENANT)

| Service | File:Line | Why cross-tenant | Bypass scope verified? |
|---|---|---|---|
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/task.go:NN | startup recovery enumerates shops across tenants | yes — bypass ends before per-tenant ctx is built at line NN |
| atlas-data | services/atlas-data/... | read-only WZ data, no tenant axis | yes — entity has no tenant_id column |
| atlas-saga-orchestrator | services/atlas-saga-orchestrator/.../saga/store.go:NN | saga recovery across all tenants on startup | yes — per-saga handling re-derives ctx |
````

Important rules while filling the table:
- **Every row must include file:line evidence.** If a classifier cannot point to a line, the classification is UNCLEAR and a fix is required.
- F6 sites (Create without TenantId): list them as `LEAK-F6 (resolved)` with a note that the fix is the callback hardening in Task 2. Do not generate per-site fix tasks for these.
- A LEAK-F10 row exists for any `WithoutTenantFilter` call site that lacks a justification comment; the fix is "add comment".

- [ ] **Step 8: Verify audit.md is internally consistent**

```bash
# Every LEAK row must have a non-empty Fix column.
rg '\| LEAK-' docs/tasks/task-041-tenant-filter-leaks/audit.md | rg '\|\s*\|' && echo "GAP: LEAK row with empty fix" || echo "OK"

# Every UNCLEAR row must have a Fix column starting with 'resolve:' explaining what to look at.
rg '\| UNCLEAR' docs/tasks/task-041-tenant-filter-leaks/audit.md | rg -v 'resolve:' && echo "GAP: UNCLEAR row missing resolve plan" || echo "OK"

# The Summary counts add up to the call-site row count.
awk '/^\| atlas-/ {n++} END {print n " call-site rows"}' docs/tasks/task-041-tenant-filter-leaks/audit.md
```

Expected: all three checks print `OK` and the row count matches the Summary total.

- [ ] **Step 9: Commit**

```bash
git add docs/tasks/task-041-tenant-filter-leaks/audit.md
git commit -m "docs(task-041): audit of every tenant-scoped GORM call site across services

Classifies each as PASS-CB / PASS-EXPLICIT / PASS-CROSS-TENANT / LEAK-F<n>
per design.md §2. Drives Task 6 (per-class fixes) and Task 8 (per-service
test coverage)."
```

---

## Task 4: Regression Tests for atlas-guilds Providers

PRD §10 names these explicitly: `getAll`, `getById`, `getForName`. Each gets a two-tenant overlap fixture and an isolation assertion. Use `database.NewInMemoryTenantDB` from Task 1 plus the guild package's own `Migration` chain.

**Files:**
- Create: `services/atlas-guilds/atlas.com/guilds/guild/provider_test.go`

- [ ] **Step 1: Inspect the guild Migration chain so the helper boots a complete schema**

```bash
rg -n 'func Migration' services/atlas-guilds/atlas.com/guilds/guild/
```

Expected: at least `guild/entity.go:Migration`, `guild/member/entity.go:Migration`, `guild/title/entity.go:Migration`. The test wires all three.

- [ ] **Step 2: Write the failing test file**

```go
// services/atlas-guilds/atlas.com/guilds/guild/provider_test.go
package guild

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/services/atlas-guilds/atlas.com/guilds/guild/member"
	"github.com/Chronicle20/atlas/services/atlas-guilds/atlas.com/guilds/guild/title"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGuildsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration, member.Migration, title.Migration)
	tidA, tidB := uuid.New(), uuid.New()
	// Same guild id in both tenants to prove isolation by tenant, not by id.
	require.NoError(t, db.Create(&Entity{Id: 1, TenantId: tidA, WorldId: 0, Name: "Phoenix", LeaderId: 100}).Error)
	require.NoError(t, db.Create(&Entity{Id: 1, TenantId: tidB, WorldId: 0, Name: "Phoenix", LeaderId: 200}).Error)
	return db, tidA, tidB
}

func TestGuildProvider_GetById_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newGuildsDB(t)

	gotA, err := getById(1)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, uint32(100), gotA.LeaderId, "tenant A's row")

	gotB, err := getById(1)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, uint32(200), gotB.LeaderId, "tenant B's row")
}

func TestGuildProvider_GetForName_FiltersByTenant(t *testing.T) {
	db, tidA, _ := newGuildsDB(t)
	results, err := getForName(world.Id(0), "Phoenix")(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, results, 1, "even though both tenants have a 'Phoenix' in world 0, only tenant A's row is returned")
	assert.Equal(t, uint32(100), results[0].LeaderId)
}

func TestGuildProvider_GetAll_FiltersByTenant(t *testing.T) {
	db, _, tidB := newGuildsDB(t)
	all, err := getAll()(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, all, 1, "GetAll must not leak across tenants")
	assert.Equal(t, uint32(200), all[0].LeaderId)
}
```

The import `"gorm.io/gorm"` must be added to the import block too — keeping it together with the existing imports.

- [ ] **Step 3: Run tests — expect them to pass already**

```bash
cd services/atlas-guilds && go test -race -run 'TestGuildProvider_' ./atlas.com/guilds/guild/... -v
```

Expected: PASS on all three. (They guard against future regression — if a future change drops the callback or removes `WithContext`, the tests fail.)

- [ ] **Step 4: Drop the callback to confirm the tests would catch a leak**

Temporarily edit `libs/atlas-database/testdb.go` to skip `registerTenantCallbacks(l, db)` (revert immediately after). Re-run:

```bash
cd services/atlas-guilds && go test -race -run 'TestGuildProvider_' ./atlas.com/guilds/guild/... -v
```

Expected: FAIL on at least one assertion (two rows where one was expected). **Revert the edit before continuing.** Confirm tests pass again.

- [ ] **Step 5: Run vet + race on the service**

```bash
cd services/atlas-guilds && go vet ./... && go test -race ./...
```

Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-guilds/atlas.com/guilds/guild/provider_test.go
git commit -m "test(atlas-guilds): two-tenant regression tests for guild provider

Covers getById, getForName, getAll (PRD §10). Each test seeds two tenants
with overlapping guild ids and overlapping names, queries through a tenanted
context, and asserts only the matching tenant's row is returned. Tests fail
if the tenant callback is dropped or if WithContext is removed from the
provider call chain."
```

---

## Task 5: Regression Tests for atlas-character Providers

PRD §10 names these: `getById`, `getForAccount`, `getForAccountInWorld`, `getForName`, `getAll`. Same shape as Task 4.

**Files:**
- Create: `services/atlas-character/atlas.com/character/character/provider_test.go`

- [ ] **Step 1: Inspect the character Migration chain**

```bash
rg -n 'func Migration' services/atlas-character/atlas.com/character/character/
```

Expected: `entity.go:Migration` (at minimum). If the character package has child migrations (e.g., quests, skills), include those that the provider's queries actually exercise. For these five providers, the `character.Migration` alone is sufficient (no preloads).

- [ ] **Step 2: Write the failing test file**

```go
// services/atlas-character/atlas.com/character/character/provider_test.go
package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newCharsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	// Same id, same accountId, same name across tenants — prove isolation by tenant only.
	require.NoError(t, db.Create(&entity{ID: 1, TenantId: tidA, AccountId: 7, World: 0, Name: "Hero", Level: 1, JobId: 0}).Error)
	require.NoError(t, db.Create(&entity{ID: 1, TenantId: tidB, AccountId: 7, World: 0, Name: "Hero", Level: 200, JobId: 0}).Error)
	return db, tidA, tidB
}

func TestCharacterProvider_GetById_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newCharsDB(t)

	gotA, err := getById(1)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, byte(1), gotA.Level)

	gotB, err := getById(1)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, byte(200), gotB.Level)
}

func TestCharacterProvider_GetForAccount_FiltersByTenant(t *testing.T) {
	db, tidA, _ := newCharsDB(t)
	rows, err := getForAccount(7)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rows, 1, "account 7 has overlapping characters across tenants — only tenant A's should return")
	assert.Equal(t, byte(1), rows[0].Level)
}

func TestCharacterProvider_GetForAccountInWorld_FiltersByTenant(t *testing.T) {
	db, _, tidB := newCharsDB(t)
	rows, err := getForAccountInWorld(7, world.Id(0))(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, byte(200), rows[0].Level)
}

func TestCharacterProvider_GetForName_FiltersByTenant(t *testing.T) {
	db, tidA, _ := newCharsDB(t)
	rows, err := getForName("Hero")(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, byte(1), rows[0].Level)
}

func TestCharacterProvider_GetAll_FiltersByTenant(t *testing.T) {
	db, _, tidB := newCharsDB(t)
	rows, err := getAll()(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, rows, 1, "GetAll must not leak across tenants")
	assert.Equal(t, byte(200), rows[0].Level)
}
```

- [ ] **Step 3: Run tests — expect pass**

```bash
cd services/atlas-character && go test -race -run 'TestCharacterProvider_' ./atlas.com/character/character/... -v
```

Expected: PASS on all five.

- [ ] **Step 4: Sanity-check the negative case**

As in Task 4 Step 4: temporarily disable `registerTenantCallbacks` in the helper, re-run, confirm failures, revert.

- [ ] **Step 5: Run vet + race on the service**

```bash
cd services/atlas-character && go vet ./... && go test -race ./...
```

Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/provider_test.go
git commit -m "test(atlas-character): two-tenant regression tests for character provider

Covers getById, getForAccount, getForAccountInWorld, getForName, getAll
(PRD §10). Each test seeds tenant A and tenant B with overlapping id,
accountId, and name; queries with a tenanted context; asserts only the
matching tenant's row is returned."
```

---

## Task 6: Apply F-Class Fix Templates to Every LEAK Row in `audit.md`

For each LEAK-F<n> row produced by Task 3 (excluding the resolved-by-Task-2 F6 rows), apply the matching template below. Each fix lands in its own commit so the audit table maps one-to-one to commits. Reviewers can `git log -- docs/tasks/task-041-tenant-filter-leaks/audit.md` to see the inventory and `git log --grep 'fix(task-041' --oneline` to see the per-class fixes.

For each LEAK row, the executor does this loop:

```
for row in LEAK rows:
    apply fix template for row.Class to row.File:Line
    run `go test -race ./...` in the changed service
    commit with the exact message in the template
```

### Template F1 — Missing `WithContext`

**Symptom (audit example):** `services/atlas-foo/.../bar/provider.go:NN` issues `p.db.Find(...)` without `.WithContext(p.ctx)`, so the callback's `tenant.FromContext` errors out and the query runs un-scoped.

- [ ] **Step F1.1: Edit the call to thread context**

Replace:

```go
return p.db.Where("id = ?", id).First(&e).Error
```

with:

```go
return p.db.WithContext(p.ctx).Where("id = ?", id).First(&e).Error
```

If the processor does not already have a `ctx` field, add one to its struct and thread it through `New<Type>Processor(ctx context.Context, ...)`. Match the existing pattern in `services/atlas-guilds/.../processor.go` (which already stores `ctx`).

- [ ] **Step F1.2: Inside `database.ExecuteTransaction`, verify `tx` inherits**

If the call lives inside `ExecuteTransaction(p.db, func(tx *gorm.DB) error { ... })`, change the outer call to `ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error { ... })`. The inner `tx` inherits the parent statement context. No change to inner `tx.Where(...)` calls needed.

- [ ] **Step F1.3: Run the service's tests**

```bash
cd services/atlas-<svc> && go test -race ./...
```

Expected: PASS.

- [ ] **Step F1.4: Commit**

```bash
git add services/atlas-<svc>/...
git commit -m "fix(task-041, atlas-<svc>): thread WithContext on <function> (F1)"
```

### Template F2 — Background Context Without Tenant

**Symptom:** A goroutine in a background task builds its own `ctx := context.Background()` and issues queries against tenant-scoped tables. The callback skips because no tenant is in the context.

Reference good pattern: `services/atlas-guilds/atlas.com/guilds/guild/task.go:43` — `tctx := tenant.WithContext(sctx, g.Tenant())`.

- [ ] **Step F2.1: Enumerate tenants from the registry, run per-tenant**

Inside the task, replace:

```go
ctx := context.Background()
// queries against tenant-scoped tables...
```

with:

```go
tenants, err := tenantRegistry.AllTenants(ctx) // however the service obtains them; existing tasks have a pattern
if err != nil {
    p.l.WithError(err).Errorf("Failed to enumerate tenants for <task name>; skipping cycle.")
    return
}
for _, tnt := range tenants {
    tctx := tenant.WithContext(ctx, tnt)
    // queries against tenant-scoped tables, using tctx
}
```

If the task is intentionally cross-tenant (e.g., aggregated metrics), reclassify the audit row as PASS-CROSS-TENANT and wrap the queries in `database.WithoutTenantFilter(ctx)` instead. **The reclassification must be reflected in audit.md.**

- [ ] **Step F2.2: Run the service's tests**

```bash
cd services/atlas-<svc> && go test -race ./...
```

- [ ] **Step F2.3: Commit**

```bash
git commit -m "fix(task-041, atlas-<svc>): scope <task> per tenant via tenant.WithContext (F2)"
```

### Template F3/F8 — Missing `tenant_id` Column on Child Table

**Symptom:** A child table (e.g., `quest_progress`, `inventory_slot`) does not declare `TenantId` on its entity but is joined/preloaded from a parent that does. Or its administrator inserts rows without a tenant axis.

Per design §5: add a `tenant_id` column + index + idempotent backfill in a single `Migration` function next to the entity.

- [ ] **Step F3.1: Add `TenantId` to the entity**

```go
// services/atlas-<svc>/.../child/entity.go
type Entity struct {
    TenantId uuid.UUID `gorm:"not null;index"`
    // ...existing fields
}
```

- [ ] **Step F3.2: Update the `Migration` function to backfill from the parent**

```go
func Migration(db *gorm.DB) error {
    if err := db.AutoMigrate(&Entity{}); err != nil {
        return err
    }
    // Idempotent backfill — once tenant_id is non-zero everywhere, this is a no-op.
    return db.Exec(`
        UPDATE child_table c
        SET tenant_id = p.tenant_id
        FROM parent_table p
        WHERE c.<fk> = p.id AND (c.tenant_id IS NULL OR c.tenant_id = '00000000-0000-0000-0000-000000000000')
    `).Error
}
```

Replace `child_table`, `parent_table`, and `<fk>` with the actual names from the audit row.

- [ ] **Step F3.3: Update the `Make`/`Build` model code so reads round-trip the field**

If the package's `Model` struct does not carry `tenantId`, leave it out of the public surface (no caller currently depends on it). The DB column is sufficient for the callback to scope.

- [ ] **Step F3.4: Run the service's tests + a fresh schema reload via the in-memory helper**

```bash
cd services/atlas-<svc> && go test -race ./...
```

- [ ] **Step F3.5: Commit**

```bash
git commit -m "fix(task-041, atlas-<svc>): add tenant_id to <child entity> with backfill (F3/F8)"
```

### Template F4 — Raw SQL Bypassing Callback

**Symptom:** A call uses `p.db.Raw(...)` or `p.db.Exec(...)`; the GORM Query/Row callbacks operate on parsed schema and do not run.

- [ ] **Step F4.1: Prefer rewriting to the GORM API**

If the raw SQL is a straightforward SELECT, replace with `db.WithContext(p.ctx).Where(...).Find(&dst)`. The callback then handles tenant scoping.

- [ ] **Step F4.2: If raw SQL must remain, pass tenant explicitly**

```go
t, err := tenant.FromContext(p.ctx)()
if err != nil {
    return err
}
return p.db.WithContext(p.ctx).Raw(`
    SELECT ... FROM foo
    WHERE bar = ? AND tenant_id = ?
`, barVal, t.Id()).Scan(&dst).Error
```

- [ ] **Step F4.3: Add a regression test**

The two-tenant overlap test from Task 4/5 applied to this specific raw SQL call site.

- [ ] **Step F4.4: Commit**

```bash
git commit -m "fix(task-041, atlas-<svc>): close raw SQL tenant leak in <function> (F4)"
```

### Template F5 — Cross-Table Join Where Joined Table Lacks `tenant_id`

**Symptom:** `db.Joins("LEFT JOIN child c ON c.<fk> = parent.id").Where(...)` — driving table has `tenant_id`, joined table does not, foreign key collides across tenants.

- [ ] **Step F5.1:** Apply F3/F8 template to the joined table (add `tenant_id` column + index + backfill), then ensure the join predicate references both: `... ON c.<fk> = parent.id AND c.tenant_id = parent.tenant_id`. The callback alone is not enough for joins; the join predicate has to pair both columns.

- [ ] **Step F5.2:** Regression test as in Task 4/5.

- [ ] **Step F5.3:** Commit `fix(task-041, atlas-<svc>): pair join predicate on tenant_id (F5)`.

### Template F7 — Struct-Where with Possibly-Conflicting `TenantId`

**Symptom:** `db.Where(&Entity{ID: x, TenantId: yyy}).First(...)` where `yyy` came from a different source than the context tenant. GORM struct-where ignores zero fields; the callback adds the context `tenant_id`. If `yyy` differs from the context tenant, the query yields zero rows silently.

- [ ] **Step F7.1: Drop the explicit `TenantId` from the struct-where**

```go
// Before:
db.WithContext(p.ctx).Where(&Entity{ID: x, TenantId: someUUID}).First(&e)
// After:
db.WithContext(p.ctx).Where(&Entity{ID: x}).First(&e)
```

The callback supplies `tenant_id` from context — the only authoritative source.

- [ ] **Step F7.2: Regression test + commit `fix(task-041, atlas-<svc>): rely on callback for tenant_id in struct-where (F7)`.**

### Template F9 — Test Setup Forgot the Callback

**Symptom:** A `*_test.go` opens a sqlite DB directly without `RegisterTenantCallbacks`. The test passes locally even though the prod code path would leak.

- [ ] **Step F9.1: Replace the test's DB setup with `database.NewInMemoryTenantDB`** from Task 1.
- [ ] **Step F9.2: Commit `fix(task-041, atlas-<svc>): use NewInMemoryTenantDB so tests cover the callback (F9)`.**

### Template F10 — Over-Broad `WithoutTenantFilter`

**Symptom:** A `WithoutTenantFilter` site lacks a justification comment, or its bypass scope extends past the cross-tenant operation into downstream tenant-aware calls.

- [ ] **Step F10.1: Add a comment immediately above the bypass call**

```go
// Cross-tenant: startup recovery enumerates shops across every tenant before
// per-tenant goroutines are spawned. Scope ends when each per-tenant ctx is
// built below (line NN).
ctx := database.WithoutTenantFilter(parentCtx)
```

- [ ] **Step F10.2: If the scope leaks downstream, narrow it**

Replace any pattern where the bypass context flows into a tenant-aware function with an explicit re-derivation at the boundary:

```go
// Cross-tenant load done. From here, every caller derives its own tenanted ctx.
parentCtx = ctx // discard the bypass before any tenant-aware downstream call
```

- [ ] **Step F10.3: Regression test asserting that downstream tenant-aware calls do not see the bypass.** Pattern: spy logger or assert query result shape.

- [ ] **Step F10.4: Commit `fix(task-041, atlas-<svc>): justify and narrow WithoutTenantFilter scope (F10)`.**

### After all LEAK rows are fixed

- [ ] **Step 6.Final: Update `audit.md` to mark each fixed row**

For every LEAK row, change the `Fix` cell from `pending: <template>` to `done: <commit-sha>`. Commit:

```bash
git add docs/tasks/task-041-tenant-filter-leaks/audit.md
git commit -m "docs(task-041): mark fixed LEAK rows with their commit shas"
```

---

## Task 7: Verify atlas-guilds Child-Table Preloads (F8 follow-up)

The audit will already classify `Preload("Members")` and `Preload("Titles")` against the child entities. Both children declare `TenantId` (verified during planning: `services/atlas-guilds/atlas.com/guilds/guild/member/entity.go:12` and `services/atlas-guilds/atlas.com/guilds/guild/title/entity.go:11`), so this should be a PASS-CB row in `audit.md`. This task is the proof-by-test that the preload is in fact tenant-scoped.

**Files:**
- Modify: `services/atlas-guilds/atlas.com/guilds/guild/provider_test.go` (add preload-isolation test)

- [ ] **Step 1: Add a two-tenant preload isolation test**

```go
func TestGuildProvider_GetById_PreloadsAreTenantScoped(t *testing.T) {
	db := database.NewInMemoryTenantDB(t, Migration, member.Migration, title.Migration)
	tidA, tidB := uuid.New(), uuid.New()

	// Same guild id 1 in both tenants
	require.NoError(t, db.Create(&Entity{Id: 1, TenantId: tidA, WorldId: 0, Name: "Phoenix", LeaderId: 100}).Error)
	require.NoError(t, db.Create(&Entity{Id: 1, TenantId: tidB, WorldId: 0, Name: "Phoenix", LeaderId: 200}).Error)

	// Members + titles with overlapping ids across tenants
	require.NoError(t, db.Create(&member.Entity{CharacterId: 11, TenantId: tidA, GuildId: 1, Name: "alice"}).Error)
	require.NoError(t, db.Create(&member.Entity{CharacterId: 11, TenantId: tidB, GuildId: 1, Name: "bob"}).Error)

	got, err := getById(1)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, got.Members, 1, "preload must not leak tenant B's members")
	assert.Equal(t, "alice", got.Members[0].Name)
}
```

- [ ] **Step 2: Run + verify**

```bash
cd services/atlas-guilds && go test -race -run 'TestGuildProvider_' ./atlas.com/guilds/guild/... -v
```

Expected: PASS. If the preload leaks, the test fails (`len(Members) == 2` with one from each tenant).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-guilds/atlas.com/guilds/guild/provider_test.go
git commit -m "test(atlas-guilds): preload tenant-isolation regression test (F8)

Proves Preload(\"Members\") on the Guild Entity issues a tenanted child
query because guild_members.tenant_id is declared and the callback fires
on the child select."
```

---

## Task 8: Per-Service Smoke Tests — One Read + One Write per Tenant-Scoped Service

PRD §10 strict: every service that touches a tenant-scoped entity gets one read and one write provider test. Use the **template** below and apply once per service listed in audit.md's "Tenant-scoped services" section.

**Template (apply per service):**

```go
// services/atlas-<svc>/.../<package>/provider_test.go
package <package>

import (
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration /*, child.Migration, ...*/)
	return db, uuid.New(), uuid.New()
}

func TestProvider_Read_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newDB(t)
	// Seed identical primary-key rows for both tenants
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tidA /*, ...*/}).Error)
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tidB /*, ...*/}).Error)

	// Pick the package's canonical read provider (getById / getAll / etc.)
	got, err := getById(1)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, got.TenantId)
}

func TestProvider_Write_ScopedToTenant(t *testing.T) {
	db, tidA, tidB := newDB(t)
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tidA, /*, ...*/}).Error)
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tidB, /*, ...*/}).Error)

	// Pick the package's canonical write provider (update / delete).
	// If the service exposes only Create, the F6 regression in libs/atlas-database
	// already covers Create injection — write a no-op assertion here instead:
	//   require.NoError(t, ...Create(...)) and assert tenant_id matches context.
	err := db.WithContext(database.TenantContext(tidA)).
		Model(&Entity{}).
		Where("id = ?", 1).
		Update("some_field", "tenantA-only").Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	// The row whose tenant matches the context was updated; the other was not.
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, "tenantA-only", r.SomeField)
		} else {
			assert.NotEqual(t, "tenantA-only", r.SomeField, "tenant B must be untouched")
		}
	}
}
```

**Service checklist (apply template once per service):**

The audit.md "Tenant-scoped services" section is authoritative. The following list comes from a preliminary entity scan and bounds the work; reconcile with audit.md before starting.

- [ ] atlas-account
- [ ] atlas-asset-expiration
- [ ] atlas-ban
- [ ] atlas-buddies
- [ ] atlas-buffs
- [ ] atlas-cashshop
- [ ] atlas-chairs
- [ ] atlas-chalkboards
- [ ] atlas-character-factory
- [ ] atlas-consumables
- [ ] atlas-drop-information
- [ ] atlas-drops
- [ ] atlas-effective-stats
- [ ] atlas-expressions
- [ ] atlas-fame
- [ ] atlas-families
- [ ] atlas-gachapons
- [ ] atlas-inventory
- [ ] atlas-invites
- [ ] atlas-keys
- [ ] atlas-marriages
- [ ] atlas-merchant
- [ ] atlas-messages
- [ ] atlas-messengers
- [ ] atlas-monster-book
- [ ] atlas-notes
- [ ] atlas-npc-shops
- [ ] atlas-parties
- [ ] atlas-party-quests
- [ ] atlas-pets
- [ ] atlas-quest
- [ ] atlas-saga-orchestrator
- [ ] atlas-skills
- [ ] atlas-storage

For each:

- [ ] **Step 8.X.1:** Identify the package containing the canonical tenant-scoped entity (usually one entity.go declares `TenantId`; one provider.go has the reads).
- [ ] **Step 8.X.2:** Copy the template into `provider_test.go` (or `administrator_test.go` if the package's writes live in an administrator), filling in (a) Migration chain, (b) entity fields needed for primary-key conflict, (c) the chosen read provider, (d) the chosen write provider or Create injection check.
- [ ] **Step 8.X.3:** Run `cd services/atlas-<svc> && go test -race ./...`. Expected: PASS.
- [ ] **Step 8.X.4:** Run `cd services/atlas-<svc> && go vet ./...`. Expected: clean.
- [ ] **Step 8.X.5:** Commit one per service:

```bash
git commit -m "test(atlas-<svc>): two-tenant smoke regression for tenant scoping

Adds the read + write tenant isolation tests required by task-041 §10."
```

Batch reasonable: feel free to bundle 3–5 services into one commit if their changes are mechanically identical; the commit body must list every service and every provider test added.

---

## Task 9: Verification — `go test`, `go vet`, `go build`, `docker build`

Per CLAUDE.md's mandatory checklist before claiming the branch is done. The verification spans every module the branch touches, including `libs/atlas-database`.

- [ ] **Step 1: Collect the changed-modules list**

```bash
git diff --name-only origin/main...HEAD \
  | rg '^(services/[^/]+|libs/[^/]+)/' -o \
  | sort -u > /tmp/task-041-changed-modules.txt
cat /tmp/task-041-changed-modules.txt
```

- [ ] **Step 2: `go test -race ./...` in every changed module**

For every line in `/tmp/task-041-changed-modules.txt`:

```bash
cd <WORKTREE>/<module> && go test -race ./...
```

Expected: PASS for every module. If any fail, fix and re-run before continuing.

- [ ] **Step 3: `go vet ./...` in every changed module**

```bash
cd <WORKTREE>/<module> && go vet ./...
```

Expected: clean.

- [ ] **Step 4: `go build ./...` in every changed service**

```bash
cd <WORKTREE>/services/atlas-<svc> && go build ./...
```

Expected: clean for every service.

- [ ] **Step 5: Identify services whose `go.mod` or `Dockerfile` was touched**

```bash
git diff --name-only origin/main...HEAD \
  | rg '^(services/[^/]+/(go\.mod|Dockerfile))$'
```

If the list is empty, skip Step 6. If not empty, every listed service must be Docker-built.

- [ ] **Step 6: `docker build` from the worktree root for each Dockerfile-touched service**

```bash
cd <WORKTREE>
docker build -f services/atlas-<svc>/Dockerfile .
```

Expected: build succeeds. This catches drift in the Dockerfile's hand-edited COPY lists for atlas-* libs (CLAUDE.md §Build & Verification). If the build fails complaining about missing files in the COPY stage, update the four locations in the Dockerfile per CLAUDE.md.

In the common case for this task (only `libs/atlas-database/tenant_scope.go`, `libs/atlas-database/testdb.go`, and service test files change), no `go.mod` or `Dockerfile` is touched and Step 6 is a no-op.

- [ ] **Step 7: Record the verification in a single commit if any post-fix tweaks were needed**

If Steps 2–6 forced any code changes, commit them with:

```bash
git commit -m "chore(task-041): post-verification fixes from build/test sweep"
```

If nothing needed fixing, do not create an empty commit.

---

## Task 10: Code Review — Dispatch the Reviewer Agents

Per CLAUDE.md "Code Review Before PR": before opening the PR, run `superpowers:requesting-code-review`. With only Go file changes (no atlas-ui), the dispatch fans out `plan-adherence-reviewer` + `backend-guidelines-reviewer` in parallel.

- [ ] **Step 1: Invoke the requesting-code-review skill**

```
Use the Skill tool: superpowers:requesting-code-review
```

It dispatches:
- `plan-adherence-reviewer` — verifies every task in this plan was implemented; writes findings to `docs/tasks/task-041-tenant-filter-leaks/audit.md` (the reviewer's adherence audit appends to the same file; reviewers segregate sections by header).
- `backend-guidelines-reviewer` — runs the DOM-* checklist over the changed Go packages and writes to the same audit file.

- [ ] **Step 2: Address the reviewer findings**

For every PASS-FAIL flagged item, either fix the code or push back in writing in the audit file with file:line justification. Re-run the failed reviewer subset only.

- [ ] **Step 3: Commit any review-driven fixes**

```bash
git commit -m "chore(task-041): address code review findings"
```

---

## Task 11: PR

- [ ] **Step 1: Confirm worktree state**

```bash
cd <WORKTREE>
git status
git log --oneline origin/main..HEAD
```

Expected: clean working tree, every commit follows the `task-041` convention.

- [ ] **Step 2: Open the PR**

```bash
gh pr create --title "fix(task-041): tenant filter leak audit + F6 callback hardening + regression tests" --body "$(cat <<'EOF'
## Summary

- Audits every GORM call site across Atlas services against the F1–F10 tenant-filter threat model (see \`docs/tasks/task-041-tenant-filter-leaks/audit.md\`).
- Hardens \`libs/atlas-database/tenant_scope.go\` so \`tenantCreateCallback\` injects \`tenant_id\` from context on Create when the struct value is zero (warn → inject).
- Adds two-tenant overlap regression tests for atlas-guilds (\`getAll\`, \`getById\`, \`getForName\`) and atlas-character (\`getById\`, \`getForAccount\`, \`getForAccountInWorld\`, \`getForName\`, \`getAll\`), plus one read + one write smoke per tenant-scoped service.
- Applies per-F-class fixes to every LEAK row in audit.md (one commit per row).

## Audit

See [audit.md](./docs/tasks/task-041-tenant-filter-leaks/audit.md) — every PASS / LEAK classification with file:line evidence and the commit sha that resolved it.

## Test plan

- [ ] \`go test -race ./...\` passes in every changed module
- [ ] \`go vet ./...\` passes in every changed module
- [ ] \`go build ./...\` passes in every changed service
- [ ] \`docker build -f services/<svc>/Dockerfile .\` passes for every service whose go.mod or Dockerfile is touched (none expected in this PR)
- [ ] PR overlay smoke test with two tenants does not regress per-tenant reads/writes
EOF
)"
```

- [ ] **Step 3: Capture the PR URL in the task folder**

```bash
gh pr view --json url -q .url >> docs/tasks/task-041-tenant-filter-leaks/audit.md
git add docs/tasks/task-041-tenant-filter-leaks/audit.md
git commit -m "docs(task-041): record PR URL in audit.md"
```

---

## Self-Review

**Spec coverage check (PRD §10 acceptance criteria → tasks):**

| PRD criterion | Task |
|---|---|
| `audit.md` lists every GORM call site classified with file:line evidence | Task 3 |
| Every classified leak fixed in this PR including F6 callback hardening | Task 2 (F6) + Task 6 (everything else) |
| Regression tests for atlas-guilds (`getAll`, `getById`, `getForName`) | Task 4 |
| Regression tests for atlas-character (`getById`, `getForAccount`, `getForAccountInWorld`, `getForName`, `getAll`) | Task 5 |
| At least one read + one write provider test per tenant-scoped service | Task 8 |
| Two-tenant overlap fixtures, tenant isolation asserted for reads + writes | Tasks 4, 5, 8 (template) |
| F6 regression test: Create with zero TenantId persists context tenant | Task 2 Step 1–4 |
| `go test -race ./...` passes in every changed module | Task 9 Step 2 |
| `go vet ./...` passes in every changed module | Task 9 Step 3 |
| `go build ./...` passes in every changed service | Task 9 Step 4 |
| `docker build` per touched-Dockerfile service | Task 9 Step 6 |
| PR description references audit.md + F-class breakdown + F6 callback change | Task 11 Step 2 |

No gaps.

**Placeholder scan:** Searched for "TBD", "TODO", "fill in", "similar to" — none. The audit-driven loop in Task 6 is templated per F-class with full code for each template, not a placeholder. The per-service test in Task 8 is templated with a complete example and a checklist; the engineer fills in the package-specific entity fields when applying the template, but the template itself is complete code.

**Type consistency:** `NewInMemoryTenantDB(t, migrations ...Migrator)` and `TenantContext(id uuid.UUID)` are defined in Task 1 and used identically in Tasks 4, 5, 7, 8. `injectTenantIdIfZero` is defined once in Task 2. The `tenantCreateCallback` signature is preserved. F-class commit message format `fix(task-041, atlas-<svc>): ... (F<n>)` is consistent across templates.
