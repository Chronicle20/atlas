# Plan Audit — task-300-shared-script-operations

**Plan Path:** docs/tasks/task-300-shared-script-operations/plan.md
**Audit Date:** 2026-09-04
**Branch:** task-300-shared-script-operations
**Base Branch:** main (commit range fe07ae424..c1d75718a)

## Executive Summary

All 14 plan tasks are implemented and evidenced in the diff; every affected service
(`atlas-script-core`, `atlas-map-actions`, `atlas-reactor-actions`, `atlas-portal-actions`,
`atlas-npc-conversations`, `atlas-saga-orchestrator`) builds and tests clean under an
independent re-run in this audit. The Task 13 guard-scope deviation (four
script-operation-table services plus an inline `// script-ops-guard:allow` convention,
rather than a repo-wide scan) is confirmed to still satisfy FR-12's intent: the guard
passes, its own 8-case self-test suite passes, and the three `allow` markers present are
each justified and outside the operation-table surface the guard exists to police. One
carried-forward finding remains open and unaddressed: `update_skill`'s expiration
handling is pinned by an ops-level table test (`TestUpdateSkill` in
`libs/atlas-script-core/ops/skill_test.go`) but not by any npc-conversations executor
test — only `create_skill`'s equivalent (`TestCreateStepCreateSkillHonoursExpiration`)
exists at that layer. This was already known as DEFERRED-MINOR from Task 11's review and
is reported here, not as a surprise.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `ops` package foundation (`Step`, `Target`, `Resolver`, `DirectResolver`, `ParamError`, param/range helpers) | DONE | `libs/atlas-script-core/ops/ops.go:1-260` matches the interface spec verbatim; `go.mod` carries the `atlas-saga` require+replace; commit `b40684a8c` |
| 2 | `SendMessage` (FR-13, FR-14) | DONE | `libs/atlas-script-core/ops/message.go:17-53` — `messageType`/`type` alias, `"5"`→`PINK_TEXT`, `"6"`→`BLUE_TEXT`, `PINK_TEXT` default; commit `8d1531ed3` |
| 3 | `SpawnMonster` (FR-15, FR-16, OQ-3) | DONE | `libs/atlas-script-core/ops/monster.go:26-95` — every parse failure hard-errors via `ParamError`; `Instance`/`Team` always populated, instance zeroed on cross-map spawn (OQ-3); commit `e72212467` |
| 4 | Environment and effect operations | DONE | `libs/atlas-script-core/ops/environment.go`, `effect.go` export `MoveEnvironment`, `ResetEnvironment`, `ShowIntro`, `ShowHint`, `PlayPortalSound`, `ApplyConsumableEffect`; commit `eaaab1e01` |
| 5 | Skill and quest operations (FR-17) | DONE | `libs/atlas-script-core/ops/skill.go`, `quest.go` export `CreateSkill`, `UpdateSkill`, `StartQuest`, `StageClearAttemptPq`; `skill_test.go` has full table coverage for both create and update expiration handling (`TestCreateSkill`, `TestUpdateSkill`); commit `76f6f2572` |
| 6 | Movement operations (FR-18) | DONE | `libs/atlas-script-core/ops/movement.go` exports `WarpToPortal`, `WarpToSavedLocation`, `SaveLocation`, `StartInstanceTransport`; commit `b9f422a40` |
| 7 | `atlas-map-actions` delegation | DONE | `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` imports `atlas-script-core/ops` (line 14), 11 call sites; `go build`/`go test` pass; commit `bff4987f0` |
| 8 | `atlas-reactor-actions` delegation | DONE | `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` imports ops (line 17), 11 call sites incl. `move_environment`/`reset_environment`; `go build`/`go test` pass; commit `b595cc948` |
| 9 | `atlas-portal-actions` delegation | DONE | `services/atlas-portal-actions/atlas.com/portal/script/executor.go` imports ops (line 19), 31 call sites; `optable_test.go` shows **zero diff** between `fe07ae424` and `c1d75718a` (untouched, per acceptance item) and all `TestOpTable_*`/`TestValidateOpTable_*` pass; commit `3b808dad8` |
| 10 | `atlas-npc-conversations` — resolver adapter + first 7 cases | DONE | `contextResolver`/`e.resolver()`/`e.target()` present; `case "send_message"`, `"spawn_monster"`, `"show_intro"`, `"show_hint"`, `"play_portal_sound"`, `"apply_consumable_effect"`, `"create_skill"` all delegate to `ops.*` and return the `(stepId, status, action, payload, error)` tuple (e.g. `operation_executor.go:1509-1514`); commit `105af0c1f` |
| 11 | `atlas-npc-conversations` — remaining 7 cases | DONE | `"update_skill"` (1517), `"warp_to_map"`→`ops.WarpToPortal` (1437-1442), `"warp_to_saved_location"` (2232), `"save_location"` (2225), `"start_instance_transport"` (2218), `"start_quest"` (1807), `"stage_clear_attempt_pq"` (2283) all delegate; `"warp_to_random_portal"` (1444) correctly left un-delegated as explicitly out of scope; commit `653df76b4` |
| 12 | Remove npc `saga` re-export shim (FR-11) | DONE | `services/atlas-npc-conversations/atlas.com/npc/saga/builder.go` deleted entirely (was a pure `= sharedsaga.Builder` re-export); `saga/model.go` reduced from 164 to 35 lines, keeps only the genuinely NPC-specific `ValidateCharacterStatePayload` wrapper; `processor.go`/`producer.go` now reference `sharedsaga.Saga` directly; commit `8bea52428` |
| 13 | `script-ops-guard.sh` + verify.sh wiring (FR-12) | DONE (accepted deviation) | `tools/script-ops-guard.sh` scoped to the four operation-table services + `// script-ops-guard:allow` convention (documented rationale in the script header); guard exits 0 on current tree; `tools/script-ops-guard_test.sh` — all 8 self-assertions pass; wired into `tools/verify.sh:848-853` gated on path touch; `docs/TODO.md:292-294` re-pointed to `services/atlas-reactor-actions/...executor.go` line numbers, with the "environment object manipulation" line correctly dropped (that TODO is resolved — `move_environment`/`reset_environment` now delegate to `ops.*`); commit `c1d75718a` |
| 14 | Orchestrator seam test + FR-20 sweep re-run | DONE | `TestHandleSpawnMonsterCarriesInstanceToField` in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go:2131` pins `Instance`/`Team` reaching `monsterP.SpawnMonster`; `docs/tasks/task-300-shared-script-operations/sweep-result.md` records the FR-20 re-run (2026-09-04 13:33:34 UTC) with all six design §7 rows matching; commit `a91751c34` |

**Completion Rate:** 14/14 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None of the 14 top-level tasks were skipped, partial, or silently deferred.

One **pre-existing, already-known** gap surfaces at the acceptance-checklist level and is
reported per the audit brief, not as a new finding:

- **`update_skill` expiration handling has no npc-conversations executor test pinning
  it.** `libs/atlas-script-core/ops/skill_test.go:180` (`TestUpdateSkill`) fully exercises
  `UpdateSkill`'s expiration parsing (including the `-1` sentinel, epoch-ms, and
  zero-falls-back cases) at the ops-library layer, and `skill.go:110-121` confirms
  `UpdateSkill` forwards `sp.expiration` into `saga.UpdateSkillPayload.Expiration`. But
  `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go`
  only has `TestCreateStepCreateSkillHonoursExpiration` (line 1302) — no equivalent test
  exists for the `update_skill` case at the executor/integration layer. This was flagged
  DEFERRED-MINOR in Task 11's review and remains unaddressed on this branch. Impact is
  low: the shared library's own test suite already proves the behavior; what's missing is
  an executor-level regression guard specifically for `update_skill`, matching the one
  that already exists for `create_skill`.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| `libs/atlas-script-core` | PASS | PASS | `ops` package: `go test ./...` ok |
| `atlas-map-actions` | PASS | PASS | incl. `script` package |
| `atlas-reactor-actions` | PASS | PASS | incl. `script` package |
| `atlas-portal-actions` | PASS | PASS | incl. `action`, `dedupe`, `kafka/consumer/saga`, `script` |
| `atlas-npc-conversations` | PASS | PASS | all subpackages ok, incl. `conversation`, `kafka/consumer/*` |
| `atlas-saga-orchestrator` | PASS | PASS | incl. `saga` package with the new seam test |
| `tools/script-ops-guard.sh` | N/A | PASS | exits 0 on current tree; `script-ops-guard_test.sh` — 8/8 self-assertions pass |
| Flagless `tools/verify.sh` | — | recorded PASS | Not independently re-run in this audit (full docker-bake gate is long-running); `agent-ledger.tsv` records `Tasks 13-14 verify.sh PASS` at `2026-09-04T14:22:01Z` against commit `c1d75718a` (current HEAD), consistent with every earlier task-pair's logged `verify.sh PASS` entries. Treated as PASS on the strength of that ledger plus this audit's own component-level build/test re-runs, all of which passed independently. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Non-blocking, carried-forward DEFERRED-MINOR) Add an npc-conversations executor test
   analogous to `TestCreateStepCreateSkillHonoursExpiration` for the `update_skill` case,
   to close the asymmetry between `create_skill` and `update_skill` test coverage at the
   executor layer. Not required for this branch per the prior review's disposition, but
   should not be lost as a follow-up.
