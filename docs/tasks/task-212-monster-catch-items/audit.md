# Plan Audit — task-212-monster-catch-items

**Plan Path:** docs/tasks/task-212-monster-catch-items/plan.md
**Audit Date:** 2026-08-10
**Branch:** task-212-monster-catch-items
**Base Branch:** main

## Executive Summary

All 13 plan tasks have corresponding commits on the branch (`b531cb3db`..`48cb55054`) with matching file-level changes, tests, and documentation. The plan.md checkboxes (`- [ ]`) were never checked off during execution, but this is cosmetic — every task's file structure, interfaces, and acceptance behavior described in the plan is present in the diff, verified against actual commit contents rather than the checkbox state. All six affected Go modules build, vet, and test clean; all seven mandatory guard scripts pass. No TODOs, stubs, or silent-drop paths were found; three legitimate mid-flight plan amendments (documented in commit messages, not silent deviations) tightened the catch resolution ladder's failure-handling semantics beyond what the original brief specified.

**Update (post-audit, same session):** `docker buildx bake atlas-monsters atlas-consumables atlas-channel` was run from the worktree root after this audit's initial pass (which had flagged it as not-run) and all three targets built and exported cleanly (exit 0). `go run ./tools/packet-audit matrix --check`, `tools/lint.sh --check` (with `nvm use 22` sourced — the atlas-ui half needs Node 22, not a false-fail), `tools/skill-job-id-guard.sh`, and `tools/buff-duration-guard.sh` were also re-run against current HEAD (post-Task-13 commit `48cb55054`) and all passed clean. This resolves the audit's blocking recommendation below — see the Task 13 report at `.superpowers/sdd/plan/task-13-report.md` for the full gate table with exit codes.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `UseCatchItem` serverbound codec | DONE | `6f9b52da4`; `libs/atlas-packet/monster/serverbound/use_catch_item.go` (73 lines) + test (55 lines), no version gate, matches plan's field layout (`updateTime`, `slot`, `itemId`, `monsterUniqueId`) |
| 2 | Registry entries + packet-audit fname linkage | DONE | `fa21e9873`; `docs/packets/registry/gms_v{48,61,72,79}.yaml` + `tools/packet-audit/cmd/run.go` `candidatesFromFName` case added; v61 opcode correction documented and justified in commit body (a plan-compliant, sourced fix, not scope creep) |
| 3 | Route `USE_CATCH_ITEM` in all ten seed templates | DONE | `5c6c61b1e`; all ten `template_*_1.json` files gained the handler entry at the plan's specified opCode; `socket/corpus_test.go` count bumped 3075→3085 |
| 4 | `CMobPool::OnMobPacket` uniqueId prefix unconditional | DONE | `389c6b3ea`; `catch_monster.go`, `inc_mob_charge_count.go`, `monster_special_effect_by_skill.go` — `legacyMobPoolPrefix` deleted, fixtures updated with v83/v95/v92/v48 pins |
| 5 | `CatchMonsterWithItem` gains `uniqueId`; v48 drops result byte | DONE | `c7ea88432`; codec + writer (`socket/writer/catch_monster_with_item.go`) updated with `MajorAtLeast`/`v48CatchByItemNoResult` gate; plan's own byte-fixture arithmetic typo caught and corrected (documented in commit) |
| 6 | Writer routes for v48/v92 clientbound catch cells | DONE | `d73ce0862`; `template_gms_48_1.json` (CatchMonster/CatchMonsterWithItem), `template_gms_92_1.json` (BridleMobCatchFail/CatchMonster/CatchMonsterWithItem) — matches design §8's v92 gap correction; registry entries added for v48 172/173 |
| 7 | Promote coverage-matrix cells | DONE | `bfa98a0ac`; all USE_CATCH_ITEM (10 versions) + CATCH_MONSTER/CATCH_MONSTER_WITH_ITEM/BRIDLE_MOB_CATCH_FAIL cells promoted with evidence + audit reports; no "not promoted, and why" note needed in context.md since all cells were successfully promoted (grep confirms none present) |
| 8 | Atomic delete-and-claim in `libs/atlas-redis`, `ClaimMonster` | DONE | `b0c7fd533`; `libs/atlas-redis/registry.go:73` `RemoveExisting`, `services/atlas-monsters/.../monster/registry.go:577` `ClaimMonster`, concurrency test proving exactly-one-winner |
| 9 | atlas-monsters catch-item data client | DONE | `efbc6a8fc`; `monster/consumable/{requests.go,rest.go,model.go,processor.go,mock/processor.go}` created, uncached per design §7 |
| 10 | atlas-monsters `CATCH` command + resolution ladder | DONE (with documented amendments) | `00d469c1e` base implementation, then `fa1f342cf`, `9e643af49`, `9eff5ea6a` fix commits close every silent-drop path the original brief left open (bare 0-return overload, catch-item lookup failure, ClaimMonster's Redis-error branch). Each fix is described in its commit body as a "plan amendment approved by user" — I could not independently verify a user approval event occurred; treating this as a disclosed, well-reasoned deviation rather than a silent one. `monster/catch.go` final state (read directly) shows no remaining silent-drop exit — every return path emits either the success triple or a failure pair. |
| 11a | atlas-consumables data getters + catch classification | DONE | `248406616`; `data/consumable/model.go` gains 8 getters; `libs/atlas-constants/item/constants.go:41` `ClassificationConsumableCatchItem = Classification(227)` |
| 11b | atlas-consumables request path | DONE | `79f90e7e4`; `catchdelay/registry.go` (Redis useDelay gate), `consumable/catch.go` (`RequestCatchMonster`), producer/consumer wiring; commit body documents 11b+11c as one compile unit — resolved by 11c landing immediately after |
| 11c | atlas-consumables resolution — commit/grant/cancel | DONE | `d00f35d2d` + `4b9f2ff59` (fixes dangling reservation on CATCH command produce failure); `catchResolvedValidator`, `catchResolutionHandler` implemented |
| 12a | atlas-channel handler + command emitter | DONE | `df210a4ff`; `socket/handler/monster_catch_item_use.go` + test, forwards `REQUEST_CATCH_MONSTER`, no unlock sent here (correctly deferred to 12b) |
| 12b | atlas-channel rendering — effects, failure, unlock | DONE | `59f49b2e8`; `kafka/consumer/monster/consumer.go` (99 lines) renders CAUGHT/CATCH_FAILED, maps cause to wire reason, unlocks on every path including UNRESOLVED |
| 13 | Topic configuration, documentation, verification sweep | DONE | `48cb55054`; `deploy/k8s/base/env-configmap.yaml:136` `EVENT_TOPIC_MONSTER_CATCH`, both overlays carry suffixed forms (`-main` / `-PLACEHOLDER_ATLAS_ENV`) — no unsuffixed-fallback trap; `services/atlas-monsters/docs/kafka.md` (+88 lines) and `services/atlas-consumables/docs/kafka.md` (+73 lines) document CATCH/CATCH_RESOLVED/CAUGHT/CATCH_FAILED; both `README.md` topic tables updated |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 13 tasks and their sub-steps have corresponding code, tests, and documentation on the branch.

One item to flag rather than a gap: Task 10's resolution-ladder fix commits (`fa1f342cf`, `9e643af49`, `9eff5ea6a`) describe themselves as "plan amendments approved by user," diverging from the brief's original Kill-style silent-drop text for data-lookup failures. I read the final `monster/catch.go` directly and confirmed the resulting behavior is strictly more correct than the plan's literal text (no dangling reservations, no wedged clients) and is fully covered by `catch_test.go`. I cannot independently verify the claimed user approval occurred, but the change is transparently documented in the commit history rather than silently made, and it improves on FR compliance (every terminal path unlocks — PRD acceptance criterion). Recommend confirming with the task owner that this amendment was in fact approved before merge.

## Build & Test Results

| Service/Module | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | PASS | all packages `ok` |
| `libs/atlas-redis` | PASS | PASS | PASS | includes `RemoveExisting` concurrency test |
| `libs/atlas-constants` | PASS | PASS | PASS | |
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS | PASS | `monster` pkg 14.1s, `monster/information` 15.4s |
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS | PASS | `consumable` pkg 19.3s |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS | full package list `ok`, no failures |

| Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` | PASS |
| `tools/goroutine-guard.sh` | PASS |
| `tools/template-opcode-order-guard.sh` | PASS — 22 arrays ascending |
| `tools/template-duplicate-binding-guard.sh` | PASS — no duplicate bindings |
| `tools/template-movement-types-guard.sh` | PASS — 54 handlers, 11 templates valid |
| `tools/service-registration-guard.sh` | PASS |

**Now run (post-audit):** `docker buildx bake atlas-channel atlas-consumables atlas-monsters` — all three targets PASS (exit 0). `go run ./tools/packet-audit matrix --check` — PASS. `tools/lint.sh --check` (with Node 22 via nvm) — PASS, no drift. `tools/skill-job-id-guard.sh` — PASS. `tools/buff-duration-guard.sh` — PASS. All re-verified against current HEAD (`48cb55054`), not just Task 7's historical commit assertion.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY (the previously-blocking item — mandatory `docker buildx bake` for the three touched services — has since been run and confirmed passing; see "Update" note above)

## Action Items

1. ~~Run `docker buildx bake atlas-channel atlas-consumables atlas-monsters`~~ — DONE, all three targets PASS.
2. ~~Run `go run ./tools/packet-audit matrix --check`, `tools/lint.sh --check`, `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh` against current HEAD~~ — DONE, all PASS.
3. Confirm with the task owner that the Task 10 resolution-ladder amendments (rounds 2–3, commits `fa1f342cf`/`9e643af49`/`9eff5ea6a`) were in fact approved, since this audit could not independently verify the approval event referenced in those commit messages — though the resulting code is demonstrably correct and well-tested. (Still open — not a code/build gate, needs a human answer.)
4. Check off the plan.md checkboxes (cosmetic only) so the document accurately reflects execution state for future readers. (Still open — cosmetic, non-blocking.)
