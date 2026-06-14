# Context — task-095 version-scoped-canonical-fallback

## What & why (one line)
Make atlas-data's canonical/baseline fallback version-aware by keying canonical rows under a
deterministic per-`(region,major,minor)` synthetic tenant id instead of the single all-zeros sentinel.

## Authoritative pinned facts (verified in code, 2026-06-14)

- **tenant lib** (`libs/atlas-tenant/`):
  - `func Create(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) (Model, error)` — `processor.go:30`
  - `func (m *Model) MajorVersion() uint16` / `MinorVersion() uint16` — `tenant.go:25,29`; `Region() string` exists.
- **worker Params** (`data/workers/worker.go:14`): `ScopeKey string`, `Region string`, `MajorVersion uint16`, `MinorVersion uint16`, `ScratchDir string`.
  → `canonical.TenantId(region string, major, minor uint16)` matches Params and tenant getters exactly.
- **`canonical.TenantUUID`** (`canonical/canonical.go:11`) = `"00000000-0000-0000-0000-000000000000"` = `uuid.Nil`. Keep the const (legacy-refusal + migration source); stop using it as a write/fallback key.
- **Publish signature** (`baseline/publish.go:38`): `Publish(ctx, region string, major, minor int)` — note `int`, so calls into `canonical.TenantId` need `uint16(major)`, `uint16(minor)`.
- **Two read-fallback resolvers only** (OQ-5 resolved):
  - `document/storage.go` — `ByIdProvider` (~44) and `AllProvider` (~85), both `tenant.Create(uuid.Nil, …)`.
  - `searchindex/searchindex.go:ResolveTenantId` (~91) `return uuid.Nil, nil` — covers all 5 search-index tables (monster/npc/reactor/map/item all route through it).
- **Other sentinel sites:** ingest `data/workers/runtime.go:tenantFromParams` (~39); publish `baseline/publish.go:runCopyOut` (~130); status `data/status.go:resolveStatusTenantId` (~127); purge `tenantpurge/purge.go:Purge` (~32). `document/db_storage.go` write path needs **no change**; `baseline/restore.go` writes to caller `target`, **no change**.
- **Purge wiring:** only caller is `tenantpurge/handler.go:45 purgeInner`; it has the path `{id}` AND the request tenant via `r.Context()`. → do the canonical-id refusal handler-side; keep `Purge`'s all-zeros guard as defense-in-depth.

## Critical dependencies / constraints

1. **`ORDER BY id` PR (`fix/baseline-publish-order-by-id`, separate, user-owned) must land first.** This
   worktree is branched off `main` *before* that PR, so `baseline/publish.go` here still has the old
   `runCopyOut` with hardcoded `ORDER BY id` and `WHERE tenant_id = '<all-zeros>'`. **Rebase this branch
   onto `main` after that PR merges**, then implement C5 against the post-fix shape
   (`copyOutSQL`/`orderColumn`): thread `(region,major,minor)` into `copyOutSQL` and swap the WHERE id to
   `canonical.TenantId(...)`. If executing before the rebase, implement C5 against the current
   `runCopyOut` and re-reconcile after rebase.
2. **Tests use in-memory sqlite** (`document/storage_test.go:57`, `searchindex/searchindex_test.go:33`).
   Postgres-only `COPY … (FORMAT binary)` (publish/restore) **cannot** run under sqlite. So:
   - Fallback behavior (documents + search-index) IS unit-testable on sqlite.
   - `canonical.TenantId` / `IsCanonical` are pure-Go unit tests.
   - Publish version-keying is asserted at the **SQL-string** level (the WHERE uses the canonical id),
     not via a live COPY.
   - The publish→restore **round-trip** (FR-4.3) is an **operational/integration** verification on a real
     PG (folded into the migration runbook), not a sqlite unit test.
3. **`Namespace` constant is immutable forever** — changing it orphans every canonical row in every env.
   The determinism unit test pins it.
4. **No `go.mod` change** expected (`uuid`, `fmt` already deps) → `docker buildx bake` not required unless
   `go.mod` ends up touched.

## Decisions locked
- Version-derived canonical UUIDv5; string form `canonical:<region>:<major>.<minor>` (includes minor).
- Migration = re-ingest per version, then delete all-zeros rows; operator runbook, no new persistent code.
- All six live versions: GMS 83.1/84.1/87.1/92.1/95.1, JMS 185.1.
- Purge guard generalized handler-side via `canonical.IsCanonical(id, reqRegion, reqMajor, reqMinor)`.

## Out of scope
ORDER BY id fix (separate PR), per-tenant ingest, MinIO layout, task-084 socket/config, new game content,
atlas-tenants API version filtering.
