# Backend Audit — fix/1213-spawn-index-canonical-fallback (diff-scoped)

- **Service:** atlas-data
- **Scope:** Single commit `cae4f99b8` — `git diff main...HEAD`, 3 Go files + 2 new test files + docs
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS (`go build ./...` in `services/atlas-data/atlas.com/data`)
- **Tests:** PASS — full package list green, including `atlas-data/monster`, `atlas-data/npc`, `atlas-data/searchindex`
- **Overall:** NEEDS-WORK (one Important structural finding; no correctness/isolation defect found)

## Build & Test Results

```
$ go build ./...          # clean, no output
$ go test ./... -count=1  # all packages ok, including monster, npc, searchindex
```

## Changed-file inventory

| File | Nature of change |
|---|---|
| `services/atlas-data/atlas.com/data/searchindex/searchindex.go` | Extracts `ResolvePartitionTenantId[E]` (generic); old `ResolveTenantId` now a 1-line delegate |
| `services/atlas-data/atlas.com/data/monster/resource.go` | `handleGetMonsterMapsRequest`: tenant guard simplified to discard-tenant form, adds `ResolvePartitionTenantId` call, query now runs under `database.WithoutTenantFilter` filtered by `partition` |
| `services/atlas-data/atlas.com/data/npc/resource.go` | `handleGetNpcMapsRequest`: adds `ResolvePartitionTenantId` call ahead of the existing query, same `WithoutTenantFilter` + `partition` change |
| `services/atlas-data/atlas.com/data/monster/spawn_index_fallback_test.go` (new) | httptest-level fallback test, 2 subtests |
| `services/atlas-data/atlas.com/data/npc/spawn_index_fallback_test.go` (new) | httptest-level fallback test, 2 subtests |
| `services/atlas-data/docs/rest.md`, `docs/storage.md` | Doc updates describing the new partition-resolution behavior |

## Item-by-item review (per the three areas flagged for scrutiny)

### 1. Multi-tenancy isolation risk of `database.WithoutTenantFilter` + explicit `tenant_id = ?`

**Verdict: PASS — no cross-tenant leak.**

- `ResolvePartitionTenantId` (`searchindex/searchindex.go:97-115`) computes `partition` as either (a) the caller's own `t.Id()` — only if an `EXISTS`-style probe against `t.Id()` on E's own table returns a row (line 105: `Where("tenant_id = ?", t.Id())`), or (b) `canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion())` — a value derived entirely from the *calling* tenant's own region/version, not from any request-supplied value.
- Both call sites (`monster/resource.go:219-224`, `npc/resource.go:199-201`) then run the actual data query filtered by `Where("tenant_id = ? AND monster_id = ?", partition, monsterId)` / `..., npc_id = ?`. There is no code path where `partition` can become an arbitrary attacker-chosen UUID — it is one of exactly two values, both derived from the requester's own validated tenant context.
- The `TenantWithOwnRowsDoesNotReadCanonical` subtests (`monster/spawn_index_fallback_test.go:60-86`, `npc/spawn_index_fallback_test.go:65-90`) directly assert the negative case — a tenant with its own rows does not see the canonical/other-tenant rows — which is the isolation property that matters here.
- Net effect: a request can see (a) its own data, or (b) the version-scoped canonical/shared data every tenant on that region+version is intended to share. This matches the documented `ResolveTenantId` partition semantics already in production for `map_search_index`/`npc`/`monster`/`item-strings` search (`docs/rest.md:57`, unchanged by this diff) — the fix generalizes an existing, already-reviewed partitioning contract rather than inventing a new one.

### 2. Tenant-guard ordering before `ResolvePartitionTenantId` (which panics via `tenant.MustFromContext` on a tenant-less request)

**Verdict: PASS for both handlers — guard precedes the call in both.**

- `monster/resource.go:200-206`: explicit guard added by this diff —
  ```go
  // Guard before ResolvePartitionTenantId, which resolves the tenant
  // with MustFromContext and would panic on a tenant-less request.
  if _, terr := tenant.FromContext(d.Context())(); terr != nil {
      d.Logger().WithError(terr).Errorf("Unable to resolve tenant for monster-maps request.")
      server.WriteErrorResponse(d.Logger())(w)(terr)
      return
  }
  ```
  followed by the `ResolvePartitionTenantId` call at `monster/resource.go:211`. Order is correct.
- `npc/resource.go:181-185`: the pre-existing (unchanged by this diff) guard —
  ```go
  t, err := tenant.FromContext(d.Context())()
  if err != nil {
      w.WriteHeader(http.StatusBadRequest)
      return
  }
  ```
  precedes the new `ResolvePartitionTenantId` call at `npc/resource.go:190`. Order is correct; `t` remains live and is used later for the log line at `npc/resource.go:222` (`"tenant_id": t.Id().String()`), so it is not dead code.
- Non-blocking observation (pre-existing, not introduced by this diff): the npc guard fails with a bare `w.WriteHeader(http.StatusBadRequest)` while the monster guard (as touched by this diff) fails via `server.WriteErrorResponse`. This inconsistency predates the commit and was not touched by it, so it is not scored as a diff finding, but it's worth folding into a future pass since the two handlers are now near-identical in shape.

### 3. `ResolvePartitionTenantId` living in package `searchindex`

**Verdict: Minor / non-blocking naming smell — not a hard violation.**

- `searchindex` is an atlas-data-internal shared helper package (not a domain package with its own `model.go`), imported by `monster` and `npc` for a purely infrastructural concern (partition resolution + `Migrate`/`Upsert`/`Search`/`Count` helpers used by every `*_search_index` table). Housing a generic, type-parameterized `ResolvePartitionTenantId[E]` there is consistent with how the package already serves multiple unrelated domain entities (`item`, `map`, `npc`, `monster` search-index rows) — it was already a shared cross-domain utility before this diff, not a domain-owned file.
- That said, the package is literally named `searchindex`, and the function's own doc comment (`searchindex.go:80-96`) has to spend three paragraphs explaining that it *also* now serves `*_spawn_index` tables, which are not search indices. The name no longer describes the package's full contents. This is a readability/discoverability smell, not a layering violation — there is no guideline requiring a rename, and no anti-pattern doc entry against a generic partition-resolution helper living alongside sibling table helpers in the same internal package. Recommend (non-blocking): rename the package/file (e.g., `tenantpartition`) or move `ResolvePartitionTenantId` to a smaller `internal/tenantpartition` package in a follow-up, but this does not block the fix.

## Mechanical Checklist (applicable items only — full DOM-01..28 sweep is out of scope since only 3 functions + 1 new helper changed, not full domain packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 / anti-patterns "Handlers Calling Providers Directly" | Handler business/data-access logic lives in `processor.go`, not `resource.go` | **FAIL (Important)** | `monster/resource.go:219-224` and `npc/resource.go:199-201` run `db.WithContext(database.WithoutTenantFilter(d.Context())).Where(...).Find(&rows)` directly inside the HTTP handler closure, not via `monster.Processor` / `npc.Processor` (both of which exist: `monster/processor.go:14-33`, `npc/processor.go:14-33`) or a `provider.go`. This is the documented anti-pattern (`.claude/skills/backend-dev-guidelines/resources/anti-patterns.md:48-80`, "Critical Layer Violations: Handlers Calling Providers Directly") — worse than the canonical example, since here the handler talks straight to GORM (`entity.go`-equivalent) rather than through even a `provider.go`. The documented "circular dependency" exception (anti-patterns.md:100-124) does not apply: `SpawnIndexEntity` is defined in the same package as the handler (`monster/spawn_index.go:10`, `npc/spawn_index.go:10`), so there is no cross-package cycle forcing a raw query. Sibling handlers in the same files (`handleGetNpcsRequest` at `npc/resource.go:75`, `handleGetNpcRequest` at `npc/resource.go:300`) go through `NewStorage(...)`, showing the abstraction is available and simply wasn't used here. **This pattern pre-dates the commit** (the prior code already ran a raw `db.Where(...).Find()` in the handler) but the diff extends it by adding a second, more sensitive raw-GORM call (`ResolvePartitionTenantId`'s own probe query is out-of-package and fine, but the *handler's* own `Find` now also carries the tenant-bypass context directly at the handler layer instead of being encapsulated in `provider.go`/`processor.go` per `patterns-multitenancy-context.md`'s documented "Cross-Tenant Queries" pattern, which shows the bypass used inside a provider function, not a handler). |
| DOM-14 | Handlers don't call providers/DB directly | **FAIL (Important)** | Same evidence as above. |
| Multi-tenancy: guard ordering vs `MustFromContext` panic | Tenant-presence guard precedes any `MustFromContext` call | PASS | `monster/resource.go:202` (new) before `monster/resource.go:211`; `npc/resource.go:181` (pre-existing) before `npc/resource.go:190` (new). |
| Multi-tenancy: no cross-tenant data exposure via `WithoutTenantFilter` | Resolved partition is derived only from the requester's own validated tenant, never request input | PASS | `searchindex/searchindex.go:98-114`; negative-case tests `monster/spawn_index_fallback_test.go:60-86`, `npc/spawn_index_fallback_test.go:65-90`. |
| DOM-10 | Test DB registers tenant callbacks | PASS | `monster/resource_test.go:97` `database.RegisterTenantCallbacks(logrus.StandardLogger(), db)`, shared by `setupResourceTestDB` which both new fallback tests call (`monster/spawn_index_fallback_test.go:22`, `npc/spawn_index_fallback_test.go:25`). |
| DOM-20 | Table-driven / subtest structure | PASS | Both new test files use `t.Run(...)` subtests exercising the "own rows" vs "no rows → canonical" cases (`monster/spawn_index_fallback_test.go:34,60`; `npc/spawn_index_fallback_test.go:35,64`). |
| DOM-06 | Processor/helper accepts `logrus.FieldLogger`, not `*logrus.Logger` | N/A | `ResolvePartitionTenantId` takes no logger parameter at all; it returns `(uuid.UUID, error)` and lets the caller log (`monster/resource.go:212-216`, `npc/resource.go:191-195` both log via `d.Logger()`). No violation. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | No matches in the changed hunks. |
| DOM-17 | Domain error → HTTP status mapping | PASS (unchanged behavior) | Both new error branches use `server.WriteErrorResponse(d.Logger())(w)(perr)` (`monster/resource.go:213-216`, `npc/resource.go:192-195`), consistent with the pre-existing query-error branches in the same functions. |
| Anti-patterns: bare `go` statements | No new unguarded goroutines | PASS (N/A) | No `go` statements in the diff; `tools/goroutine-guard.sh` output has no hits in `services/atlas-data`. |
| DOM-21 | No duplicate atlas-constants types | PASS (N/A) | No new domain type/enum introduced; `ResolvePartitionTenantId` is generic over the caller's existing entity types. |

## Summary

### Blocking (must fix)
- **FILE-01 / DOM-14** — `handleGetMonsterMapsRequest` (`services/atlas-data/atlas.com/data/monster/resource.go:219-224`) and `handleGetNpcMapsRequest` (`services/atlas-data/atlas.com/data/npc/resource.go:199-201`) run a raw GORM query — including the `database.WithoutTenantFilter` tenant-bypass — directly in `resource.go`, bypassing `processor.go`/`provider.go`. No circular-dependency exception applies (`SpawnIndexEntity` is same-package). Recommend moving the `ResolvePartitionTenantId` + `Find` sequence into a `Processor` method (e.g. `SpawnMapsProvider(monsterId)` / `SpawnMapsProvider(npcId)`) so the handler only calls the processor, matching the pattern already used by sibling handlers in the same files (`npc/resource.go:75`, `npc/resource.go:300`). This is a pre-existing pattern in these two functions that the commit extends rather than originates, but per audit convention prevalence does not exempt it and the added tenant-bypass line raises the stakes of keeping it un-encapsulated.

### Non-Blocking (should fix)
- Package `searchindex` now resolves partitions for tables that are not search indices (`monster_spawn_index`, `npc_spawn_index`); consider a rename or a small dedicated package in a follow-up for discoverability. Not a layering violation.
- `npc/resource.go`'s pre-existing tenant-missing guard (`npc/resource.go:181-185`) returns a bare `w.WriteHeader(http.StatusBadRequest)` instead of `server.WriteErrorResponse`, unlike the guard this diff added to the monster handler. Pre-existing, untouched by this diff, but worth reconciling since the two handlers are now structurally parallel.

### Not found (checked, passed)
- No cross-tenant data leakage: the resolved partition can only be the requester's own tenant id or the version-scoped canonical id derived from the requester's own tenant — verified in code and by the new negative-case tests.
- Guard-before-`MustFromContext` ordering is correct in both handlers.
- Build and full test suite pass.

---

## Resolution (post-audit)

### Blocking finding — FIXED

`ResolvePartitionTenantId` + `Find` (including the `database.WithoutTenantFilter`
bypass) moved out of the handler layer into the owning package's spawn-index
module:

- `monster/spawn_index.go` — new `SpawnMapsFor(db, ctx, monsterId) ([]SpawnIndexEntity, error)`
- `npc/spawn_index.go` — new `SpawnMapsFor(db, ctx, npcId) ([]SpawnIndexEntity, error)`

Both handlers now hold no GORM call and no tenant-bypass context; they call
`SpawnMapsFor` and map the rows to their rest model. The doc comment on
`monster.SpawnMapsFor` records why the partition is resolved and why the read
bypasses the automatic tenant filter, so the invariant travels with the query
rather than with the handler.

Placed in `spawn_index.go` (which already owns the entity, `TableName`, and the
migration) rather than in a new `Processor` method: these packages' `Processor`
is the document-model processor and `SpawnIndexEntity` is not a document, so a
processor method would have crossed two different storage models. The layering
objection — a raw query in `resource.go` — is resolved either way.

Re-verified after the refactor: `go build ./...`, `go vet ./...`,
`go test -race -count=1 ./...` (all packages ok), `tools/lint.sh --check` OK.

### Non-blocking findings — deliberately not addressed

- **`searchindex` package name** — now resolves partitions for the
  `*_spawn_index` tables too. Agreed; a rename touches every search-index caller
  and is out of scope for a bugfix branch.
- **`npc/resource.go` bare `w.WriteHeader(http.StatusBadRequest)`** vs the
  monster handler's `server.WriteErrorResponse` on a tenant-less request.
  Pre-existing and load-bearing: `TestNpcResourceIntegration/GetNpcMaps/MissingTenantReturns400`
  asserts the 400. Reconciling the two would change one endpoint's status code
  contract, which does not belong in this fix.
