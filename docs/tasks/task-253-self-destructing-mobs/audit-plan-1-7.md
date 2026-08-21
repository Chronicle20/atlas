# Plan Audit — task-253-self-destructing-mobs (Tasks 1-7)

**Plan Path:** docs/tasks/task-253-self-destructing-mobs/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-253-self-destructing-mobs
**Base Branch:** main
**Commit Range:** d17404dbc..77fc9bc64

## Executive Summary

All seven tasks in this range are fully implemented as specified. Every
plan-mandated interface signature, code comment, and constant was found
verbatim at the cited (or near-identical) locations. Task 4 and Task 7
surfaced and correctly resolved a real arithmetic error in design.md's D2
rolling-deploy rationale (the `Hp > -1 || RemoveAfter > -1` predicate is
`true`, not `false`, under the old `{0,0,0}` sentinel) — this is documented
per the standing ruling and is not a finding. `go build ./...` and
`go test ./...` pass clean for all three touched modules
(`atlas-data`, `libs/atlas-packet`, `atlas-monsters`).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `atlas-data` — absent-`selfDestruction` sentinel | DONE | `services/atlas-data/atlas.com/data/monster/reader.go:206-216` returns `selfDestruction{Action:0, RemoveAfter:-1, Hp:-1}` on missing node, verbatim per plan Step 3. `reader_test.go:1267-1268` expects `{0,-1,-1}`. Commit `1adaf6c9c`. |
| 2 | `libs/atlas-packet` — dead-type constants and version-gated trailing `int32` | DONE | `libs/atlas-packet/monster/clientbound/destroy.go`: `DestroyType` constants (Disappear/FadeOut/Bomb/DestructByMiss/Swallow/SelfDestruct) at lines 26-41, `hasSwallowCharacterId` at 47-49 matching plan's gate exactly `(GMS>=92)\|\|JMS`, `Encode`/`Decode` gated on `destroyType == DestroyTypeSwallow && hasSwallowCharacterId(t)`. Tests `TestMonsterDestroySwallowGate` (line 102) and `TestMonsterDestroyNonSwallowTypesAreFiveBytes` (line 135) present with all 9 `packet-audit:verify` markers (lines 93-101). Commit `81c719bec`. |
| 3 | Packet coverage matrix — re-verify every `MonsterDestroy` cell | DONE | `docs/packets/evidence/gms_v92/monster.clientbound.MonsterDestroy.yaml` pinned with `address: "0x64bb90"`, `function: CMobPool::OnMobLeaveField`. All 9 evidence YAMLs (`gms_v61`…`jms_v185`) carry `verifies:` block referencing the two new tests. `docs/packets/gates.yaml:118-124` has the new `# ---- boundary v87 / v92 ----` group with `boundary: ">=92"`. `status.json` `KILL_MONSTER`/`gms_v92` cell reads `"state": "verified"` (confirmed at line ~25442, was `partial` before). `matrix --check` exits 0. Commit `cf248f49a`. |
| 4 | `atlas-monsters` — `information.SelfDestruction` value type | DONE | `information/model.go`: `SelfDestruction` struct, `NewSelfDestruction`, `Present()/Action()/RemoveAfter()/Hp()/OnHpThreshold()/OnTimer()`, `Model.SelfDestruction()` accessor — all present per plan. `builder.go` `SetSelfDestruction`. `rest.go:101` `sdPresent := rm.SelfDestruction.Hp > -1 \|\| rm.SelfDestruction.RemoveAfter > -1` matches the mandated formula exactly (no `action != 0` pattern-match found by grep, per prior review-task-4.md). `self_destruction_test.go` has all three named tests. One test row ("legacy absent {0,0,0}") deviates from the plan's literal table (plan said `wantPresent: false`; implementation correctly reports `true` per the actual arithmetic) — documented inline in the test with a comment explaining the design.md D2 discrepancy, later formally corrected in Task 7's commit `c5a1da3d0`. This is the standing ruling #2 item, not a new finding. Commit `29bf5bc65`. |
| 5 | `atlas-monsters` — `Registry.SelfDestruct` atomic transition | DONE | `monster/registry.go:503-525` `func (r *Registry) SelfDestruct(t tenant.Model, uniqueId uint32) (DamageSummary, error)` — CAS via `r.reg.Update`, `transitioned := cur.Hp > 0; cur.Hp = 0`, returns `Killed: transitioned`, matches plan verbatim. `self_destruct_registry_test.go` has all three named tests plus a bonus `TestRegistrySelfDestructConcurrentCallersExactlyOneWins` (goroutine/-race test, added later per Task 7's commit message, strictly additive). Commit `4c4a4c3f4`. |
| 6 | `atlas-monsters` — extract kill epilogue and `deathType` on the wire | DONE | `kafka.go:58-64` `DeathTypeUnset byte = 0`, `DeathTypeFadeOut byte = 1` with the specified rolling-deploy comment; `DeathType byte` field on both body structs (line 105 confirmed for one). `producer.go:31-33` `destroyedStatusEventProvider(m Model, deathType byte)`; `killedStatusEventProvider(..., deathType byte)` at line 149. `processor.go:601-622` `finalizeKill(m Model, killerId uint32, isBoss bool, revives []uint32, deathType byte)` matches plan body (a `GetSelfDestructTimerRegistry().Unregister` line was added later by Task 9 — expected, not a Task 6 deviation). Call sites: `processor.go:687` (`DeathTypeFadeOut` on ordinary kill) and `processor.go:1413` (`DeathTypeUnset` on `Destroy`). `producer_test.go` has all three named tests (`TestKilledBodyCarriesDeathType`, `TestDestroyedBodyCarriesDeathType`, `TestKilledBodyDeathTypeIsOmittedShapeCompatible`). Commit `7526324eb`. |
| 7 | `atlas-monsters` — HP-threshold detonation and `SelfDestruct` processor API | DONE | `processor.go:91-96` `SelfDestructTrigger` type with `TriggerThreshold/TriggerTimer/TriggerContact`; `monsterInformation` seam at line 103; `SelfDestruct` (line 1847) and `selfDestructFrom` (line 1873) match the plan's code blocks verbatim, including the FR-6.3 damage-leader fallback and the debug-level silent rejections. `damageCore` threshold wiring at `processor.go:631-635` (`var sd information.SelfDestruction; sd = ma.SelfDestruction()`) and the guard at line 691-695 (`if !killed && sd.OnHpThreshold() && int64(last.Monster.Hp()) <= int64(sd.Hp())`) exactly matches plan Step 5. Also fixes `Model.DamageLeader()` (`model.go:232-245`) to return 0 instead of panicking on no damage entries, and corrects design.md D2's rolling-deploy rationale (see commit `c5a1da3d0` message) — both are in-scope fixes surfaced by this task's own test-writing, not scope creep. Commit `c5a1da3d0`. |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All seven tasks in this range are fully implemented with evidence matching the plan's specified files, function signatures, and test names.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-data (services/atlas-data/atlas.com/data) | PASS | PASS | `go test ./monster/...` → `ok atlas-data/monster 0.060s` |
| libs/atlas-packet | PASS | PASS | `go test ./monster/...` → all four subpackages `ok` |
| atlas-monsters (services/atlas-monsters/atlas.com/monsters) | PASS | PASS | `go test ./monster/...` → `ok atlas-monsters/monster 22.724s`, `ok atlas-monsters/monster/information 15.662s`, plus consumable/drop packages ok |

`go run ./tools/packet-audit gate-lint --check` reports 38 pre-existing raw-comparison sites in `libs/atlas-packet`, none of them newly introduced by Tasks 1-7 (grepped and confirmed all listed sites are outside `monster/clientbound/destroy.go`) — per standing ruling #1, not a finding.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this range; overall branch verdict depends on the Tasks 8-13 shard)

## Action Items

None. Tasks 1-7 require no fixes.
