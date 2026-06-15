# Backend Audit — atlas-data (task-095 version-scoped canonical fallback)

- **Service Path:** services/atlas-data/atlas.com/data
- **Scope:** `git diff origin/main...HEAD -- services/atlas-data` (BASE b6251031, HEAD 129fa6b6)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-14
- **Build:** PASS
- **Vet:** PASS
- **Tests:** PASS (full `go test -race -count=1 ./...` exit 0, no failures)
- **redis-key-guard:** PASS (exit 0)
- **Overall:** PASS

## Build & Test Results

```
go build ./...        EXIT 0
go vet ./...          EXIT 0
go test -race -count=1 ./...   EXIT 0  (no FAIL/panic)
  canonical / baseline / data / data/workers / document / searchindex / tenantpurge / map all ok
tools/redis-key-guard.sh   EXIT 0
```

## Package Classification (Phase 2)

None of the touched packages is a JSON:API **domain package** — none has `model.go`
(`builder.go`/`ToEntity`/`Make`/`Transform` lifecycle). They are infrastructure/support
packages, so the DOM-01..DOM-20 lifecycle checklist is **N/A**. Applicable checks are
multi-tenancy correctness, SQL safety, determinism/immutability, layer separation,
DOM-10 (test DB tenant callbacks), and DOM-21 (atlas-constants duplication).

- `canonical/` — support pkg (no model.go/resource.go); pure id-derivation helper.
  `canonical.go:1`, no `model.go` (verified absent).
- `baseline/` — support pkg (publish/dump infra). No model/resource/builder.
- `document/` — support pkg (generic storage). No model/resource/builder.
- `searchindex/` — support pkg (generic search lib). No model/resource/builder.
- `tenantpurge/` — handler pkg; `handler.go` registers a DELETE (no input body) — GET/DELETE use `RegisterHandler`, correct (no `RegisterInputHandler` needed).
- `data/` — has `resource.go` but no `model.go`; `status.go` is a read handler. Sub-domain-ish; SUB checks below.

## Applicable Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Determinism/Immutability | Canonical id is deterministic UUIDv5, namespace pinned, frozen pin test | PASS | `canonical/canonical.go:23` (Namespace var), `:34` (TenantId NewSHA1), `canonical/canonical_test.go:60` frozen pin `144ba144-…`, `:14` determinism, `:22` uniqueness |
| SQL safety (COPY WHERE) | Interpolated tenant id is UUID-derived, not user input; table/order are hardcoded whitelists | PASS | `baseline/publish.go:131` `tenantId := canonical.TenantId(...).String()`; `:132` `fmt.Sprintf("COPY ... WHERE tenant_id = '%s' ORDER BY %s ...", table, tenantId, orderColumn(table))`; `table` ∈ `baseline/dump.go:13` `DumpTables` (hardcoded var); `orderColumn` `baseline/publish.go:160` is a closed switch over literal column names — no caller-supplied SQL fragment |
| Multi-tenancy — read fallback symmetric | GetById and GetAll both fall back to the same version-scoped canonical id | PASS | `document/storage.go:44` (ByIdProvider) and `:86` (AllProvider) both `tenant.Create(canonical.TenantId(t.Region(),t.MajorVersion(),t.MinorVersion()), …)`; test `document/storage_test.go:149` `TestVersionScopedCanonicalFallback_GetAll`, `:185` `_GetById` prove no cross-version bleed |
| Multi-tenancy — search resolve | Zero-row tenant resolves to version-scoped canonical id, not uuid.Nil | PASS | `searchindex/searchindex.go:93` `return canonical.TenantId(t.Region(),t.MajorVersion(),t.MinorVersion()), nil`; test `searchindex/searchindex_test.go:429` `TestResolveTenantId_MultiVersion` |
| Multi-tenancy — ingest scope | scope=shared writes to version-scoped canonical id | PASS | `data/workers/runtime.go:40` `id = canonical.TenantId(p.Region,p.MajorVersion,p.MinorVersion)`; test `data/workers/runtime_test.go:15`, `:41` distinct-per-version |
| Multi-tenancy — status read scope | shared scope reads version-scoped canonical id, operator-gated | PASS | `data/status.go:128` `return canonical.TenantId(t.Region(),t.MajorVersion(),t.MinorVersion()).String(), true`; operator gate `:124-126`; test `data/status_test.go` |
| Purge guard | Refuses both legacy sentinel and version-scoped canonical id with 403, before Purge | PASS | `tenantpurge/handler.go:48-51` checks `id.String() == canonical.TenantUUID || canonical.IsCanonical(id, t.Region(),…)` → 403; tests `tenantpurge/handler_test.go:91` (version-scoped 403), `:111` (sentinel 403), `:131` (non-canonical not refused) |
| Layer separation | No new handler→provider/DB bypass introduced | PASS | `data/status.go` change is a one-line id-source swap inside existing `resolveStatusTenantId` helper; no new DB call from handler |
| DOM-10 — test DB tenant callbacks | SQLite test DBs register tenant callbacks | PASS | `searchindex/searchindex_test.go:43` `database.RegisterTenantCallbacks(...)`; `document/storage_test.go:66` same |
| DOM-21 — atlas-constants duplication | No new domain type/const duplicating libs/atlas-constants | PASS | `canonical.TenantId(region string, major, minor uint16)` uses primitives matching `tenant.Model` getters (`MajorVersion()/MinorVersion()` return `uint16`, verified `libs/atlas-tenant/processor.go:30`); no new region/version type declared |
| Test quality | Real behavior (SQLite + tenant callbacks), no `*_testhelpers.go`, no mock-only | PASS | searchindex/document tests seed real rows via `database.WithoutTenantFilter` and assert real query results; tenantpurge routes through real gorilla mux + httptest; no `_testhelpers.go` files added (verified) |
| Dead-code cleanup | uuid.Nil/parse-sentinel paths removed where replaced | PASS | `data/workers/runtime.go` removed the `uuid.Parse(canonical.TenantUUID)` error branch; `document/storage.go` dropped now-unused `github.com/google/uuid` import |

## Notes / Observations (non-blocking, no action required)

- `canonical.TenantUUID` (all-zeros sentinel) is retained intentionally as a
  defense-in-depth purge refusal (`tenantpurge/handler.go:49`) and is covered by
  `TestHandlerRefusesAllZerosSentinel`. Not dead code.
- The `Namespace` carries an explicit WARNING comment (`canonical/canonical.go:14-21`)
  and a frozen-id pin test, correctly guarding the immutability invariant that a
  namespace change would orphan every canonical row. This is the right safeguard.
- The COPY statement string-interpolates `tenant_id` rather than using a bound
  parameter, but the value is strictly `uuid.UUID.String()` output (RFC-4122,
  hex+hyphens only) — not injectable. `table` and order columns are closed literal
  sets. No finding.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.
