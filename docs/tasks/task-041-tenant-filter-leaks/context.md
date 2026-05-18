# task-041 Implementation Context

> Companion to `plan.md` and `design.md`. Read this first when starting a fresh session — it answers "where do I look for X" before the plan tells you "do Y".

## Where things live

### Tenant infrastructure (the invariant this task hardens)

- `libs/atlas-database/tenant_scope.go` — the four GORM callbacks (`tenant:query`, `tenant:row`, `tenant:create`, `tenant:update`, `tenant:delete`). `tenantQueryCallback` injects `WHERE tenant_id = ?`. `tenantCreateCallback` currently only warns when `TenantId` is zero — this task changes it to inject.
- `libs/atlas-database/tenant_scope_test.go` — canonical in-memory sqlite test pattern (`setupTestDB`, `tenantContext`, `tenantEntity` / `globalEntity` fixtures). `TestDoubleWhereIsHarmless` proves manual `WHERE tenant_id = ?` + callback is safe.
- `libs/atlas-database/connection.go:123` — `registerTenantCallbacks(l, db)` invoked by `database.Connect` for every service. Always wired in prod.
- `libs/atlas-database/transaction.go` — `ExecuteTransaction(db, fn)` reuses an in-flight tx if `db` is already in one. The inner `tx` inherits the parent statement context, so a parent `db.WithContext(p.ctx)` carries through.
- `libs/atlas-database/provider.go` — `EntityProvider[E]`, `Query`, `SliceQuery`. Providers are functions of `*gorm.DB` returning a `model.Provider[E]`. The processor's `p.db.WithContext(p.ctx)` is where context lands on the chain — providers themselves never see the context.
- `libs/atlas-tenant/` — `tenant.WithContext(ctx, t)`, `tenant.FromContext(ctx)()`, `tenant.Create(id, region, version, regionId)`.

### Atlas-guilds (PRD-named target)

- `services/atlas-guilds/atlas.com/guilds/guild/provider.go` — `getAll`, `getById`, `getForName`. Uses `Preload("Members")`, `Preload("Titles")`.
- `services/atlas-guilds/atlas.com/guilds/guild/member/entity.go` — has `TenantId` (verified).
- `services/atlas-guilds/atlas.com/guilds/guild/title/entity.go` — has `TenantId` (verified).
- `services/atlas-guilds/atlas.com/guilds/guild/task.go:43` — reference good pattern for F2 fix: `tctx := tenant.WithContext(sctx, g.Tenant())` per tenant inside a background loop.

### Atlas-character (PRD-named target)

- `services/atlas-character/atlas.com/character/character/provider.go` — `getById`, `getForAccountInWorld`, `getForAccount`, `getForName`, `getAll`. All use `database.Query` / `database.SliceQuery` and rely on `p.db.WithContext(p.ctx)` upstream from the processor.

### Known cross-tenant call sites (PASS-CROSS-TENANT in audit)

- `services/atlas-merchant/atlas.com/merchant/shop/task.go` — startup recovery enumerates shops across tenants.
- `services/atlas-merchant/atlas.com/merchant/frederick/{task.go,notification_task.go}` — same shape.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go` — saga recovery across tenants.
- `services/atlas-data/atlas.com/data/searchindex/searchindex.go` — global WZ search index, no tenant axis (data service is read-only).

These already use `WithoutTenantFilter`. Audit confirms each scope is justified and bounded; classifies them PASS-CROSS-TENANT.

### Services explicitly out of scope (no Go GORM tenanted use)

`atlas-ui`, `atlas-assets`, `atlas-data` (WZ read-only), `atlas-wz-extractor`, `atlas-pr-bootstrap`, `atlas-runtime-orchestrator`, `atlas-tenants` (registry itself). Do not touch these.

## Key design decisions (from design.md §11, locked)

- **OQ-1 — Option A.** Trust the callback. Audit closes F1–F10 gaps. No defense-in-depth duplicate WHERE clauses. Tenant scoping is a per-provider decision (most are scoped; a justified subset is intentionally cross-tenant).
- **OQ-2 — Bundled.** F6 hardening (`tenantCreateCallback`: warn → inject) lands in the same PR as the audit + consumers. Single atomic change.
- **OQ-3 — Single-PR migration.** F3/F8 child-table column additions use one idempotent `Migration` per entity. No two-phase deploy. Tables are small.
- **OQ-4 — Sqlite, defer testcontainers.** Use the existing in-memory sqlite pattern from `libs/atlas-database/tenant_scope_test.go`. Plain `go test -race` in CI. No Docker dep in the verification loop.
- **OQ-5 — Strict per-PRD §10.** Regression test per fix + one read + one write per tenant-scoped service. ~30 services × 2 thin tests = acceptable budget.

## Threat model (F1–F10) — abbreviated

The plan's per-class fix templates assume you've read this.

| F | Failure mode | One-line cue |
|---|---|---|
| F1 | No `WithContext` | callback can't find ctx → bails silently |
| F2 | Ctx has no tenant | `tenant.FromContext` errors → callback bails |
| F3 | Entity lacks `tenant_id` | `hasTenantColumn` false → callback skips |
| F4 | Raw SQL (`Exec`/`Raw`) | callback not invoked at all |
| F5 | Join target lacks `tenant_id` | join leaks rows whose FK collides across tenants |
| F6 | Create with zero `TenantId` | row written with zero UUID, invisible to real tenants |
| F7 | Struct-where with non-context tenant_id | yields zero rows silently when sources disagree |
| F8 | Preload of tenant-less child | same as F3 for the preload target |
| F9 | Test setup skips `RegisterTenantCallbacks` | false-positive green tests |
| F10 | Over-broad `WithoutTenantFilter` | bypass scope leaks into downstream tenant-aware calls |

## Dependencies between tasks

```
Task 1 (helper) ──┬─→ Task 4 (atlas-guilds tests)
                  ├─→ Task 5 (atlas-character tests)
                  ├─→ Task 7 (atlas-guilds preload test)
                  └─→ Task 8 (per-service smoke)
Task 2 (F6 hardening) — independent; can land before or after audit
Task 3 (audit) ───┬─→ Task 6 (per-F-class fixes) — every LEAK row maps to a Task 6 template
                  └─→ Task 8 (the "Tenant-scoped services" section is the checklist)
Task 6 + 8 ──────→ Task 9 (verification)
Task 9 ──────────→ Task 10 (code review)
Task 10 ─────────→ Task 11 (PR)
```

Tasks 1 and 2 are independent and can run in parallel; everything else has to wait on the audit (Task 3) for scope.

## Useful one-liners

Find every provider function that issues `db.Where`/`Find`/`First` without `WithContext` upstream:
```bash
rg --type go -B3 'p\.db\.(Where|Find|First|Create|Save|Updates|Delete)\(' services/ \
  | rg -B0 -A4 -v 'WithContext'
```

Find every entity declaring `TenantId`:
```bash
rg --type go -l 'TenantId\s+uuid\.UUID' services/ | sort -u
```

Find entity files without `TenantId` (candidate F3 if join target is tenant-scoped):
```bash
find services -path "*/atlas-*/*entity.go" | xargs rg -L 'TenantId'
```

Find every `WithoutTenantFilter` usage:
```bash
rg --type go -n 'WithoutTenantFilter' services/ libs/ | rg -v _test.go
```

Find every raw SQL site:
```bash
rg --type go -n -e '\.Raw\(' -e '\.Exec\(' services/ | rg -v _test.go
```

## Verification reference (CLAUDE.md §Build & Verification)

Before claiming the branch is done:

1. `go test -race ./...` clean in every changed module.
2. `go vet ./...` clean in every changed module.
3. `go build ./...` clean in every changed service.
4. `docker build -f services/<svc>/Dockerfile .` from worktree root for every service whose `go.mod` or `Dockerfile` was touched. **Mandatory if either file changes** — `go build`/`go test` against the workspace `go.work` will NOT catch the four-place hand-edited COPY drift.

For this task: only `libs/atlas-database/tenant_scope.go` + `libs/atlas-database/testdb.go` + service-level test files change in the common case. No Dockerfile touch expected. Step 4 is a no-op unless an F3/F8 fix forces a new import that requires a Dockerfile update.

## Test helper / fixture pattern

CLAUDE.md §Test Helper Pattern: use the project's Builder pattern. **Do not** create `*_testhelpers.go` files with test-only constructors. The `database.NewInMemoryTenantDB` helper from Task 1 is a generic test fixture, not a domain test helper — it belongs in the lib, not in a `*_testhelpers.go` somewhere.

## File ownership map (who-touches-what)

| File | Owner of changes |
|---|---|
| `libs/atlas-database/testdb.go` | Task 1 |
| `libs/atlas-database/testdb_test.go` | Task 1 |
| `libs/atlas-database/tenant_scope.go` | Task 2 |
| `libs/atlas-database/tenant_scope_test.go` | Task 2 |
| `docs/tasks/task-041-tenant-filter-leaks/audit.md` | Task 3, 6.Final, 10, 11 |
| `services/atlas-guilds/atlas.com/guilds/guild/provider_test.go` | Task 4, Task 7 |
| `services/atlas-character/atlas.com/character/character/provider_test.go` | Task 5 |
| Any service file flagged LEAK by audit | Task 6 |
| `services/atlas-<svc>/.../provider_test.go` (×~30) | Task 8 |
| PR description | Task 11 |

No file is owned by more than one task except `audit.md` (Task 3 writes the inventory; Task 6.Final marks fixes; Tasks 10–11 append reviewer/PR notes) and the atlas-guilds `provider_test.go` (Task 4 base + Task 7 preload test).
