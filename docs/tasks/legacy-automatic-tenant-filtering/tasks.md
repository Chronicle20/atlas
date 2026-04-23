# Automatic Tenant Filtering — Task Checklist

**Last Updated: 2026-02-19**

## Phase 1: Shared Database Library (`libs/atlas-database`)

- [x] 1.1 Create `libs/atlas-database` Go module and add to `go.work`
- [x] 1.2 Implement `Connect()` with GORM tenant-scoping callbacks
  - [x] Port DSNBuilder, Migrator, Configuration pattern
  - [x] Register Query callback (injects WHERE tenant_id = ?)
  - [x] Register Create callback (validates TenantId field set)
  - [x] Register Update callback (injects WHERE tenant_id = ?)
  - [x] Register Delete callback (injects WHERE tenant_id = ?)
  - [x] Entity field detection via `stmt.Schema.FieldsByDBName["tenant_id"]`
  - [x] Log warning (not error) when tenant missing from context
- [x] 1.3 Implement `WithoutTenantFilter(ctx)` / `SkipTenantFilter` escape hatch
- [x] 1.4 Write comprehensive test suite
  - [x] Query with tenant context → adds WHERE
  - [x] Query without tenant context → warning, no WHERE
  - [x] Create with tenant → validates TenantId set
  - [x] Update/Delete with tenant → adds WHERE
  - [x] WithoutTenantFilter → skips injection
  - [x] Entity without TenantId field → callback is no-op
  - [ ] Preload → tenant filter on parent only, not FK-based children

## Phase 2: Pilot Migration

- [x] 2.1 Migrate **atlas-notes** (simplest: single entity)
- [x] 2.2 Migrate **atlas-fame** (single entity, delete operations)
- [x] 2.3 Migrate **atlas-guilds** (complex: Preload, nested entities)
- [ ] 2.4 Validate pilot services in dev environment

## Phase 3: Bulk Service Migration

- [x] 3.1 Migrate straightforward services (no cross-tenant queries)
  - [x] atlas-keys
  - [x] atlas-quest
  - [x] atlas-skills
  - [x] atlas-maps (visit package)
  - [x] atlas-ban
  - [x] atlas-buddies
  - [x] atlas-pets
  - [x] atlas-character
  - [x] atlas-inventory
  - [x] atlas-cashshop
  - [x] atlas-storage
  - [x] atlas-families
  - [x] atlas-npc-shops
  - [x] atlas-npc-conversations
  - [x] atlas-portal-actions
  - [x] atlas-reactor-actions
  - [x] atlas-map-actions
  - [x] atlas-party-quests
  - [x] atlas-marriages
  - [x] atlas-drop-information
- [x] 3.2 Migrate **atlas-account** (cross-tenant `allEntities()` was dead code — removed, no WithoutTenantFilter needed)
  - [x] Removed dead `allEntities()` function
  - [x] Normal queries are tenant-scoped
  - [x] Run tests + build
- [x] 3.3 Migrate **atlas-saga-orchestrator** (cross-tenant recovery + reaper)
  - [x] Add WithoutTenantFilter to GetAllActive()
  - [x] Add WithoutTenantFilter to GetTimedOut()
  - [x] Verify normal saga queries are tenant-scoped
  - [x] Run tests + build
- [x] 3.4 Migrate **atlas-configurations** (already migrated — providers already use db.WithContext(ctx))
- [x] 3.5 Migrate **atlas-data** (document storage with tenant scope)
  - [x] Run tests + build
- [x] 3.6 Migrate **atlas-gachapons** (uses shared Query/SliceQuery helpers)
  - [x] Run tests + build

## Phase 4: Cleanup

- [x] 4.1 Remove per-service `database/connection.go` files (removed during migration by atlas-character agent)
- [x] 4.2 Remove tenant parameter from provider/administrator function signatures
  - Done as part of each service migration (providers no longer take tenantId, administrators keep it only for create)
- [x] 4.3 Standardize TenantId field naming
  - [x] atlas-notes: TenantID → TenantId
  - [x] atlas-maps/visit: TenantID → TenantId
  - [x] atlas-tenants/configuration: TenantID → TenantId (+ model accessor, builder methods, tests)
- [x] 4.4 Update developer guidelines and documentation
  - [x] patterns-multitenancy-context.md — Added automatic filtering section, test setup, gotchas
  - [x] file-responsibilities.md — Updated provider/administrator signatures (no tenantId)
  - [x] patterns-provider.md — Updated example, added automatic tenant filtering section
  - [x] anti-patterns.md — Added 4 new anti-patterns for manual tenant filtering
  - [x] testing-guide.md — Added RegisterTenantCallbacks and WithContext checklist items
  - [x] scaffolding-checklist.md — Added database & tenant filtering notes
- [x] 4.5 Fix build error in atlas-npc-conversations (`kafka/consumer/saga/consumer.go:79,89` — `t` → `ctx`)
  - Also fixed test compile error in `operation_executor_test.go` (SetContext/ClearContext need context.Context not tenant.Model)
  - Note: test still panics on nil registry (pre-existing — registry not initialized in test)
- [x] 4.6 Verify no services have redundant `RegisterTenantCallbacks` in main.go — confirmed clean
