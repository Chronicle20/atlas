# Plan Audit — task-255-auto-aggro-mobs

**Plan Path:** docs/tasks/task-255-auto-aggro-mobs/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-255-auto-aggro-mobs
**Base Branch:** main (merge-base d17404dbc)
**HEAD at audit time:** b3606cdac (one commit past the b72b83c62 named in the brief — a coverage-manifest wire-layout correction landed after the brief was written; reviewed below and folded into Task 15)

## Executive Summary

All 15 plan tasks are fully implemented with direct file:line evidence matching the plan's prescribed shapes, constants, gate ordering, and interfaces almost verbatim. The two out-of-plan commits (`ac165f745` boundary-equality tests, `b72b83c62` seam-routing fix for `handleStatusEventAggroChanged`, and the further `b3606cdac` coverage-manifest wording fix) are legitimate review-driven strengthenings, not scope changes. `go build ./... && go test ./... -count=1` passes clean in all three affected module roots (`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-monsters/atlas.com/monsters`, `services/atlas-channel/atlas.com/channel`). The registry, IDA-export, and matrix promotion machinery for AUTO_AGGRO checks out exactly against the plan's Global Constraints table (all ten opcodes/addresses, including the corrected v84 194/`0x684492`). No skipped or deferred work found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `AutoAggro` codec + report linkage | DONE | `libs/atlas-packet/monster/serverbound/auto_aggro.go:14-71` matches the plan's struct/Encode/Decode shape exactly; 10 `packet-audit:verify` markers in `auto_aggro_test.go`; `tools/packet-audit/cmd/run.go:1202` adds the `CMob::ApplyControl` arm. `go test ./monster/serverbound/` passes. |
| 2 | Registry corrections (10 versions) | DONE | All ten `docs/packets/registry/<v>.yaml` files carry exactly one `AUTO_AGGRO` row each, opcode/`ida.address` matching the Global Constraints table verbatim (v84 corrected to 194/6833298); no duplicate opcodes observed. |
| 3 | Route AUTO_AGGRO in ten seed templates | DONE | `grep -l '"handler": "AutoAggro"' ... | wc -l` = 10; `template-opcode-order-guard.sh` and `template-duplicate-binding-guard.sh` both exit 0; `template_gms_12_1.json` untouched (confirmed via `git diff --name-only`). |
| 4 | Splice `CMob::ApplyControl` into 10 IDA exports | DONE | All ten `docs/packets/ida-exports/*.json` files resolve `functions["CMob::ApplyControl"]` to the exact address/direction=serverbound from the Global Constraints table; `git diff --stat` shows small per-file insertions only (14-16 lines each), consistent with `--splice` not a wholesale re-export. |
| 5 | Audit reports, evidence pins, matrix promotion | DONE | 10× `docs/packets/audits/<v>/MonsterAutoAggro.{json,md}` present; 10× `docs/packets/evidence/<v>/monster.serverbound.MonsterAutoAggro.yaml` present, each carrying the `verifies:` tag to `TestAutoAggro`; `STATUS.md`/`status.json` show `tier1: true` and all ten cells `state=verified` with matching opcodes; `feature-na-evidence.yaml` contains no `AUTO_AGGRO` entry (correct — nothing is n/a). |
| 6 | `information.Model.FirstAttack()` | DONE | `model.go:110-114`, `builder.go:11,50-54,82`, `rest.go:104` all present as specified; `TestExtractFirstAttack`/`TestModelBuilderSetFirstAttack` in `rest_test.go`. |
| 7 | Aggro lease state on the monster registry | DONE | `model.go:67-71,209-217,330-331`; `builder.go:38,67,134-136,238`; `registry.go` storedMonster field + `toStored`/`fromStored` mapping (lines 52,168,262), `ControlMonsterWithAggro` (421), `AggroSummary`/`SetAggro` (428-475); `processor.go:427` passes `p.now()`. Tests `TestRegistrySetAggro`, `TestRegistrySetAggroMissingMonster`, `TestControlMonsterWithAggroStampsLease`, `TestAggroRefreshedMsRoundTripsThroughRedis` all present in `registry_test.go`. Uses the accepted `newTestTenant(t)` substitution for tenant construction (Task 12/13/14's helper, reused here) — preserves intent, no `tenant.Create` arity issue. |
| 8 | Aggro lease release in the decay sweep | DONE | `aggro.go:30-35` `AutoAggroLeaseTtlMs = 15_000`; `aggro_task.go:89-97` the no-damage-entries branch calling `ReleaseAggroLease`; `registry.go:480-505` `ReleaseAggroLease` mirrors `DecayDamageEntries` shape. Tests `TestAggroDecayTaskReleasesExpiredAutoAggroLease`, `TestAggroDecayTaskLeaseSkipsBosses`, `TestAggroDecayTaskLeaseLeavesControllerIntact` present, plus the follow-up `ac165f745` exact-boundary pin. |
| 9 | `ProcessorImpl.SetAggro` gates + arbitration | DONE | `processor.go:1904-1977` implements all 5 gates in the exact order and wording the plan specifies (exists → alive → firstAttack via `testInformationLookup` seam → inField → arbitration with GM-hidden check before `startControl(..., true)`). `TestSetAggro_Gates`, `TestSetAggro_Arbitration`, `TestSetAggro_LeavesDamageEntriesUntouched` present in `set_aggro_test.go`; follow-up commit `33ef8c431` pinned exact emit counts. |
| 10 | `SET_AGGRO` consumer (atlas-monsters) | DONE | `kafka.go:32,168-176` (`CommandTypeSetAggro`, `setAggroCommandBody`); `consumer.go:66,232-238` registration + handler mirroring `handleForceControlCommand`. `TestSetAggroCommandUnmarshal`, `TestSetAggroCommandTypeConstant` present. |
| 11 | `SET_AGGRO` producer + `ControlCharacterId` mirror (channel) | DONE | `kafka/message/monster/kafka.go:23,127-134`; `monster/producer.go:226-228` `SetAggroCommandProvider`; `monster/processor.go:35,175-180` `SetAggro`; `live_mirror.go:30,77,151-166` `ControlCharacterId`/`UpdateControl`; `kafka/consumer/monster/consumer.go:331,373` wire `UpdateControl` into START_CONTROL/STOP_CONTROL. Mock updated (`mock/processor.go:28,152-155`). Tests `TestSetAggroCommandProviderShape`, `TestLiveMirrorUpdateControl` present. |
| 12 | Auto-aggro rate gate (channel) | DONE | `auto_aggro_gate.go` implements the exact constants (`AutoAggroProximityThreshold=40`, `AutoAggroRefreshInterval=5s`, `autoAggroMinInterval=1s`, sweep constants), singleton + sweepLoop with `//goroutine-guard:allow` annotation, `Admit`/`SweepStale`/`EvictTenant` all matching the plan's described semantics. Tests present including the plan's full table plus `ac165f745`'s exact-boundary additions. `RegisterEvictor` was not wired for the new gate — permitted per plan Step 3 ("otherwise the staleness sweep alone is sufficient and no wiring is added"), not a finding. |
| 13 | `AutoAggroHandleFunc` + registration | DONE | `socket/handler/auto_aggro.go` matches the plan's seam names (`autoAggroMirrorLookupFn`, `autoAggroEmitFn`) and body-order (character check → distance ≤ threshold → mirror lookup → field match → rate gate → emit) exactly; `main.go:915` registers `handlerMap[monstersb.AutoAggroHandle]` immediately after the `MobDropPickupRequestHandle` line (914). `template-symbol-check.sh` was run per-template; the 5 pre-existing `ChatGeneralChat` dangling-symbol failures reproduce identically on `main` (confirmed by checking out the pre-branch version of one such template) and are unrelated to this task — not a finding. |
| 14 | Lifecycle regression tests (FR-6.1/6.2) | DONE | `aggro_lifecycle_test.go` has `TestAggroChangedReissuesControlWithAggroSet` and `TestStartControlCarriesAggroThroughHandover`. The plan explicitly deferred wiring `handleStatusEventAggroChanged` through the package's own seams to a later fix if review found it — commit `b72b83c62` did exactly that (routes the handler through `monsterGetByIdFn`/`announceFn`, strengthening both handlers' tests to assert the aggro bool reaching the announce, not just the mirror side effect). This is the documented "Task 14 seam added later" substitution from the brief, confirmed intact. |
| 15 | Coverage manifest and deploy notes | DONE | `coverage-manifest.yaml` uses the repo's real `ops`/`versions`/`fields`/`out_of_scope` schema (accepted substitution) and was corrected post-hoc by `b3606cdac` to describe the actual wire layout (`Decode4(mobId)+Decode4(distance)`) rather than a stale draft description — the correction is accurate against the shipped codec. `deploy-notes.md` (79 lines) has the full ten-row PATCH table plus post-deploy checks and the three tuning-dial values. |

**Completion Rate:** 15/15 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task has direct, verifiable evidence in the diff.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---------|-------|-------|-------|
| `libs/atlas-packet` | PASS | PASS | `go test ./...` all packages ok, including new `monster/serverbound` codec tests. |
| `tools/packet-audit` | PASS | PASS | All internal packages ok. |
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS | All packages ok, including `monster` (15.4s) and `monster/information` (15.4s) which carry the new tests. |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | All packages ok, including `socket/handler` and `monster`. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. The branch has already absorbed three review-driven follow-up commits (`ac165f745`, `b72b83c62`, `b3606cdac`) that closed prior review findings; no further gaps identified in this pass. Note for the controller: HEAD has moved to `b3606cdac`, one commit past the `b72b83c62` cited in the audit brief — this was a small documentation-only correction to `coverage-manifest.yaml` and does not change any of the above findings.
