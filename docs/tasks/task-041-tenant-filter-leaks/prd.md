# Tenant Filter Leaks — Audit & Fix — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-17
---

## 1. Overview

Atlas is a multi-tenant Go microservices game server. Tenant isolation is enforced at the GORM query layer — every persistent entity carries a `tenant_id` column, and every read/write is expected to filter on the tenant from request context.

**Existing infrastructure (discovered during design):** `libs/atlas-database/tenant_scope.go` registers GORM `Before` callbacks for `Query`, `Row`, `Update`, and `Delete` that automatically append `WHERE tenant_id = ?` (from `tenant.FromContext`) on any statement whose schema declares a `tenant_id` column. `database.Connect` wires this up for every service. The Create callback currently only **warns** when `TenantId` is zero — it does not inject it.

An initial audit on 2026-05-17 flagged GORM providers in atlas-guilds and atlas-character as leaks because their function bodies do not include an explicit `tenant_id` predicate:
- `services/atlas-guilds/atlas.com/guilds/guild/provider.go:10` (`getAll`), `:21` (`getById`), `:32` (`getForName`).
- `services/atlas-character/atlas.com/character/character/provider.go:11` (`getById`), `:17` (`getForAccountInWorld`), `:23` (`getForAccount`), `:29` (`getForName`), `:40` (`getAll`).

Re-reading these sites with the callback in mind reclassifies them as callback-covered (PASS-CB) pending audit verification — the callback rewrites each query at execution time. The real leak surface is enumerated as F1–F10 in `design.md` §2 and includes: missing `WithContext` (F1), context without tenant (F2), entity missing `tenant_id` column (F3), raw SQL bypassing callbacks (F4), preloads/joins into child tables without `tenant_id` (F5/F8), Create not setting `TenantId` while the callback only warns (F6), test setups that forget to register the callback (F9), and over-broad `WithoutTenantFilter` scopes (F10).

This task delivers (a) a comprehensive audit of every GORM call site across all Go services classified against F1–F10, (b) targeted fixes per gap class — including hardening the Create callback to inject `tenant_id` when missing, (c) regression tests using the project's existing in-memory sqlite pattern (per `libs/atlas-database/tenant_scope_test.go`).

## 2. Goals

Primary goals:
- Enumerate every GORM read/write call site across `services/atlas-*` and classify each against F1–F10 (PASS-CB, PASS-EXPLICIT, PASS-CROSS-TENANT, LEAK-F<n>, UNCLEAR).
- Close every classified leak: add `WithContext` where missing, harden the existing tenant Create callback to inject `tenant_id` from context (currently only warns), add explicit predicates only where the callback cannot reach (raw SQL, etc.), and add `tenant_id` columns to child tables where the audit determines the column is required.
- Per-provider decision: tenant scoping is intentional, not universal. Cross-tenant call sites continue to use `WithoutTenantFilter` and must be explicitly justified.
- Add regression tests with two-tenant fixtures that fail if the tenant filter is removed.
- Ship as one bundled PR (audit doc + callback hardening + per-service fixes + tests) for atomic deploy/rollback.

Non-goals:
- Replacing or refactoring the existing tenant callback in `libs/atlas-database/tenant_scope.go`. Hardening the Create path (warn → inject) is in scope; the broader callback design is not.
- Refactoring the `EntityProvider` / `database.Query` / `database.SliceQuery` abstractions.
- Removing redundant `TenantId: t.Id()` assignments at Create call sites once the callback injects them — mechanical follow-up, not part of this PR.
- CI lint to catch future regressions (deferred; revisit if drift recurs).
- Testcontainers Postgres integration tests. The existing in-memory sqlite harness (`libs/atlas-database/tenant_scope_test.go`) is sufficient for the leak being tested (WHERE-clause filtering reproduces faithfully on sqlite). Testcontainers may be revisited if Postgres-specific behavior (uuid type semantics, collation, RETURNING) becomes load-bearing.
- Changes to non-GORM data stores (Redis, in-memory caches) — those already key by `tenant_id` in their cache keys.

## 3. User Stories

- As a server operator, I want guarantees that tenant A's data cannot be observed or mutated by tenant B's requests, so the platform meets its multi-tenancy contract.
- As a player on tenant A, I want my character data to be invisible to a request on tenant B even if our character IDs collide, so my account is not affected by another tenant's actions.
- As a backend engineer, I want regression tests that fail when a GORM provider omits `tenant_id`, so I cannot silently reintroduce the leak.

## 4. Functional Requirements

### 4.1 Audit phase

- Enumerate every Go file under `services/atlas-*` that declares a GORM provider function (typical shape: `func name(args) database.EntityProvider[T]`) or directly invokes `db.Create / db.Save / db.Updates / db.Delete / db.Exec / db.Raw`.
- For each call site, classify against F1–F10 (see `design.md` §2): determine whether (a) the entity has a `tenant_id` column, (b) the call chain uses `WithContext` to deliver a tenanted context, (c) the call relies on the callback or uses an explicit predicate, (d) the call deliberately bypasses tenant scoping via `WithoutTenantFilter`, (e) a preload/join target lacks `tenant_id`, or (f) raw SQL is used.
- Produce an audit table in `audit.md` (committed alongside this PRD) listing: service, file:line, function, op (R/W), classification (PASS-CB / PASS-EXPLICIT / PASS-CROSS-TENANT / LEAK-F<n> / UNCLEAR), fix required, notes.
- Treat the following as in-scope query operations: SELECT (`First`, `Find`, `Take`, `Scan`, `Raw` reads), INSERT (`Create`), UPDATE (`Updates`, `Save`, `UpdateColumns`), DELETE (`Delete`, raw `Exec` writes).
- Treat the following as out-of-scope: queries against tables that intentionally span tenants (e.g., a global registry table — these must be classified PASS-CROSS-TENANT with an explicit comment and a verified `WithoutTenantFilter` boundary in the audit doc).

### 4.2 Fix phase

Fix strategy per gap class (full detail in `design.md` §5):
- **F1 (missing `WithContext`)**: thread `WithContext(p.ctx)` onto the call chain. For transaction blocks, verify `tx` inherits context from the parent.
- **F2 (context without tenant)**: convert ad-hoc background contexts to `tenant.WithContext(parent, t)` (pattern: `atlas-guilds/guild/task.go:43`).
- **F3 / F8 (entity or child table missing `tenant_id`)**: add the column, an index, and an idempotent backfill in a single `Migration` function next to the entity. Single-PR migration (no two-phase deploy) — table sizes in scope are small.
- **F4 (raw SQL)**: prefer rewriting to the GORM API. Where unavoidable, parameterize tenant and pass it explicitly.
- **F6 (Create without TenantId)**: harden `tenantCreateCallback` to *inject* `tenant_id` from context onto the reflected struct/slice when the field is zero and the skip flag is unset. Bundled in this PR — its consumers land in the same change.
- **F9 (test setup forgets the callback)**: add an assertion helper or rely on the shared sqlite harness from §4.3.
- **F10 (over-broad `WithoutTenantFilter`)**: each existing call site gets a comment justifying the bypass and a verified scope boundary.
- **Per-provider decision rule**: tenant scoping is intentional, not universal. Cross-tenant call sites remain valid when documented; the audit captures the decision.

### 4.3 Test phase

- Add provider-level regression tests using in-memory sqlite (per the existing `libs/atlas-database/tenant_scope_test.go` pattern) for every fixed call site, plus at least one read and one write provider per service that touches a tenant-scoped entity (PRD §10 acceptance criterion is strict and authoritative).
- Each test must:
  - Insert at least two rows belonging to two distinct `tenant_id` values, with overlapping non-tenant key data (e.g., same character name, same guild id).
  - Invoke the provider with tenant A's context.
  - Assert only tenant A's rows are returned (for reads) or affected (for writes).
- Tests must live next to the provider (`provider_test.go`) and use the project's existing Builder pattern for fixture construction. Do not create `*_testhelpers.go` files (CLAUDE.md Test Helper Pattern).
- Reuse the existing sqlite setup pattern from `libs/atlas-database/tenant_scope_test.go`; lift it into a small shared helper under `libs/atlas-database` if duplication becomes painful, but do not introduce testcontainers.
- Add a regression test for F6: a tenant-scoped `Create` with `TenantId` left zero on the struct must produce a row with `tenant_id = tenant_from_context.Id()`.

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
- The tenant callback already appends `tenant_id` to every applicable WHERE clause; this PRD does not change the runtime cost of those queries. F3/F8 fixes that add `tenant_id` to a child table must include an index on the new column.

### Backward compatibility
- No client-visible behavioral changes for legitimate single-tenant traffic. Any traffic relying on cross-tenant lookups was a bug.

### Migration / rollout
- Single bundled PR. After merge, the PR-overlay environment must run integration smoke tests with two tenants present before the change reaches main.

## 9. Open Questions

Resolved during design (full discussion in `design.md` §11):

- **Cross-tenant queries that are legitimate** — known sites: atlas-merchant (3), atlas-data (5), atlas-saga-orchestrator (2). The audit verifies each existing `WithoutTenantFilter` scope and confirms no new cross-tenant API is needed unless a new gap is found.
- **Asset expiration / monster id allocator interactions** — Postgres-side is covered by the existing callback assuming entities have `tenant_id` and calls use `WithContext`. Audit verifies the Postgres side. Redis-side is explicitly non-goal (see §2).
- **Testcontainers helper** — none exists in the repo. This task uses the existing in-memory sqlite pattern (see §4.3) rather than introducing testcontainers.

## 10. Acceptance Criteria

- [ ] `audit.md` exists in this task folder, lists every GORM call site across all `services/atlas-*` Go services, and classifies each (PASS-CB / PASS-EXPLICIT / PASS-CROSS-TENANT / LEAK-F<n> / UNCLEAR) with file:line evidence and fix status.
- [ ] Every leak classified in the audit is fixed in a single PR on branch `task-041-tenant-filter-leaks`. This includes the F6 hardening of `tenantCreateCallback` (warn → inject) bundled into the same PR.
- [ ] Regression tests exist for atlas-guilds (`getAll`, `getById`, `getForName`) and atlas-character (`getById`, `getForAccount`, `getForAccountInWorld`, `getForName`, `getAll`), plus at least one read and one write provider per service that touches a tenant-scoped entity. PRD §4.3 strict interpretation.
- [ ] Regression tests are written against in-memory sqlite (per `libs/atlas-database/tenant_scope_test.go`), use two-tenant fixtures with overlapping IDs, and assert tenant isolation for both reads and writes.
- [ ] An F6 regression test exists: a tenant-scoped `Create` with `TenantId` left zero on the struct produces a row whose stored `tenant_id` matches the context's tenant.
- [ ] `go test -race ./...` passes in every changed module.
- [ ] `go vet ./...` passes in every changed module.
- [ ] `go build ./...` passes in every changed service.
- [ ] `docker build -f services/<svc>/Dockerfile .` passes for every service whose go.mod or Dockerfile is touched.
- [ ] PR description summarizes the audit results, lists fixed call sites by F-class, calls out the F6 callback change explicitly, and links to `audit.md`.
