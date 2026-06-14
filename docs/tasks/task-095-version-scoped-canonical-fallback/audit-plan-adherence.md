# Plan Audit — task-095-version-scoped-canonical-fallback

**Plan Path:** docs/tasks/task-095-version-scoped-canonical-fallback/plan.md
**Audit Date:** 2026-06-14
**Branch:** task-095-version-scoped-canonical-fallback
**Base Branch:** origin/main (BASE b6251031974a5a641afee1b1c85dabf64653c672, HEAD 129fa6b6bb8933fd0c4ad7594cf65fe4c09ba7f7)

## Executive Summary

All nine plan tasks (T1–T9) were faithfully implemented exactly as specified, each with accompanying tests. `go build`, `go vet`, and `go test -race ./...` are all clean in the atlas-data module. The only verification gap is `tools/redis-key-guard.sh` reporting FAIL — confirmed to be a pre-existing repo-wide environment artifact that reproduces identically on clean origin/main and is unrelated to this task (the diff introduces zero redis code). Recommendation: READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| T1 | Canonical id helper (`Namespace`/`TenantId`/`IsCanonical` + tests) | DONE | `canonical/canonical.go:26-41` — `Namespace` UUIDv5 with MUST-NOT-change warning (`:22-25`), `TenantId` uses exact `"canonical:%s:%d.%d"` format from plan (`:33`), `IsCanonical` (`:39-41`). `TenantUUID` const retained (`:17`). Tests in `canonical/canonical_test.go` (68 lines) incl. determinism pin. |
| T2 | Document fallback in ByIdProvider + AllProvider | DONE | `document/storage.go:44` (ByIdProvider) and `:86` (AllProvider) both use `canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion())` instead of `uuid.Nil`. Unused `uuid` import dropped (file imports `atlas-data/canonical`, no `google/uuid`). Tests extended in `document/storage_test.go` (+143 lines) covering v83/v84 isolation + GetById/GetAll agreement. |
| T3 | Search-index fallback in ResolveTenantId | DONE | `searchindex/searchindex.go:93` returns `canonical.TenantId(...)`; `uuid.Nil` retained only at the error return `:88` (exactly as plan specified). `atlas-data/canonical` import added (`:9`). Tests +88 lines incl. multi-version variant. |
| T4 | Version-scoped shared ingest in tenantFromParams | DONE | `data/workers/runtime.go:39-40` `case p.ScopeKey == "shared":` sets `id = canonical.TenantId(p.Region, p.MajorVersion, p.MinorVersion)`; old `uuid.Parse(TenantUUID)` block removed; `tenants/<uuid>` branch unchanged (`:41-46`). New `runtime_test.go` (86 lines) asserts `Id() != uuid.Nil` and per-version distinctness. |
| T5 | Version-scoped publish (copyOutSQL threading) | DONE | `baseline/publish.go`: `region, major, minor` threaded `Publish`→`dumpTable` (`:74,:103`)→`runCopyOut` (`:114,:124`)→`copyOutSQL` (`:130,:150`). `copyOutSQL:151` builds WHERE with `canonical.TenantId(region, uint16(major), uint16(minor)).String()`; `ORDER BY orderColumn(table)` preserved (`:152,:160-176`). Tests +50 lines assert canonical id present and all-zeros absent. |
| T6 | Status scope=shared via resolveStatusTenantId | DONE | `data/status.go:128` `case "shared":` returns `canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()).String()`; operator-header gate retained (`:124-127`). Tests +61 lines incl. assertion that result is NOT the all-zeros sentinel. |
| T7 | Purge guard generalization | DONE | `tenantpurge/handler.go:47-51` reads `tenant.MustFromContext`, refuses with 403 when `id.String() == canonical.TenantUUID \|\| canonical.IsCanonical(id, ...)`, before calling `Purge`. `Purge`'s all-zeros guard left in place (`purge.go:32`, defense-in-depth). Tests +128 lines. |
| T8 | Operator runbook | DONE | New `docs/runbooks/canonical-version-migration.md` (147 lines): provision-before-delete ordering, all six versions table (GMS 83/84/87/92/95.1 + JMS 185.1), ingest/verify/publish/cleanup steps, OQ-4 "atlas-pr-bootstrap needs no change". Bidirectional cross-ref added to `ephemeral-pr-deployments.md:6-9`. |
| T9 | Full verification | DONE | build/vet/test-race clean (see table below); residual-sentinel grep clean. redis-key-guard FAIL is pre-existing (see Skipped/Deferred). |

**Completion Rate:** 9/9 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None silently skipped. One verification sub-item warrants a note:

- **T9 redis-key-guard:** `tools/redis-key-guard.sh` exits 1 ("FAIL — raw keyed redis client calls found"). Investigation shows: (a) the task-095 diff introduces zero redis code (only doc references to the guard itself); (b) the guard exits 1 identically when run against the clean main repo (`<repo-root>`); (c) the output lists no actual `file:line` violations — only repeated `./... matched no packages` warnings, indicating a GOWORK=off workspace-resolution artifact in this environment, not a real keyed-redis call. This is a pre-existing environmental condition, NOT a regression from this branch.

- **FR-6 operational rollout** (provisioning all six versions in live envs + removing legacy all-zeros rows) is explicitly out of the code branch and tracked by the T8 runbook — correctly excluded from the code deliverable, not a code gap.

## Residual-Sentinel Grep (plan T9)

- `grep -rn "canonical.TenantUUID" --include='*.go'`: matches only the const definition (`canonical.go:17`), the two legacy-refusal guards (`tenantpurge/purge.go:32`, `tenantpurge/handler.go:48`), and tests. No write/fallback uses. CLEAN.
- `grep -rn "uuid.Nil"` in `document/`: no non-test uses. `searchindex/`: only the error return at `searchindex.go:88`. CLEAN.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-data | PASS | PASS | `go build ./...` exit 0; `go vet ./...` exit 0; `go test -race ./...` all `ok`. Key packages re-run uncached (`-count=1`): canonical, baseline, tenantpurge, data, data/workers, document, searchindex all PASS. |

go.mod / go.sum unchanged on this branch → `docker buildx bake atlas-data` not required per plan T9 (and CLAUDE.md build rule 4).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None blocking. Optional:

1. (Environmental, not branch-scoped) The repo-wide `tools/redis-key-guard.sh` FAIL under GOWORK=off should be triaged separately — it predates this branch and produces no file:line evidence. It does not gate this task.
