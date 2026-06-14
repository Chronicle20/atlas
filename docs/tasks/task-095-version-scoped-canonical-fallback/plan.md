# Implementation Plan — task-095 version-scoped-canonical-fallback

Design: ./design.md · PRD: ./prd.md · Context: ./context.md

All work is in `services/atlas-data/atlas.com/data/` (module `atlas-data`). TDD: write/extend the test
first (red), implement (green). Tasks are ordered bottom-up; T1 unblocks the rest. Paths below are relative
to the atlas-data module root unless noted.

> **Pre-flight (do once before T5):** if `fix/baseline-publish-order-by-id` has merged to `main`, rebase
> this branch onto `main` so `baseline/publish.go` has the `copyOutSQL`/`orderColumn` shape. See context.md
> dependency #1. T1–T4, T6, T7 are independent of that rebase.

---

## T1 — Canonical id helper (single source of truth)

**Files:** `canonical/canonical.go`, new `canonical/canonical_test.go`

**Test first** (`canonical_test.go`):
- `TenantId("GMS",83,1)` is deterministic (two calls equal).
- `TenantId("GMS",83,1) != TenantId("GMS",84,1)` and `!= TenantId("JMS",83,1)`.
- `TenantId(...) != uuid.Nil` and `!= uuid.MustParse(TenantUUID)`.
- `IsCanonical(TenantId("GMS",84,1),"GMS",84,1)` true; `IsCanonical(<that id>,"GMS",83,1)` false;
  `IsCanonical(uuid.New(),"GMS",84,1)` false.
- Determinism pin: `TenantId("GMS",83,1).String()` equals a hardcoded expected string (computed once and
  frozen) so a `Namespace`/format change fails loudly.

**Implement** (`canonical.go`): add imports `fmt`, `github.com/google/uuid`; add:
```go
var Namespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://atlas-data/canonical"))
func TenantId(region string, major, minor uint16) uuid.UUID {
    return uuid.NewSHA1(Namespace, []byte(fmt.Sprintf("canonical:%s:%d.%d", region, major, minor)))
}
func IsCanonical(id uuid.UUID, region string, major, minor uint16) bool {
    return id == TenantId(region, major, minor)
}
```
Keep `TenantUUID` const. Add a prominent comment on `Namespace`: MUST NOT change once canonical rows exist.

**Verify:** `go test ./canonical/`.

---

## T2 — Version-aware document fallback (C1, C2)

**Files:** `document/storage.go`, `document/storage_test.go`

**Test first** (extend `storage_test.go`, sqlite): seed canonical rows under
`canonical.TenantId("GMS",83,1)` and different content under `canonical.TenantId("GMS",84,1)` (insert with
`WithoutTenantFilter` or direct tenant ctx). Assert:
- a v83 tenant with no per-tenant rows reads v83 canonical via `GetById` and `GetAll`;
- a v84 tenant reads v84 canonical (different content) — proves no cross-version bleed;
- a tenant WITH its own per-tenant rows still reads its own (fallback not taken);
- `GetById` and `GetAll` agree for the same tenant (FR-2.4).

**Implement** (`storage.go`): in `ByIdProvider` (~44) and `AllProvider` (~85) replace
`tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())` with
`tenant.Create(canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()), t.Region(), t.MajorVersion(), t.MinorVersion())`.
Add import `atlas-data/canonical`. **Remove the now-unused `github.com/google/uuid` import** (storage.go
used `uuid` only for `uuid.Nil` at those two sites — confirm and drop it, else build fails on unused import).

**Verify:** `go test ./document/`.

---

## T3 — Version-aware search-index fallback (C3)

**Files:** `searchindex/searchindex.go`, `searchindex/searchindex_test.go`

**Test first:** update `TestSearch_SinglePartition_ZeroRowTenantFallsBack` to expect the fallback partition
is `canonical.TenantId(region,major,minor)` (not `uuid.Nil`); add a multi-version variant where v83 and v84
canonical partitions hold different rows and a zero-row tenant of each version searches its own version's
canonical data.

**Implement:** in `ResolveTenantId` (~91) change `return uuid.Nil, nil` to
`return canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()), nil`. Add import
`atlas-data/canonical`. Keep the `uuid` import (still used at the error return ~86). Confirm no import cycle
(`canonical` imports only `fmt`+`uuid`).

**Verify:** `go test ./searchindex/`.

---

## T4 — Version-scoped shared ingest (C4)

**Files:** `data/workers/runtime.go`, test (extend existing workers test or add `runtime_test.go`)

**Test first:** `tenantFromParams(Params{ScopeKey:"shared", Region:"GMS", MajorVersion:84, MinorVersion:1})`
returns a `tenant.Model` whose `Id() == canonical.TenantId("GMS",84,1)` (and `!= uuid.Nil`); the
`tenants/<uuid>` branch is unchanged.

**Implement:** in `tenantFromParams` (~38) `case p.ScopeKey == "shared":` set
`id = canonical.TenantId(p.Region, p.MajorVersion, p.MinorVersion)` and drop the
`uuid.Parse(canonical.TenantUUID)` block. Keep `canonical` import; drop `uuid` only if it becomes unused
(the `tenants/` branch still parses a uuid → import stays).

**Verify:** `go test ./data/...`.

---

## T5 — Version-scoped publish (C5)  *(after rebase — see pre-flight)*

**Files:** `baseline/publish.go`, `baseline/publish_test.go`

**Test first:** assert the generated COPY SQL for a given `(region,major,minor)` contains
`WHERE tenant_id = '<canonical.TenantId(region,major,minor)>'` and **not** the all-zeros literal. (String
assertion — no live PG. If post-rebase the code exposes `copyOutSQL`, test that; otherwise test a small
extracted SQL-builder.)

**Implement:** thread `region string, major, minor int` from `Publish` → `dumpTable` → the SQL builder
(`runCopyOut`/`copyOutSQL`). Build the WHERE with
`canonical.TenantId(region, uint16(major), uint16(minor)).String()` instead of `canonical.TenantUUID`.
Preserve the `ORDER BY <orderColumn(table)>` from the merged ORDER-BY fix (do not regress it). Keep
`canonical` import.

**Verify:** `go test ./baseline/`. Round-trip (publish→restore→read) is operational (T8 runbook), not a
sqlite unit test.

---

## T6 — Version-scoped status `scope=shared` (C6)

**Files:** `data/status.go`, test for `resolveStatusTenantId`

**Test first:** `resolveStatusTenantId` with `?scope=shared` + operator header and a tenant model for
`(GMS,84,1)` returns `canonical.TenantId("GMS",84,1).String()`, ok=true; `scope=tenant` returns the
tenant's own id; missing operator header → 403/ok=false (unchanged).

**Implement:** in `resolveStatusTenantId` (~127) the `case "shared":` returns
`canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()).String()` instead of
`canonical.TenantUUID`. Keep `canonical` import.

**Verify:** `go test ./data/...`.

---

## T7 — Purge guard generalization (C7)

**Files:** `tenantpurge/handler.go`, `tenantpurge/purge.go`, `tenantpurge/*_test.go`

**Test first:**
- `purge.go`: existing test that `Purge(..., uuid.Nil)` returns `ErrCanonicalRefused` stays green
  (defense-in-depth retained).
- handler/guard: a request whose `{id}` equals `canonical.TenantId(reqRegion,reqMajor,reqMinor)` (matching
  request tenant headers) is refused with `ErrCanonicalRefused`/403; a normal v4 tenant id is allowed.

**Implement:** in `purgeInner` (`handler.go:39-44`), after parsing `id`, get
`t := tenant.MustFromContext(r.Context())` and refuse when
`id.String()==canonical.TenantUUID || canonical.IsCanonical(id, t.Region(), t.MajorVersion(), t.MinorVersion())`
(return 403 with `ErrCanonicalRefused.Error()`), before calling `Purge`. Leave `Purge`'s existing all-zeros
guard in place. Add `atlas-data/canonical` + `atlas-tenant` imports to the handler as needed.

**Verify:** `go test ./tenantpurge/`.

---

## T8 — Operator runbook (docs, no code)

**Files:** `docs/runbooks/ephemeral-pr-deployments.md` (and/or a new
`docs/runbooks/canonical-version-migration.md`) — repo-root docs, not the module.

Document the provision-before-delete cutover (design §6), enumerating all six versions
(GMS 83.1/84.1/87.1/92.1/95.1, JMS 185.1):
1. deploy new atlas-data image;
2. per version: `POST /api/data/process?scope=shared` (operator header + version headers);
3. verify per version: `GET /api/data/status?scope=shared` non-zero + a spot read from a no-per-tenant-data
   tenant of that version;
4. per version: `POST /api/data/baseline/publish` (un-breaks ephemeral `auto` bootstrap);
5. idempotent legacy cleanup:
   `DELETE FROM {documents,monster_search_index,npc_search_index,reactor_search_index,map_search_index,item_string_search_index} WHERE tenant_id = '00000000-0000-0000-0000-000000000000';`
Note: `atlas-pr-bootstrap` needs no change (OQ-4).

---

## T9 — Full verification & done-check (no new code)

From the worktree root:
- `cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./... && go build ./...` — all clean.
- `tools/redis-key-guard.sh` from repo root — clean.
- `docker buildx bake atlas-data` from worktree root **only if `go.mod` changed** (not expected).
- Residual-sentinel grep: `grep -rn "canonical.TenantUUID" --include='*.go'` should show only the const def,
  the legacy-refusal guard(s), and the determinism test — **no write/fallback uses**. `grep -rn "uuid.Nil"`
  in `document/` and `searchindex/` should show no canonical-fallback uses (only the searchindex error
  return).

---

## Rollout (operational, post-merge — tracks FR-6 / AC 7-8)
Run T8's runbook against each environment (atlas-main first, then PR envs as they cycle). FR-6 "all six
versions populated + baselines published" and "no legacy all-zeros rows remain" are verified by executing +
checking the runbook on the live env — they are not gated by the code branch itself.

## Per-task done = its `go test ./<pkg>/` green. Branch done = T9 all green + T1–T7 tests added.
