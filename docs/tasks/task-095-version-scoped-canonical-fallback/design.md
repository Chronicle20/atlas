# Version-Scoped Canonical Baseline Fallback — Design

Task: task-095-version-scoped-canonical-fallback
Status: Draft
Created: 2026-06-14
PRD: ./prd.md
---

## 1. Problem recap

Canonical (shared) game data in `atlas-data` is stored under one sentinel tenant id
`00000000-0000-0000-0000-000000000000` (`canonical.TenantUUID`, which is also `uuid.Nil`), and the
`documents` / `*_search_index` tables carry **no version dimension**. So the canonical store holds only one
version at a time, and the two read-fallback sites resolve every version to that same partition. A tenant
with no per-tenant rows (e.g. live GMS v84 `4936dff2…`) is served whatever single version currently occupies
canonical. The intended behavior is: *fall back to the canonical/baseline dataset **for the requesting
tenant's own version***.

## 2. Chosen approach: version-derived canonical tenant id

Keep the table schemas exactly as they are and make the **value** of `tenant_id` carry the version, by
deriving a deterministic synthetic canonical id per `(region, major, minor)` and using it everywhere the
all-zeros sentinel is used today. This reuses the existing tenant-scoping query machinery (every read already
filters by `tenant_id`), needs no DDL, no new columns, no composite-index changes, and no query rewrites —
only a change to *which id* the canonical paths compute.

### 2.1 Why not the alternatives

| Alternative | Why rejected |
|---|---|
| **Add `region`/`major`/`minor` columns** to `documents` + 5 search-index tables | Invasive: alter the `(tenant_id, type, document_id)` and `(tenant_id, <entity>_id)` unique indexes, thread version through every read/write/COPY/restore, larger and riskier migration. No benefit over a version-keyed id since canonical is the only multi-version-in-one-table case. |
| **Register per-version canonical as real tenants** in atlas-tenants | Pollutes the tenant registry, entangles canonical lifecycle with tenant CRUD/Kafka, and the purge/guard story gets worse. Canonical is explicitly "never a real tenant" (`canonical` package doc). |
| **Keep all-zeros + a separate `canonical_versions` mapping table** | Adds a join/lookup and a second source of truth for something a pure function already determines. |

## 3. The canonical id function (single source of truth)

Extend the `canonical` package (`services/atlas-data/atlas.com/data/canonical/canonical.go`):

```go
// Namespace anchors all version-derived canonical ids. Deterministic (UUIDv5 of
// the URL namespace over a fixed string) so there is no magic literal to guard,
// yet it is stable forever. MUST NOT change once any canonical rows exist —
// changing it orphans every canonical row in every environment.
var Namespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://atlas-data/canonical"))

// TenantId returns the deterministic canonical tenant id for a (region, major,
// minor). UUIDv5, so: stable across pods/restarts, distinct per version, never
// uuid.Nil, and collision-free against random v4 tenant ids.
func TenantId(region string, major, minor uint16) uuid.UUID {
    return uuid.NewSHA1(Namespace, []byte(fmt.Sprintf("canonical:%s:%d.%d", region, major, minor)))
}

// IsCanonical reports whether id is the canonical id for the given coordinates.
// Used by the purge guard to refuse destroying a canonical dataset.
func IsCanonical(id uuid.UUID, region string, major, minor uint16) bool {
    return id == TenantId(region, major, minor)
}
```

Notes:
- `TenantUUID` (the all-zeros literal) is retained **only** as the migration source / legacy refusal value;
  it is no longer written to after migration (FR-1.3). Keep the constant; stop using it as a write key.
- Exact integer types (`uint16` vs `byte`) follow `tenant.Model.MajorVersion()/MinorVersion()`; the plan
  pins them against the `atlas-tenant` lib signatures. The canonical **string form** (`canonical:<region>:<major>.<minor>`)
  is the contract — it must include minor (live data is uniformly `.1`, but encoding it keeps versions like a
  future `.2` distinct). **OQ-1 resolved.**

## 4. Change sites (complete enumeration)

The sentinel surface is small and fully enumerated (grep for `canonical.TenantUUID` and the `uuid.Nil`
canonical fallbacks):

| # | File:func | Today | Change |
|---|---|---|---|
| C1 | `document/storage.go:ByIdProvider` (~44) | `tenant.Create(uuid.Nil, t.Region(), t.Major(), t.Minor())` | `tenant.Create(canonical.TenantId(t.Region(), t.Major(), t.Minor()), t.Region(), t.Major(), t.Minor())` |
| C2 | `document/storage.go:AllProvider` (~85) | same `uuid.Nil` fallback | same change as C1 (batch path) |
| C3 | `searchindex/searchindex.go:ResolveTenantId` (~91) | `return uuid.Nil, nil` | `return canonical.TenantId(t.Region(), t.Major(), t.Minor()), nil` |
| C4 | `data/workers/runtime.go:tenantFromParams` (~39) | `uuid.Parse(canonical.TenantUUID)` for `scope=shared` | `id = canonical.TenantId(p.Region, p.MajorVersion, p.MinorVersion)` |
| C5 | `baseline/publish.go:runCopyOut` (~130) | `WHERE tenant_id = '<all-zeros>'` | thread `(region,major,minor)` into the COPY; `WHERE tenant_id = '<canonical.TenantId(...)>'` |
| C6 | `data/status.go:resolveStatusTenantId` (~127) | `return canonical.TenantUUID` for `scope=shared` | `return canonical.TenantId(t.Region(), t.Major(), t.Minor()).String()` |
| C7 | `tenantpurge/purge.go:Purge` (~32) | refuse iff `id == canonical.TenantUUID` | refuse legacy all-zeros **and** any version-derived canonical id (see §4.1) |

The canonical write path (`document/db_storage.go`) already keys on the context tenant id; it needs **no
change** — C4 makes the shared-ingest context carry the version-derived id, and the Add/upsert follows.
`baseline/restore.go` writes into the caller-supplied `target` and is **unchanged** (verified by the
round-trip test, FR-4.3).

### 4.1 Purge guard generalization (C7)

Version-derived canonical ids are not self-identifying (a v5 id reveals nothing without its coordinates), so
the guard can't compare against a single constant. `Purge` is invoked from `DELETE /api/data/tenants/{id}`
where the request carries the tenant context (region/major/minor). The guard becomes:

```go
if tenantID.String() == canonical.TenantUUID || canonical.IsCanonical(tenantID, reqRegion, reqMajor, reqMinor) {
    return ErrCanonicalRefused
}
```

The handler passes the requesting tenant's coordinates (already in context) into `Purge`. This refuses an
operator who targets a version's canonical id with matching headers, while normal per-tenant purges (real v4
id ≠ canonical id) are unaffected. The plan pins whether to extend `Purge`'s signature or resolve
coordinates inside the handler before the call.

## 5. Behavior after change

- **Read fallback (C1–C3):** a tenant with no per-tenant rows now queries `canonical.TenantId(its region,
  major, minor)`. If that version's canonical dataset is absent, the query returns empty/not-found exactly as
  before (FR-2.2) — no cross-version bleed, no new error class. Single and batch use the same id, so they
  agree (FR-2.4, closes the PR #759 class).
- **Shared ingest (C4):** `POST /api/data/process?scope=shared` for version X writes canonical rows under
  `TenantId(X)`. Re-running for version Y touches only `TenantId(Y)` (FR-3.2).
- **Publish (C5):** the dump for `(region,major,minor)` contains exactly that version's canonical rows,
  matching its already version-keyed MinIO object path. (Builds on the separate `ORDER BY id` fix.)
- **Status (C6):** `scope=shared` status reports the canonical counts for the caller's version.

## 6. Migration & cutover

Approach (confirmed): re-ingest, then drop legacy rows. **OQ-2 / OQ-3 resolved** as an operator runbook, not
new persistent code.

Ordering per environment (provision-before-delete, FR-5.3 — avoids an empty-canonical window):

1. Deploy the new atlas-data image (the code now reads/writes version-derived canonical ids; existing
   per-tenant rows are untouched; tenants with per-tenant data are unaffected throughout).
2. For each live version, run `POST /api/data/process?scope=shared` (operator-gated) with that version's
   region/major/minor headers → populates `TenantId(version)`.
3. Verify each version's canonical dataset is populated (`GET /api/data/status?scope=shared` per version, and
   a spot read from a no-per-tenant-data tenant of that version).
4. Re-publish baselines per version (`POST /api/data/baseline/publish`) so the version-correct dump + sha256
   sidecar exist (this also un-breaks ephemeral `auto`-mode bootstrap).
5. Delete the legacy all-zeros rows — idempotent, safe to re-run:
   `DELETE FROM <each table> WHERE tenant_id = '00000000-0000-0000-0000-000000000000';`
   Recommend a documented `psql` step (one-time per env) over new code; the tables are the
   `tenantpurge.PurgeTables` set minus `tenant_baselines` (canonical never had a baseline row).

**OQ-4 resolved:** `atlas-pr-bootstrap` needs **no code change**. Its probe and restore already version-key
the MinIO path (`baseline/regions/<R>/versions/<M>.<m>/…`); once step 4 publishes per-version baselines,
`auto` mode restores the correct version for free. Fresh PR-env tenants restore into a real target tenant
(so they hold per-tenant rows and never hit the fallback); the fallback fix matters for tenants provisioned
without ingest/restore.

**OQ-5 resolved:** there are exactly two read-fallback resolvers — `document/storage.go` (C1/C2) and
`searchindex.ResolveTenantId` (C3). All search-index consumers (monster/npc/reactor/map/item) route through
`ResolveTenantId`, so C3 covers all five tables; no per-resource read path bypasses it.

## 7. Test strategy

- **Unit (`canonical`):** `TenantId` is deterministic, distinct per `(region,major,minor)`, never `uuid.Nil`,
  and differs from `TenantUUID`; `IsCanonical` true only for the matching coordinates.
- **Document storage (`document`):** with two versions' canonical rows present under their derived ids, a
  no-per-tenant-data tenant of version A reads A's data; a version-B tenant reads B's data; a tenant *with*
  per-tenant rows still reads its own; single (`ById`) and batch (`All`) agree. Extends the existing storage
  tests (which already drive a test DB).
- **Search index (`searchindex`):** update `TestSearch_SinglePartition_ZeroRowTenantFallsBack` and add a
  multi-version variant proving `ResolveTenantId` returns the version-derived canonical id and search results
  are version-correct.
- **Ingest (`data/workers`):** `tenantFromParams` for `scope=shared` yields `TenantId(p…)`, not all-zeros.
- **Baseline (`baseline`):** publish→restore round-trip per version (publish v84 → restore into a fresh
  tenant → reads return v84 data); publish writes the version-derived `WHERE`.
- **Purge (`tenantpurge`):** refuses both the legacy all-zeros id and a version-derived canonical id;
  allows a normal tenant id.

Verification gates (CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-data;
`tools/redis-key-guard.sh` clean; `docker buildx bake atlas-data` only if `go.mod` changes (none expected —
`uuid`/`fmt` are already deps).

## 8. Risks & mitigations

| Risk | Mitigation |
|---|---|
| A fallback site missed → silent version bleed | The five canonical-read/write sites are fully enumerated in §4; a `grep` for `canonical.TenantUUID` and `uuid.Nil`-as-canonical is part of the plan's done-check. |
| `Namespace` constant changed later → orphans all canonical rows | Document prominently in code; it is derived from a fixed string and must never change. Covered by the determinism unit test (a value change breaks it loudly). |
| Empty-canonical window during cutover | Provision-before-delete ordering (§6); per-tenant tenants are never affected. |
| Purge guard gap (multiple canonical ids) | Generalized guard via `IsCanonical` with request coordinates (§4.1). |
| Operator forgets a version | FR-6 checklist + step-3 verification per version; all six live versions enumerated. |

## 9. Out of scope (carried from PRD)

`ORDER BY id` publish crash (separate PR, assumed landed); per-tenant ingest semantics; MinIO layout;
task-084 socket/config provisioning; new game content; atlas-tenants API version filtering.
