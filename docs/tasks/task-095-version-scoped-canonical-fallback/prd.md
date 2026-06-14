# Version-Scoped Canonical Baseline Fallback (atlas-data) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-14
---

## 1. Overview

In `atlas-data`, game data is stored as `documents` (and five `*_search_index` tables), tenant-scoped
by `tenant_id`. A tenant (`region` + `majorVersion` + `minorVersion`) may have its **own** per-tenant
dataset, or — when it has none — fall back to a shared **canonical** dataset. The canonical dataset is
the source for ephemeral-PR-env bootstrap (fast baseline restore) and for any version-tenant that was
provisioned without a full per-tenant ingest.

The intended behavior is: *each tenant can have its own dataset, but a tenant with no per-tenant data
must fall back to the canonical/baseline dataset **for its own version**.* The current implementation
cannot honor the "for its own version" half. All canonical data is written under a **single** synthetic
tenant UUID `00000000-0000-0000-0000-000000000000` (`canonical/canonical.go`), and the `documents` table
is keyed `(tenant_id, type, document_id)` with **no `region`/`version` dimension** (`document/entity.go`).
The read-time fallback (`document/storage.go`) builds the canonical lookup carrying the requester's
version but then filters on `tenant_id` only — so the version is silently discarded.

Consequences, confirmed during investigation (2026-06-14):

- The canonical store physically holds **only one version at a time**. A second `SCOPE=shared` ingest for
  a different version upserts the first away (conflict key is the three columns above; `db_storage.go`).
- A version-tenant with no per-tenant rows (live example: GMS v84 tenant `4936dff2…`, which has **no**
  per-tenant WZ copy in MinIO) falls back to canonical and is served **whatever single version currently
  occupies the canonical tenant** — not guaranteed to be its own version. This is the same root cause
  behind the earlier "v84 batch GetAll returned `{data:[]}`" incident (PR #759).
- Because there is no version column, you cannot even tell from SQL which version canonical currently
  holds — the absence of the dimension *is* the bug.

This task adds a version dimension to canonical storage so the fallback resolves per-version, and aligns
the `SCOPE=shared` ingest and the baseline publish/restore paths to the same per-version keying.

### Relationship to other work

- **task-084 (multi-version provisioning)** solved coexistence of N versions at the **socket/config**
  layer (per-version ports, additive `services` config, per-tenant consumers). It did **not** touch the
  atlas-data canonical *document* store. This task is the data-layer complement.
- **`fix/baseline-publish-order-by-id` (separate PR, owned by the user)** fixes the `ORDER BY id` crash in
  `baseline/publish.go` that currently 500s every publish. This task **assumes that PR has landed**; the
  version-keying here is what makes publish/restore's existing `region/major/minor` parameters actually
  meaningful.
- **task-071 (gamedata MinIO consolidation)** introduced `SCOPE=shared|tenants/<id>` and the
  version-keyed MinIO path layout (`<scope>/regions/<R>/versions/<M>.<m>/…`). MinIO is already
  version-correct; only the **DB** keying is version-blind.

## 2. Goals

Primary goals:

- Canonical data is stored per `(region, majorVersion, minorVersion)` so multiple versions' canonical
  datasets coexist in the DB without collision.
- A tenant with no per-tenant rows for a document type falls back to the canonical dataset **for its own
  region+version**, for both single-id reads and batch reads, across `documents` and all five
  `*_search_index` tables.
- `SCOPE=shared` ingest writes canonical data under the version-derived canonical id for the version it
  is ingesting.
- `baseline/publish` dumps exactly one version's canonical rows (matching its already version-keyed MinIO
  object path); `baseline/restore` is unaffected in target-tenant behavior but is verified to interoperate
  with the new dumps.
- Existing single-UUID canonical data is migrated to the new scheme with no orphaned/ambiguous rows left
  behind.
- All six live versions (GMS v83.1, v84.1, v87.1, v92.1, v95.1; JMS v185.1) have a populated per-version
  canonical dataset by the end of the task.

Non-goals:

- The `ORDER BY id` publish crash fix (separate PR).
- Any change to per-tenant dataset semantics or the per-tenant ingest path (`SCOPE=tenants/<id>`).
- Any change to MinIO bucket names or the version-keyed object path layout (already correct).
- Socket/config multi-version provisioning (task-084).
- Adding new game-version *content* for any version.
- Region/version filtering on the `atlas-tenants` REST API.

## 3. User Stories

- As a **platform operator**, I want a newly provisioned version-tenant with no per-tenant data to serve
  that version's canonical game data, so clients of that version see correct items/mobs/maps instead of
  another version's data.
- As a **platform operator**, I want to run `SCOPE=shared` ingest for several versions into one
  environment without each run clobbering the previous version's canonical data.
- As an **ephemeral-PR-env owner**, I want `auto`-mode bootstrap to restore the correct per-version
  canonical baseline, so a PR env on any supported version is correct without a full per-tenant ingest.
- As a **developer**, I want canonical data identifiable by version in storage, so the "which version is
  this canonical row?" question is answerable and testable.

## 4. Functional Requirements

### FR-1 Version-derived canonical identity
- **FR-1.1** Introduce a deterministic canonical tenant id derived from `(region, majorVersion,
  minorVersion)` — a UUIDv5 over a fixed namespace and a canonical string form (e.g.
  `canonical:<region>:<major>.<minor>`). Same inputs always yield the same id; different versions yield
  different ids.
- **FR-1.2** The derivation is defined in exactly one place (extend `canonical/`), consumed by the read
  fallback, the ingest worker, and baseline publish. No second copy of the formula.
- **FR-1.3** The legacy all-zeros canonical UUID is retired as a storage key. If retained at all, it is
  only as a migration source, never written to after migration.

### FR-2 Version-aware read fallback
- **FR-2.1** When a tenant read finds no per-tenant rows, the fallback queries the canonical dataset for
  the **requesting tenant's** `(region, major, minor)` via the FR-1 id — replacing the current
  `uuid.Nil`-only lookup in `document/storage.go` (single-id `ById`/`ByIdProvider` and batch
  `All`/`AllProvider`).
- **FR-2.2** If the canonical dataset for the requester's version is absent, the read returns the same
  empty/not-found result the current code would for a missing canonical row (no new error class, no
  cross-version bleed).
- **FR-2.3** The fallback behavior is identical in shape for `documents` and all five `*_search_index`
  tables (monster/npc/reactor/map/item_string).
- **FR-2.4** Batch (`?ids=`) and single (`/id`) reads return consistent results for the same tenant —
  closing the divergence class from PR #759.

### FR-3 Version-scoped shared ingest
- **FR-3.1** When ingest runs with `SCOPE=shared`, documents and search-index rows are written under the
  FR-1 version-derived canonical id for `(region, MAJOR_VERSION, MINOR_VERSION)`, not the all-zeros UUID.
- **FR-3.2** Re-running `SCOPE=shared` ingest for a different version in the same environment does not
  modify another version's canonical rows.
- **FR-3.3** The `tenants/<id>` (per-tenant) ingest path is unchanged.

### FR-4 Version-scoped baseline publish/restore
- **FR-4.1** `baseline/publish` dumps only the canonical rows for the requested `(region, major, minor)` —
  i.e. the `COPY … WHERE tenant_id = <FR-1 id>` filter, not the all-zeros UUID, so the produced
  `documents.dump` content matches its version-keyed MinIO path.
- **FR-4.2** `baseline/restore` continues to write into the caller-supplied `target` tenant and remains
  correct against dumps produced under FR-4.1 (sha256 + schema gates unchanged).
- **FR-4.3** Restore/publish round-trip is verified per version (publish v84 → restore into a fresh
  tenant → reads return v84 data).

### FR-5 Migration of existing canonical data
- **FR-5.1** A migration re-provisions per-version canonical datasets. Per the chosen approach
  (re-ingest), this means running `SCOPE=shared` ingest for each live version into its FR-1 id, then
  deleting the legacy all-zeros canonical rows.
- **FR-5.2** No legacy all-zeros canonical rows remain after migration in any environment that has been
  migrated.
- **FR-5.3** Migration is documented as a runbook step (operator-run), ordered so that no version-tenant
  is left falling back to an empty canonical dataset during the cutover (provision new canonical before
  deleting old).

### FR-6 Coverage
- **FR-6.1** All six live versions (GMS 83.1/84.1/87.1/92.1/95.1, JMS 185.1) have a populated per-version
  canonical dataset and a published baseline by task completion.

## 5. API Surface

No new endpoints. Behavior changes on existing endpoints:

- `POST /api/data/baseline/publish` (`{region, majorVersion, minorVersion}`, operator-gated): now snapshots
  exactly the canonical rows for that version (FR-4.1). Response shape unchanged (`{sha256}`, 202).
- `POST /api/data/baseline/restore`: unchanged contract; verified against version-scoped dumps (FR-4.2).
- `POST /api/data/process` (`?scope=shared`, operator-gated): now writes canonical rows under the
  version-derived id (FR-3.1). Default `scope=tenants/<id>` unchanged.
- All tenant-scoped data read endpoints (`/api/data/...`, single + `?ids=` batch): unchanged contract;
  fallback now returns the requester's-version canonical data (FR-2). No request/response shape change.

## 6. Data Model

- `documents`: `(tenant_id, type, document_id)` unique key — **unchanged**. The version dimension is
  carried by the **value** of `tenant_id` (a version-derived synthetic id for canonical rows), not by a
  new column. Per-tenant rows keep their real tenant UUID.
- `*_search_index` (monster/npc/reactor/map/item_string): keyed `(tenant_id, <entity>_id)` — **unchanged**,
  same approach.
- `tenant_baselines`: existing table (`baseline/migration.go`), keyed by `tenant_id` — unchanged.
- **Migration:** delete-and-reingest of canonical rows (FR-5). No DDL/column additions expected under the
  version-derived-UUID approach; confirm during design whether any index assumes the all-zeros constant.
- **Decision (confirmed):** version-derived canonical UUID over adding `region/major/minor` columns —
  minimal blast radius, reuses the existing tenant-scoping query machinery.

## 7. Service Impact

`atlas-data` only:
- `canonical/` — add the version-derived id helper (single source of truth).
- `document/storage.go` — version-aware fallback for single + batch (the core change).
- `document/db_storage.go` — verify the write path keys on the supplied tenant id (it already does);
  ensure shared-ingest supplies the version-derived id.
- `data/workers/runtime.go` (+ ingest run path) — `SCOPE=shared` resolves to the version-derived id.
- `baseline/publish.go` — `WHERE tenant_id =` uses the version-derived id.
- `baseline/restore.go` — verified interop; likely unchanged.
- search-index packages (monster/npc/reactor/map/item) — confirm reads route through the version-aware
  fallback consistently.

Operational (no code): re-provision canonical per version; republish baselines; update the
ephemeral-PR-deployments runbook.

## 8. Non-Functional Requirements

- **Multi-tenancy:** canonical rows remain isolated by `tenant_id` value; no cross-version or cross-tenant
  bleed. The synthetic canonical id must not collide with any real tenant id (UUIDv5 over a dedicated
  namespace guarantees this in practice; document the namespace).
- **Determinism:** the canonical id for a given `(region, major, minor)` is stable across pods, restarts,
  and re-derivation — required so ingest, fallback, and publish agree without coordination.
- **Observability:** logs/metrics around fallback should make it possible to tell that a read fell back to
  canonical and for which version; publish/ingest logs already include `region`/`ms.version`.
- **Performance:** fallback adds no extra round-trips beyond today's (one per-tenant miss → one canonical
  query); the canonical query is the same shape, just a different `tenant_id` value.
- **Backward compatibility / cutover:** during migration, provision new per-version canonical before
  deleting legacy rows so no tenant serves empty data mid-cutover (FR-5.3).
- **Verification (project rules):** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
  atlas-data; `docker buildx bake atlas-data` if `go.mod` is touched; `tools/redis-key-guard.sh` clean.

## 9. Open Questions

- **OQ-1 (UUIDv5 namespace):** which namespace UUID + exact canonical string form? (Design to pin a
  constant and document it.) Should `minorVersion` always participate, or only `region`+`major`? (Live
  data uses `.1` minors uniformly; recommend including minor for completeness.)
- **OQ-2 (Migration mechanics):** is re-ingest driven by the existing `POST /api/data/process?scope=shared`
  per version, or a one-shot migration job? Who runs it per environment (atlas-main vs each PR env)?
- **OQ-3 (Legacy row cleanup):** delete all-zeros rows via SQL migration, an operator script, or a guarded
  endpoint? Must be safe to run repeatedly.
- **OQ-4 (PR-env bootstrap):** does `atlas-pr-bootstrap` need any change, or does version-correct
  publish/restore make it correct for free once baselines exist per version?
- **OQ-5 (search-index read paths):** confirm every search-index consumer routes through the same
  version-aware fallback (some may query directly).

## 10. Acceptance Criteria

- [ ] A single, documented version-derived canonical id helper exists in `canonical/` and is the only
      definition of the formula.
- [ ] Two versions' canonical datasets coexist in one DB without collision (e.g. v83 + v84 canonical rows
      both present and distinct).
- [ ] A tenant with no per-tenant rows reads its **own version's** canonical data; a different-version
      tenant reads different canonical data — verified by test for `documents` and all five search-index
      tables, single and batch.
- [ ] Batch (`?ids=`) and single (`/id`) reads agree for the same tenant (PR #759 class closed).
- [ ] `SCOPE=shared` ingest writes under the version-derived id; re-ingesting a second version leaves the
      first untouched.
- [ ] `baseline/publish` for version X produces a dump containing only X's canonical rows; publish→restore
      round-trip yields X's data in a fresh tenant.
- [ ] No legacy all-zeros canonical rows remain after migration.
- [ ] All six live versions have a populated per-version canonical dataset and a published baseline.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-data; redis-key-guard clean;
      `docker buildx bake atlas-data` clean if `go.mod` changed.
- [ ] Operator runbook updated with the per-version canonical provisioning + legacy-row cleanup steps.
