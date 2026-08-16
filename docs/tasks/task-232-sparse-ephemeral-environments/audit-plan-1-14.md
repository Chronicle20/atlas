# Plan Audit — task-232-sparse-ephemeral-environments (Tasks 1-14)

**Plan Path:** docs/tasks/task-232-sparse-ephemeral-environments/plan.md
**Audit Date:** 2026-08-16
**Branch:** task-232-sparse-ephemeral-environments
**Base Branch:** main (range audited: c8d44127c..418b2caf9)
**Scope:** Plan Tasks 1 through 14 only (plan.md lines 61-1599)

## Executive Summary

All 14 tasks in this shard's range are fully implemented with strong, verifiable
evidence: the query-scope audit document was completed in three parts plus
follow-up fix-round commits, the `atlas-storage` npc-context tenant-collision
defect was fixed with a dedicated commit and collision test, five services'
Redis state (`atlas-monsters`, `atlas-summons`, `atlas-npc-shops`, `atlas-doors`)
were migrated to the tenant-scoped Redis API, the environment-scoped Redis
registry was added and wired into `atlas-data`'s `ingestrun`, the bare Redis
constructor guard (`rediskeyguard`) was built and is clean against the fleet,
and the `environment` column plus scoped read/write paths landed in both
`atlas-configurations` and `atlas-tenants` with C2 regression tests. Every
module I built and tested locally (`go build ./... && go test ./... -count=1`)
passed. No skipped or partial tasks found in this range.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Query-path sweep, part 1 of 3 (`atlas-account`…`atlas-fame`) | DONE | `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` created; commit `4dea1087b` "docs(task-232): query-path scope audit, part 1 of 3". Doc contains 215 table rows total across all three sweeps. |
| 2 | Query-path sweep, part 2 of 3 (`atlas-families`…`atlas-portals`) | DONE | Commit `9787becad` "docs(task-232): query-path scope audit, part 2 of 3"; follow-up fix `6f8024345` "fix round 1 — atlas-kites was missing from part 2 sweep" shows the sweep was actually audited for completeness, not just claimed. |
| 3 | Query-path sweep, part 3 of 3 + FR-8.6 disposition list | DONE | Commit `9201884be` "docs(task-232): query-path scope audit, part 3 of 3 + FR-8.6 dispositions". `## 2. Non-Postgres deployment-scoped resources (FR-8.6)` section present at `query-scope-audit.md:334`. Additional fix-round commits (`029a80667`, `f485165e6`, `b67e11f78`, `3132fa29e`, `a43b8a2c1`) show the audit was iterated on and UNSCOPED findings were remediated, not swept under the rug — e.g. `a43b8a2c1` "UNSCOPED dispositions and tenant-defect remediation". |
| 4 | Fix `atlas-storage` `npc-context` cross-tenant defect | DONE | `services/atlas-storage/atlas.com/storage/storage/cache.go:48,57,62` — `Get`/`Put`/`Remove` all take `t tenant.Model` as first arg, matching the planned interface exactly. Commit `05dd24392` "fix(atlas-storage): tenant-scope the npc-context cache". Module build/test: PASS (`atlas-storage/storage` ok). |
| 5 | Migrate `atlas-monsters` Redis state | DONE | `services/atlas-monsters/atlas.com/monsters/monster/registry.go:281-282` uses `NewTenantRegistry`/`NewTenantKeyedSet` for `monster`/`monster-map`; `attack_cooldown.go:28` (`monster-attack-cooldown`), `cooldown.go:29` (`monster-cooldown`), `puppet_registry.go:55-56` (`monster-puppet`, `monster-puppet-field`) confirm all 6 planned namespaces migrated. Commit `0c23f3e64` "refactor(atlas-monsters): move Redis state to the tenant-scoped API", plus later hardening `9198e0acf`/`4f00f47d9`/`d2bcf6e04`. Module build/test: PASS. |
| 6 | Migrate `atlas-summons` and `atlas-npc-shops` Redis state | DONE | `services/atlas-summons/atlas.com/summons/summon/registry.go:176-178` — `summon`, `summon-map`, `summon-owner` all `NewTenantRegistry`/`NewTenantKeyedSet`. `services/atlas-npc-shops/atlas.com/npc/shops/cache.go:39` — `npc-shop:consumables` uses `NewTenantRegistry`. Commit `982ed082d` "refactor(atlas-summons,atlas-npc-shops): move Redis state to the tenant-scoped API"; follow-up `268446708` strengthened the tenant-isolation test. Both modules build/test: PASS. |
| 7 | Migrate `atlas-doors` Redis state (callers build the keys) | DONE | `services/atlas-doors/atlas.com/doors/door/registry.go:77-80` — all four namespaces (`door`, `door-field`, `door-owner`, `door-town`) use `NewTenantRegistry`/`NewTenantKeyedSet`. Commit `bffd89034` "refactor(atlas-doors): move Redis state to the tenant-scoped API"; follow-up `7759d4a16` "drop bare Redis mirrors, use TenantRegistry/TenantKeyedSet *AcrossTenants". Module build/test: PASS. |
| 8 | Add environment-scoped Redis API; migrate `atlas-data` `ingestrun` | DONE | `libs/atlas-redis/environment.go` and `environment_test.go` created; `libs/atlas-redis/keys.go:60,67` implement `environmentEntityKey`/`environmentScanPattern`. Commit `87c93f568` "feat(atlas-redis): add the environment-scoped registry; migrate atlas-data ingestrun". `libs/atlas-redis` build/test: PASS. |
| 9 | Withdraw bare Redis constructors; extend `rediskeyguard` | DONE | `tools/rediskeyguard/analyzer.go` implements the bare-constructor check with a documented `bareConstructorAllowlist` (14 entries per the plan's Task 16 sign-off doc). Commit `12bcc3038` "feat(task-232): rediskeyguard bare-constructor check", plus `2ba148e84` "stop flagging already-tenant-scoped constructors, allowlist 12 verified sites" and `5933da14b`/`5db61335a` fix rounds closing a live cross-tenant Redis collision found by the guard (`atlas-transports`). `./tools/redis-key-guard.sh` run directly: exit 0, "64 module(s), 8 parallel", no findings. Note: Step 6 ("unexport what nothing legitimately calls") did not lowercase any constructor — verified this is correct, not skipped: every bare constructor (`NewRegistry`, `NewIndex`, `NewUint32Index`, `NewIDGenerator`, `NewTTLRegistry`, etc.) still has legitimate external callers (`atlas-messengers`, `atlas-invites`, `atlas-expressions`, `atlas-parties`, `atlas-kites`, `atlas-rps`, `atlas-merchant`), all covered by the allowlist, so none qualified for unexporting. `tools/rediskeyguard` build/test (via `GOWORK=off go test ./...`): PASS. |
| 10 | Record intent of every deliberately-global Redis resource | DONE | `query-scope-audit.md:1006` `## 5. Deliberately-global Redis resources (Task 10)` section present. Commit `61f6401e1` "docs(task-232): state the intent of every globally-scoped Redis resource". |
| 11 | Add `environment` column to `atlas-configurations`' three control-plane tables | DONE | `Environment string` field present in `tenants/entity.go:24,39`, `templates/entity.go:23`, `services/entity.go:22,38` (both `Entity` and `HistoryEntity`). `environmentcol/migration.go` implements `BackfillEnvironment` over `tenants, tenant_history, templates, services, service_history` — note the backfill covers `tenant_history` too, beyond the plan's literal four-table list, per follow-up commit `ccab38405` "scope tenant_history by environment too". Commit `93afb84cd` "add the environment dimension to the control-plane tables". Module build/test: PASS. |
| 12 | Add `environment` column to `atlas-tenants` | DONE | `tenant/model.go:45` `func (m Model) Environment() string`; `tenant/builder.go:73` `SetEnvironment`; `tenant/environment_migration.go` created. Commit `8c8c506fa` "feat(atlas-tenants): add the environment dimension to the tenant table"; follow-up `3a4662f03` "cover the real Update path and its environment carry-forward". Module build/test: PASS. |
| 13 | Environment-scoped reads/writes in `atlas-configurations` | DONE | `services/atlas-configurations/atlas.com/configurations/scope/scope.go` and `scope_test.go` created; `templates/overlay.go` created implementing the `OverlaySingle`/`OverlayCollection`/`VisibleById` distinction called for by the plan. `rest/handler.go:38,44` maps `scope.ErrCrossEnvironmentWrite` to 403. Commit `0d61a3e10` "feat(atlas-configurations): environment-scoped control-plane reads and writes"; fix round `e9b5b2134` "task-13 fix round 1 - lint gate, sqlite pool, reseed guard test" shows real iteration rather than a first-pass claim. Module build/test: PASS. |
| 14 | Environment-scoped reads/writes in `atlas-tenants` | DONE | `services/atlas-tenants/atlas.com/tenants/scope/scope.go` created (duplicated per plan's explicit instruction, not imported cross-service); `tenant/scope_test.go` present matching the plan's corrected `AllProvider` signature (not `GetAll`). `rest/handler.go:38,44` maps the 403. Commit `4edc52aeb` "feat(atlas-tenants): environment-scoped tenant reads and writes"; fix rounds `d94280b7d` "unscope write pre-lookup so AuthorizeWrite runs" and `d57b75289` "AuthorizeWrite must authorize a legacy caller against any target" show real bugs found and fixed post-implementation. Module build/test: PASS. |

**Completion Rate:** 14/14 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| services/atlas-storage/atlas.com/storage | PASS | PASS | includes `atlas-storage/storage` (npc-context cache) |
| services/atlas-monsters/atlas.com/monsters | PASS | PASS | `monster` package test run took ~16s (redis-backed tests) |
| services/atlas-summons/atlas.com/summons | PASS | PASS | |
| services/atlas-npc-shops/atlas.com/npc | PASS | PASS | |
| services/atlas-doors/atlas.com/doors | PASS | PASS | |
| libs/atlas-redis | PASS | PASS | includes `environment_test.go` |
| tools/rediskeyguard | PASS | PASS | separate go.mod, not in go.work; built/tested with `GOWORK=off` |
| services/atlas-configurations/atlas.com/configurations | PASS | PASS | includes `scope`, `templates` (overlay) packages |
| services/atlas-tenants/atlas.com/tenants | PASS | PASS | includes `scope`, `tenant` packages |

Note: this shard did not run the full repo-wide `tools/verify.sh` gate (docker
bake, cross-module analyzer guards, `-race`) — only module-local `go build`/`go
test -count=1` for the modules this range's tasks touch, per the audit
process's build/test step. The task's own `query-scope-audit.md` §"Step 1
evidence" documents a flagless `tools/verify.sh` PASS run by the controller
(after fixing a `libs/atlas-env` Docker/go.work omission), which is consistent
with but external to this shard's scope.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; full-plan disposition depends on the other shards covering Tasks 15+)

## Action Items

None. All 14 tasks in this range are DONE with file:line evidence, commits,
and passing module-local build/test.
