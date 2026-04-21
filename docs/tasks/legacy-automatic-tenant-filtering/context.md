# Automatic Tenant Filtering — Context

**Last Updated: 2026-02-19**

## Current State

**Phase 3 is nearly complete.** All services except atlas-saga-orchestrator have been migrated. An agent is currently running for atlas-saga-orchestrator.

### Completed Services (all tests pass, all builds succeed)
Phase 2 pilots: atlas-notes, atlas-fame, atlas-guilds
Phase 3.1 (20 services): atlas-keys, atlas-quest, atlas-skills, atlas-maps, atlas-ban, atlas-buddies, atlas-pets, atlas-character, atlas-inventory, atlas-cashshop, atlas-storage, atlas-families, atlas-npc-shops, atlas-npc-conversations, atlas-portal-actions, atlas-reactor-actions, atlas-map-actions, atlas-party-quests, atlas-marriages, atlas-drop-information
Phase 3.2: atlas-account
Phase 3.4: atlas-configurations (was already migrated)
Phase 3.5: atlas-data
Phase 3.6: atlas-gachapons

### In Progress
- **atlas-saga-orchestrator** (Phase 3.3) — agent a01af4a running, migrating store.go with WithoutTenantFilter for cross-tenant queries

## Key Files

### Shared Library
- `libs/atlas-database/tenant_scope.go` — GORM callbacks for automatic tenant filtering
- `libs/atlas-database/connection.go:123` — `Connect()` internally calls `registerTenantCallbacks(l, db)`
- `libs/atlas-database/tenant_scope.go:39` — `RegisterTenantCallbacks()` exported for test files

### Critical Implementation Details

1. **`database.Connect()` already registers callbacks** — Do NOT add `RegisterTenantCallbacks` to main.go files. Only add it in test setup functions where SQLite is created directly via `gorm.Open()`.

2. **Create callback handles slices** — Bug was found and fixed during atlas-quest migration. The `tenantCreateCallback` now handles `reflect.Struct`, `reflect.Slice`, and `reflect.Array` cases to avoid panics when GORM cascades saves to associations.

3. **GORM zero-value gotcha** — When removing TenantId from struct-based WHERE queries, always use string-based `.Where("column = ?", value)` instead of struct-based `db.Where(&Entity{Field: value})`. GORM skips zero-value fields in struct queries.

4. **Batch deletes need WHERE clause** — GORM requires a WHERE clause for batch deletes. When removing `Where("tenant_id = ?", tenantId)`, replace with `Where("1 = 1")` — the tenant callback will still inject the tenant filter.

## Migration Pattern Applied to Each Service

### Provider changes
- Remove `tenantId uuid.UUID` / `tenant.Model` parameter
- Remove `"tenant_id = ? AND"` from WHERE clauses
- Remove `TenantId: tenantId` from struct-based WHERE
- For "get all" queries that only filtered by tenant: use bare `db.Find(&results)`

### Administrator changes
- **Keep** `tenantId` in create functions (needed to set entity field)
- **Remove** `tenantId` from update/delete functions
- Switch struct-based WHERE to string-based WHERE

### Processor changes
- Change all `p.db` to `p.db.WithContext(p.ctx)`
- Change `database.ExecuteTransaction(p.db, ...)` to `database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)`
- Remove `p.t.Id()` from provider/administrator calls (except create)

### Test changes
- Add `database.RegisterTenantCallbacks(l, db)` to test DB setup functions
- Update test provider/administrator calls to match new signatures

## Key Discoveries This Session

1. **atlas-account `allEntities()` was dead code** — Removed entirely. No `WithoutTenantFilter` needed.
2. **atlas-configurations was already migrated** — Providers already used `db.WithContext(ctx)` without manual tenant_id filtering.
3. **atlas-npc-conversations has pre-existing build failure** — `kafka/consumer/saga/consumer.go:79,89` uses variable `t` (string) where `ctx` (context.Context) is needed. Unrelated to migration.
4. **atlas-character agent deleted local database package** — Replaced `database/connection.go`, `database/provider.go`, `database/transaction.go` with `github.com/Chronicle20/atlas-database` imports.

## Next Steps After atlas-saga-orchestrator Completes

1. **Verify saga-orchestrator agent results** — Check tests pass, build succeeds
2. **Check for redundant RegisterTenantCallbacks in main.go** — Some early agents (atlas-keys, atlas-maps) added it to main.go before we caught this; it was removed. Verify no others slipped through.
3. **Phase 4 cleanup** — Standardize TenantId naming, fix atlas-npc-conversations build error, update developer guidelines
4. **Commit all changes** — Large commit covering entire Phase 3 migration

## Dependencies

```
Phase 1 (Library) ✅
  └── Phase 2 (Pilot: notes, fame, guilds) ✅
        └── Phase 3 (Bulk migration: 26 services) ✅ (except saga-orchestrator in progress)
              └── Phase 4 (Cleanup: standardize naming, docs)
```
