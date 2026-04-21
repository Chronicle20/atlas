# Automatic Tenant Filtering via GORM Global Scopes

**Last Updated: 2026-02-19**

## Executive Summary

Every database query across 29 Atlas services manually adds `WHERE tenant_id = ?` filtering. This manual approach creates a class of bugs where a single forgotten clause can leak data across tenants. This plan introduces a shared GORM callback mechanism that automatically injects tenant filtering from `context.Context`, eliminating the risk of accidental cross-tenant data access while reducing boilerplate code.

## Current State Analysis

### Scale of the Problem
- **29 services** use GORM with PostgreSQL via per-service `database/connection.go`
- **87 Go files** reference `tenant_id` in queries
- **3 distinct filtering patterns** are used inconsistently across services:
  1. Struct-based: `db.Where(&Entity{TenantId: tenantId})`
  2. String-based: `db.Where("tenant_id = ?", tenantId)`
  3. Map-based: `db.Where(map[string]interface{}{"tenant_id": tenantId})`
- Every `EntityProvider`, `administrator`, and `provider` function manually accepts and forwards `tenant.Model` or `uuid.UUID` for filtering

### Current Architecture
```
main.go → database.Connect() → *gorm.DB (singleton per service)
    ↓
Processor (holds ctx, db, tenant.Model)
    ↓
provider.go / administrator.go (manually adds WHERE tenant_id = ?)
```

Key observations:
- `database/connection.go` is **copy-pasted** across all 29 services (no shared library)
- Tenant is extracted via `tenant.MustFromContext(ctx)` in processor constructors
- `*gorm.DB` is created once at startup and shared across all requests
- GORM's `db.WithContext(ctx)` is **not currently used** anywhere

### Legitimate Cross-Tenant Queries (Must Be Preserved)
| Service | Function | Purpose |
|---------|----------|---------|
| atlas-account | `allEntities()` | Teardown: list all accounts across tenants |
| atlas-saga-orchestrator | `GetAllActive()` | Startup recovery across all tenants |
| atlas-saga-orchestrator | `GetTimedOut()` | Stale saga reaper across all tenants |

### Entities Without tenant_id (Exempt from Filtering)
| Service | Entity | Reason |
|---------|--------|--------|
| atlas-tenants | `tenant.Entity` | IS the tenant table itself |
| atlas-configurations | `services.Entity` | Global service config, not tenant-scoped |
| atlas-configurations | `templates.Entity` | Global templates, not tenant-scoped |
| atlas-buddies | `buddy.Entity` | Child entity, accessed via parent FK |
| atlas-inventory | `asset.Entity` (sub-entities) | Child entity, accessed via parent FK |
| atlas-quest | `progress.Entity` | Child entity, accessed via parent FK |

### TenantId Field Naming Inconsistency
Entities use inconsistent naming for the tenant column:
- `TenantId uuid.UUID` (most common)
- `TenantID uuid.UUID` (atlas-notes, atlas-maps/visit, atlas-tenants/configuration)
- All map to `tenant_id` column in PostgreSQL (GORM convention)

## Proposed Future State

### Architecture
```
main.go → database.Connect() → *gorm.DB with tenant callback registered
    ↓
Processor (uses db.WithContext(ctx) to propagate tenant)
    ↓
GORM callback auto-injects WHERE tenant_id = ? from context
    ↓
provider.go / administrator.go (no manual tenant filtering needed)
```

### Design Decisions

**1. Shared Database Library (`libs/atlas-database`)**
Create a new shared library that wraps GORM connection setup and registers tenant-scoping callbacks. Services import this instead of copy-pasting `database/connection.go`.

**2. GORM Callback-Based Scoping (not Global Scopes)**
Use GORM's callback mechanism (`db.Callback().Query().Before("gorm:query")`) rather than GORM Scopes. Callbacks are registered once on the `*gorm.DB` instance and apply to all operations automatically, whereas Scopes must be applied per-query via `db.Scopes(...)`.

**3. Context-Driven Tenant Propagation**
The tenant UUID is already stored in `context.Context` via `tenant.WithContext()`. GORM's `db.WithContext(ctx)` will carry it through to callbacks.

**4. Opt-Out for Cross-Tenant Queries**
A context key `SkipTenantFilter` allows specific queries to bypass automatic filtering:
```go
ctx = database.WithoutTenantFilter(ctx)
db.WithContext(ctx).Find(&results)
```

**5. Table Detection via Entity Reflection**
The callback inspects the GORM statement's destination struct to check if it has a `TenantId`/`TenantID` field. If the entity has no tenant field, the callback is a no-op. This handles exempt entities automatically.

### Key Benefits
- **Eliminates tenant data leakage bugs** — impossible to forget the WHERE clause
- **Reduces ~300+ lines of boilerplate** per service (tenant parameter threading)
- **Single source of truth** for database connection logic (no more copy-paste)
- **Backward compatible** — can be adopted incrementally, service by service

## Implementation Phases

### Phase 1: Shared Database Library (libs/atlas-database)
Create the foundational library with tenant-scoping callbacks.

**Effort: L**

#### Tasks
1.1 **Create `libs/atlas-database` module** — S
   - Initialize Go module with `go.mod`
   - Add to `go.work`
   - Acceptance: Module compiles, is accessible from services

1.2 **Implement connection setup with tenant callbacks** — M
   - Port `DSNBuilder`, `Connect()`, `Migrator` pattern from existing services
   - Register GORM callbacks for Query, Create, Update, Delete, Row operations
   - Callback logic:
     - Extract `tenant.Model` from `stmt.Context` via `tenant.FromContext()`
     - Check if statement's model struct has `TenantId`/`TenantID` field (reflect)
     - If both present and `SkipTenantFilter` not set: inject `WHERE tenant_id = ?`
     - For Create: verify the entity's TenantId field is set (log warning if zero)
   - Acceptance: Unit tests with miniredis/sqlite prove auto-filtering works

1.3 **Implement `WithoutTenantFilter(ctx)` escape hatch** — S
   - Context key to bypass tenant injection
   - Acceptance: Cross-tenant queries work when opt-out is used

1.4 **Write comprehensive tests** — M
   - Test Query with tenant context → adds WHERE clause
   - Test Query without tenant context → no WHERE clause (safe: returns nothing for tenant-scoped tables)
   - Test Create with tenant → validates TenantId field set
   - Test Update/Delete with tenant → adds WHERE clause
   - Test WithoutTenantFilter → skips injection
   - Test entity without TenantId field → callback is no-op
   - Test Preload → tenant filter applies to parent, not child associations
   - Acceptance: All tests pass

### Phase 2: Pilot Migration (2-3 Services)
Adopt the library in a small set of representative services to validate the approach.

**Effort: L**

#### Tasks
2.1 **Migrate atlas-notes** (simplest service, single entity) — M
   - Replace `database/connection.go` with `atlas-database` import
   - Update `main.go` to use shared `Connect()`
   - Update processor to use `db.WithContext(ctx)` instead of passing tenant to providers
   - Remove manual `tenant_id` filtering from `provider.go` and `administrator.go`
   - Run existing tests, add integration test
   - Acceptance: All tests pass, build succeeds

2.2 **Migrate atlas-fame** (two entities, delete operations) — M
   - Same pattern as 2.1 but with `deleteByCharacterId` which uses raw WHERE
   - Validates that Delete callbacks work correctly
   - Acceptance: All tests pass, build succeeds

2.3 **Migrate atlas-guilds** (complex: Preload, nested entities, cross-entity queries) — L
   - Validates Preload behavior (Members, Titles loaded via FK, not tenant-scoped themselves in the child query... but parent query is scoped)
   - Tests `db.Save()` path (used in `updateEmblem`, `updateNotice`, `updateCapacity`)
   - Acceptance: All tests pass, build succeeds

2.4 **Validate in dev environment** — M
   - Deploy pilot services
   - Verify no query regressions via logging/tracing
   - Acceptance: Services operate correctly in dev

### Phase 3: Bulk Service Migration
Migrate remaining services using the validated pattern from Phase 2.

**Effort: XL**

#### Tasks
3.1 **Migrate straightforward services** (single entity, no cross-tenant queries) — L
   Services: atlas-keys, atlas-quest, atlas-skills, atlas-maps, atlas-ban, atlas-buddies, atlas-pets, atlas-character, atlas-inventory, atlas-cashshop, atlas-storage, atlas-families, atlas-npc-shops, atlas-npc-conversations, atlas-portal-actions, atlas-reactor-actions, atlas-map-actions, atlas-party-quests, atlas-marriages, atlas-drop-information
   - Apply same migration pattern per service
   - Run tests + build after each
   - Acceptance: All services compile and pass tests

3.2 **Migrate atlas-account** (has cross-tenant `allEntities()`) — M
   - Use `WithoutTenantFilter(ctx)` for teardown function
   - Acceptance: Teardown still works across tenants, normal queries are scoped

3.3 **Migrate atlas-saga-orchestrator** (has cross-tenant recovery + reaper) — M
   - Use `WithoutTenantFilter(ctx)` for `GetAllActive()` and `GetTimedOut()`
   - Acceptance: Recovery and reaper still scan all tenants

3.4 **Migrate atlas-configurations** (mixed: some entities have tenant_id, some don't) — M
   - `tenants.Entity` has `TenantId` on HistoryEntity but not main Entity
   - `services.Entity` has no TenantId
   - `templates.Entity` has no TenantId — uses region/version matching instead
   - Callback's entity reflection handles this automatically
   - Acceptance: All tests pass, global configs still accessible

3.5 **Migrate atlas-data** — M
   - Uses `document.Entity` which has tenant-scoped storage
   - Acceptance: Data loading works correctly per tenant

3.6 **Migrate atlas-gachapons** — M
   - Uses shared `database.Query[E]` / `database.SliceQuery[E]` helper functions
   - These will work automatically when the caller passes `db.WithContext(ctx)`
   - Acceptance: All tests pass

### Phase 4: Cleanup
Remove dead code and standardize patterns.

**Effort: M**

#### Tasks
4.1 **Remove per-service `database/connection.go` files** — M
   - Delete 29 copy-pasted connection files
   - Update imports across all services
   - Acceptance: No service has its own connection.go

4.2 **Remove tenant parameter from provider/administrator function signatures** — L
   - Functions like `entityById(tenant tenant.Model, id uint32)` become `entityById(id uint32)`
   - Provider type changes: `func(db *gorm.DB) model.Provider[E]` remains the same, but tenant injection is automatic
   - This is a large refactor but mechanically straightforward
   - Acceptance: All services compile, no manual tenant filtering remains

4.3 **Standardize TenantId field naming** — S
   - Rename `TenantID` to `TenantId` in entities that use it (atlas-notes, atlas-maps/visit, atlas-tenants/configuration)
   - Database column name unchanged (`tenant_id`)
   - Acceptance: GORM column mapping still works

4.4 **Update developer guidelines** — S
   - Document the automatic tenant filtering pattern
   - Document `WithoutTenantFilter` usage for cross-tenant queries
   - Add to backend-dev-guidelines skill
   - Acceptance: Guidelines updated

## Risk Assessment and Mitigation

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| Callback adds WHERE to queries that intentionally skip tenant (e.g., admin operations) | High | Medium | `WithoutTenantFilter` escape hatch; thorough audit of cross-tenant queries |
| Performance overhead from reflection in callbacks | Low | Low | Cache reflection results per model type; benchmark shows GORM already reflects heavily |
| Preload queries affected by tenant callback | High | Medium | Test carefully — GORM Preload uses separate queries with FKs, not tenant_id. Callback should detect no TenantId field on child entities and skip |
| `db.Save()` affected by tenant callback on Update | Medium | Medium | Save does a full update by PK; callback should add tenant_id to WHERE. Test with guilds service |
| Services using `db.Where(&Entity{TenantId: tenantId})` — double filtering | Low | High | Double WHERE is harmless (same condition twice). Remove manual filtering in Phase 4 for cleanliness |
| Breaking change if callback errors on missing tenant context | High | Medium | Make callback lenient: if no tenant in context, log a warning but don't error. This allows gradual migration |

## Success Metrics

1. **Zero manual `tenant_id` WHERE clauses** remaining after Phase 4
2. **All 29 services** using shared `libs/atlas-database`
3. **No cross-tenant data leakage** possible without explicit `WithoutTenantFilter`
4. **All existing tests pass** without modification (aside from removing test tenant setup)
5. **< 1ms overhead** per query from callback (benchmark target)

## Required Resources and Dependencies

### Dependencies
- `libs/atlas-tenant` — already exists, provides `FromContext()` and `WithContext()`
- GORM v2 — already in use, supports callbacks natively
- `go.work` — already manages workspace, new lib added here

### Resources
- Primarily a single developer task (mechanical migration)
- Dev environment for validation (Phase 2.4)

## Timeline Estimates

| Phase | Duration | Dependencies |
|-------|----------|-------------|
| Phase 1: Shared Library | 1-2 sessions | None |
| Phase 2: Pilot Migration | 2-3 sessions | Phase 1 |
| Phase 3: Bulk Migration | 4-6 sessions | Phase 2 |
| Phase 4: Cleanup | 2-3 sessions | Phase 3 |

**Total: ~10-14 sessions**

Phases 3 and 4 can overlap — cleanup can begin on pilot services while bulk migration continues.
