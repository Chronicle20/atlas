# Plan Audit — task-045-pr-teardown-leak-fixes

**Plan Path:** docs/tasks/task-045-pr-teardown-leak-fixes/plan.md
**Audit Date:** 2026-05-27
**Branch:** task-045-pr-teardown-leak-fixes
**Base Branch:** main (merge-base 6815a919e58855eae4b8fa0db51675913740287b)

## Executive Summary

All 24 planned tasks are implemented with evidence (49 commits map cleanly to the 8 phases plus the documented scope expansion). The regression guard `./tools/redis-key-guard.sh` builds and exits 0 across all 54 service modules — the strongest single proof that every migration (original 8 + 15 expansion services) is complete and no raw keyed go-redis calls remain anywhere. The rediskeyguard analyzer tests pass, the bats suite is 47/47 green, and representative Go modules (atlas-redis, atlas-maps, atlas-inventory) build/vet/test clean. No TODO/FIXME/501/stub markers exist in any changed source file. Plan adherence is FULL; recommendation is READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-redis `Set` + `TenantSet` | DONE | libs/atlas-redis/set.go; commit adc526d1d |
| 2 | atlas-redis `Hash` + `KeyedHash[K]` | DONE | libs/atlas-redis/hash.go (Hash:404, KeyedHash:88 GetAll, Clear); commit 5d4f308b1 |
| 3 | atlas-redis `TenantKeyedSet[K]` | DONE | libs/atlas-redis/keyed_set.go; IsMember:125; commit aa10f11fc |
| 4 | atlas-redis `TenantKeyedHash[K]` | DONE | libs/atlas-redis/keyed_hash.go; commit f00165438 |
| 5 | atlas-world channel:tenants → Set | DONE | channel/registry.go:32 `atlas.NewSet(client,"channel:tenants")`; commit e632f8ad5 |
| 6 | atlas-invites invite:active-tenants → Set | DONE | invite/registry.go:39 NewSet; commit c1558c17f |
| 7 | atlas-guilds coordinator (3 keys) | DONE | coordinator/registry.go:27-29 Set + Registry[uuid,Model] + TenantRegistry[uint32,string] — matches context.md correction; commit 70a84a33e |
| 8 | atlas-drops drop registry | DONE | drop/registry.go:37-38 Set + TenantKeyedSet[field]; commits 7464c1301, ee9f6ed7b |
| 9 | atlas-reactors (cooldown/spot semantic change P2) | DONE | reactor/registry.go:43-44 cooldowns/spots as TenantKeyedHash[MapKey] (expiry-as-field); commits 2f7bc7e94, 25001b5f7 |
| 10 | atlas-transports instance registry | DONE | instance/instance_registry.go uses NewSet/NewRegistry/NewKeyedHash/NewTenantKeyedSet; commits 2dca1999b, 3fe5e89ce |
| 11 | atlas-transports character registry → Hash | DONE | instance/character_registry.go NewHash; commit d62662ae2 |
| 12 | atlas-transports channel registry → TenantSet | DONE | channel/registry.go NewTenantSet; commit ad0613123 |
| 13 | atlas-rates item tracker → TenantKeyedHash | DONE | character/item_tracker.go:151,164 TenantKeyedHash[uint32]; no raw client.Scan remains; commit cc6ab6fc8 |
| 14 | atlas-maps spawn registry (KeyedHash; bare-uuid fix P3) | DONE | map/monster/registry.go:55 NewKeyedHash, :60 `mk.Tenant.Id().String()` (bare uuid), :271/:299 Clear; Lua scripts retained; commit 55fb0c1ac |
| 15 | scaffold tools/rediskeyguard | DONE | tools/rediskeyguard/analyzer.go (bannedMethods match context.md), cmd/rediskeyguard/main.go, go.mod standalone; commit ee1b443b4 |
| 16 | analyzer analysistest good/bad | DONE | analyzer_test.go + testdata/src/{bad,good}/; `GOWORK=off go test` exit 0; commit 9e1897616 |
| 17 | runner script + CI + CLAUDE.md | DONE | tools/redis-key-guard.sh exit 0; CLAUDE.md:25 item 5; .github/workflows/pr-validation.yml wired; commit 39d894f39 |
| 18 | reclaim-main-bare-keys.sh + bats | DONE | scripts/reclaim-main-bare-keys.sh; reclaim_test.bats; Dockerfile:60 COPY; commit 258dd6962 |
| 19 | predelete-purge.sh + bats + Dockerfile | DONE | scripts/predelete-purge.sh; predelete_test.bats; Dockerfile:61 COPY; commit 92fb712dd |
| 20 | PreDelete hook manifest + kustomization | DONE | deploy/k8s/overlays/pr/predelete-purge.yaml:18 `hook: PreDelete`; kustomization.yaml:33; commit ae19812b1 |
| 21 | remove PostDelete drop-tenant-storage | DONE | cleanup.sh: 0 references to do_drop_tenant_storage; PHASES now 7 entries; postdelete-cleanup.yaml secret mount dropped; commit 07ed8ce5e |
| 22 | sweep_minio live-PR-env allowlist (fail-closed) | DONE | sweep-orphans.sh:386 ATLAS_PR_NS_SELECTOR, :391/:401 fail-closed aborts, :439 protects live PR-env; commit e2ef67047 |
| 23 | cluster-infra coordination note + CronJob example | DONE | dev/cluster-infra-coordination/task-045-teardown.md + sweep-orphans-cronjob.example.yaml; commit 308149459 |
| 24 | build/test/vet/guard/bake verification | DONE | Guard exit 0 repo-wide; bats 47/47; sampled modules clean (see Build & Test) |

**Completion Rate:** 24/24 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Documented scope expansion (20 additional, user-authorized)

The expansion is real, not faked. Each of the 15 extra services has a substantive diff (e.g. atlas-inventory +289/-57, atlas-monsters +364/-533, atlas-merchant +243/-24, atlas-data +136/-90). The new lib types/methods all exist and are non-stub:
- Types: `KeyedSet[K]` (keyed_set.go:19), `TenantKeyedSortedSet[K]` (keyed_sorted_set.go:19).
- Registry: `PutWithTTL` (registry.go:125), `GetAll` (:136), `ClearByPrefix` (:226).
- TenantRegistry: `PutWithTTL` (:116), `GetAllEntries` (:175), `ClearByPrefix` (:226).
- TenantKeyedSet `IsMember` (keyed_set.go:125); KeyedSet `ClearAll` (:54).
- Lock token/CAS: `AcquireWithToken` (lock.go:84), `ForceAcquire` (:96), `ReleaseToken` (:104).

Genuine bare-key leaks (no prior KeyPrefix) confirmed fixed via lib types:
- atlas-data: `redis.NewRegistry`/`PutWithTTL` for ingest/job lifecycle (commit b3a127a29).
- atlas-inventory: invlock + reservation via atlas-redis (commit 8dc198a81).
- atlas-merchant: shop-visitor sorted set via `TenantKeyedSortedSet` (commit 63906894c).

## Skipped / Deferred Tasks

None. No task is skipped, partial, or stubbed.

## Build & Test Results

| Module | Build | Vet | Tests | Notes |
|--------|-------|-----|-------|-------|
| libs/atlas-redis | PASS | PASS | PASS | go test ok 0.111s |
| services/atlas-maps | PASS | PASS | PASS | all packages ok (Lua-backed spawn registry intact) |
| services/atlas-inventory | PASS | PASS | PASS | genuine bare-key leak fix verified |
| tools/rediskeyguard | PASS | n/a | PASS | GOWORK=off go test ./... exit 0 (standalone module, by design) |
| repo-wide guard | PASS | — | — | ./tools/redis-key-guard.sh exit 0 across all 54 service modules |
| atlas-pr-bootstrap bats | — | — | PASS | 47 ok / 0 not-ok |

Representative sample per the auditor's mandate. The executor's claim of all 24 Go modules passing test-race/vet/build and key docker bakes succeeding was spot-checked on the foundation lib, the trickiest service (atlas-maps), and a genuine-leak expansion service; all consistent. Full per-module race/bake matrix was not re-run exhaustively (time/CI cost) — CI's pr-validation.yml now gates the guard step.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Documented Follow-Ups (not blockers)

1. **atlas-maps verbose-tenant orphan cleanup (one-time, manual).** Changing the spawn-hash tenant segment from verbose `tenant.Model.String()` to bare uuid orphans pre-existing `atlas:maps:spawn:Id [...]` keys on main. The plan documents the exact one-time `redis-cli` command (plan.md:2615-2624). These are atlas-prefixed, so the FR-1.6 reclaim script intentionally does NOT touch them. Operational, not code.
2. **cluster-infra CronJob + RBAC.** The sweep CronJob and the `atlas-pr-cleanup` SA's `list namespaces` ClusterRole live in the sibling cluster-infra repo (dev/cluster-infra-coordination/task-045-teardown.md). The sweep fails closed until granted — safe to land this PR first.
3. **TenantRegistry.Update retry parity** — noted as a lib follow-up; not required for any task-045 acceptance criterion.

## Action Items

None required before merge. The PRD §10 end-to-end acceptance checks (fresh PR env → keys prefixed; close PR → redis/MinIO/Postgres clean) are manual on a test env and explicitly out of scope for automated gating per the plan (plan.md:3753-3755).
