# Tenant Filter Leaks — Audit & Fix — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-17
---

## 1. Overview

Atlas is a multi-tenant Go microservices game server. Tenant isolation is enforced primarily at the GORM query layer — every persistent entity carries a `tenant_id` column, and every read/write is expected to filter on the tenant from request context. A code audit on 2026-05-17 surfaced multiple GORM providers that issue queries without a `tenant_id` predicate, allowing one tenant's request to read or modify another tenant's data when the same primary key or unique column value collides across tenants.

Confirmed leaks (file:line):
- `services/atlas-guilds/atlas.com/guilds/guild/provider.go:10` (`getAll`), `:21` (`getById`), `:32` (`getForName`).
- `services/atlas-character/atlas.com/character/character/provider.go:11` (`getById`), `:17` (`getForAccountInWorld`), `:23` (`getForAccount`), `:29` (`getForName`), `:40` (`getAll`).

The audit was limited in scope; the same anti-pattern is likely present across the other 30 GORM-using services. This task delivers (a) a comprehensive audit of every GORM provider and write call site across all Go services, (b) per-query `tenant_id` filter fixes, and (c) regression tests using testcontainers Postgres so future drift is caught at PR time.

## 2. Goals

Primary goals:
- Enumerate every GORM read/write call site across `services/atlas-*` that operates on a tenant-scoped entity but omits a `tenant_id` filter.
- Fix each identified leak by adding `tenant_id = ?` (or `WHERE tenant_id = ?` chained via the existing `tenant.MustFromContext(ctx)` pattern) to the query.
- Add regression tests with two-tenant fixtures that fail if the tenant filter is removed.
- Ship as one bundled PR for atomic deploy/rollback.

Non-goals:
- Introducing a GORM global plugin or callback to auto-inject `tenant_id` (decided against — too broad a blast radius; admin/migration queries would need opt-out plumbing).
- Refactoring the `EntityProvider` / `database.Query` / `database.SliceQuery` abstractions.
- Adding `tenant_id` columns to entities that don't currently have them (separate work).
- CI lint to catch future regressions (deferred; revisit if drift recurs).
- Changes to non-GORM data stores (Redis, in-memory caches) — those already key by `tenant_id` in their cache keys.

## 3. User Stories

- As a server operator, I want guarantees that tenant A's data cannot be observed or mutated by tenant B's requests, so the platform meets its multi-tenancy contract.
- As a player on tenant A, I want my character data to be invisible to a request on tenant B even if our character IDs collide, so my account is not affected by another tenant's actions.
- As a backend engineer, I want regression tests that fail when a GORM provider omits `tenant_id`, so I cannot silently reintroduce the leak.

## 4. Functional Requirements

### 4.1 Audit phase

- Enumerate every Go file under `services/atlas-*` that declares a GORM provider function (typical shape: `func name(args) database.EntityProvider[T]`) or directly invokes `db.Create / db.Save / db.Updates / db.Delete / db.Exec`.
- For each call site, determine whether the target entity has a `tenant_id` (or equivalent) column. If yes, verify the query includes a `tenant_id` predicate.
- Produce an audit table in `audit.md` (committed alongside this PRD) listing: service, file:line, function name, entity, leak Y/N, fix applied Y/N.
- Treat the following as in-scope query operations: SELECT (`First`, `Find`, `Take`, `Scan`, raw `Exec` reads), INSERT (`Create`), UPDATE (`Updates`, `Save`, `UpdateColumns`), DELETE (`Delete`, raw `Exec`).
- Treat the following as out-of-scope: queries against tables that intentionally span tenants (e.g., a global registry table — these must be explicitly justified in the audit doc).

### 4.2 Fix phase

- For each confirmed leak, add a tenant filter using the established pattern in the codebase. The canonical reference is `atlas-account` which reads `tenant.MustFromContext(ctx)` and passes the tenant model into providers (see `services/atlas-account/atlas.com/account/account/processor.go:82`).
- The fix must be per-query (provider signature gains a `tenantId uuid.UUID` parameter if needed, or the caller passes a pre-scoped `*gorm.DB`). Do not introduce a GORM plugin or callback.
- INSERT/UPDATE calls already supplying `tenant_id` via the struct field pass; calls that allow client input to set the tenant_id must be hardened to ignore the client value and use context.
- Where a function genuinely needs cross-tenant access (e.g., a Kafka-driven housekeeping task), rename it to make the intent explicit (e.g., `getAllAcrossTenants`) and add a comment justifying it.

### 4.3 Test phase

- Add provider-level regression tests using testcontainers Postgres for every fixed call site.
- Each test must:
  - Insert at least two rows belonging to two distinct `tenant_id` values, with overlapping non-tenant key data (e.g., same character name, same guild id).
  - Invoke the provider with tenant A's context.
  - Assert only tenant A's rows are returned (for reads) or affected (for writes).
- Tests must live next to the provider (`provider_test.go`) and use the project's existing Builder pattern for fixture construction.
- A shared test helper for testcontainers Postgres setup may be added under `libs/atlas-database` if one does not already exist; if it does, reuse it.

## 5. API Surface

No external API surface changes. All changes are internal to provider/repository layers.

Possible internal signature changes:
- Providers gaining a `tenantId uuid.UUID` (or equivalent) parameter where they did not previously have one. Callers in processors must thread the tenant through. Processor public APIs remain unchanged.

## 6. Data Model

No schema changes. The `tenant_id` columns and their indexes already exist on affected entities. If the audit surfaces a tenant-scoped entity whose table is missing an index on `tenant_id`, that index addition is in scope; data backfill is not required.

## 7. Service Impact

Confirmed services requiring fixes:
- **atlas-guilds** — provider.go in the `guild` package (3 functions).
- **atlas-character** — provider.go in the `character` package (5 functions).

Likely services requiring audit (database consumers per the services.json + go.mod survey):
- atlas-account, atlas-asset-expiration, atlas-ban, atlas-buddies, atlas-buffs, atlas-cashshop, atlas-chairs, atlas-chalkboards, atlas-character-factory, atlas-consumables, atlas-drops, atlas-effective-stats, atlas-expressions, atlas-fame, atlas-families, atlas-gachapons, atlas-inventory, atlas-invites, atlas-keys, atlas-marriages, atlas-merchant, atlas-messages, atlas-messengers, atlas-monster-book, atlas-notes, atlas-parties, atlas-party-quests, atlas-pets, atlas-quest, atlas-saga-orchestrator, atlas-skills, atlas-storage.

Out-of-scope services (no Go GORM use): atlas-ui, atlas-assets, atlas-data (read-only WZ data, no tenants), atlas-wz-extractor, atlas-pr-bootstrap, atlas-runtime-orchestrator, atlas-tenants (the tenant registry itself).

## 8. Non-Functional Requirements

### Security
- Each fix closes a multi-tenant data-leak surface. Fixes must not introduce a regression where an in-tenant query unexpectedly returns no rows. Tests must cover both tenant-isolation and same-tenant-correctness cases.

### Observability
- No new metrics required. Existing tracing/logging continues to capture tenant_id via headers.

### Performance
- Adding `tenant_id` to WHERE clauses should be net-neutral or positive if a composite index exists. Spot-check `EXPLAIN ANALYZE` on the highest-traffic affected queries (character `getById`, guild `getById`) before/after via testcontainers; document if any new index is needed.

### Backward compatibility
- No client-visible behavioral changes for legitimate single-tenant traffic. Any traffic relying on cross-tenant lookups was a bug.

### Migration / rollout
- Single bundled PR. After merge, the PR-overlay environment must run integration smoke tests with two tenants present before the change reaches main.

## 9. Open Questions

- **Cross-tenant queries that are legitimate** — does the saga orchestrator, Kafka housekeeping, or any reaper task rely on cross-tenant SELECTs today? The audit must identify these and either leave them untouched (with an explicit comment) or split into a separate cross-tenant API.
- **Asset expiration / monster id allocator interactions** — `atlas-asset-expiration` and the `atlas-object-id` allocator key by tenant in Redis but operate on assets in Postgres; verify the bridge enforces tenant scoping at both ends.
- **Existing testcontainers helper** — confirm whether `libs/atlas-database` already exports a Postgres test harness; if so, reuse it; if not, design phase decides where the helper lives.

## 10. Acceptance Criteria

- [ ] `audit.md` exists in this task folder, lists every GORM provider and write call site across all `services/atlas-*` Go services, and marks each as either filtered, intentionally cross-tenant (with justification), or fixed.
- [ ] Every leak identified in §1 and the audit is patched in a single PR on branch `task-041-tenant-filter-leaks`.
- [ ] Regression tests exist for atlas-guilds (`getAll`, `getById`, `getForName`) and atlas-character (`getById`, `getForAccount`, `getForAccountInWorld`, `getForName`, `getAll`), and for at least one read and one write provider per other affected service.
- [ ] Regression tests are written against testcontainers Postgres, use two-tenant fixtures, and assert tenant isolation.
- [ ] `go test -race ./...` passes in every changed service.
- [ ] `go vet ./...` passes in every changed service.
- [ ] `go build ./...` passes in every changed service.
- [ ] `docker build -f services/<svc>/Dockerfile .` passes for every service whose go.mod or Dockerfile is touched.
- [ ] PR description summarizes the audit results, lists fixed call sites, and links to `audit.md`.
