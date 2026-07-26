# Plan Audit — task-149-pickpocket-meso-spawn

**Plan Path:** docs/tasks/task-149-pickpocket-meso-spawn/plan.md
**Audit Date:** 2026-07-26
**Branch:** task-149-pickpocket-meso-spawn
**Base Branch:** main (commit range `3c499bad4..92f4e5d3c`, 7 commits)

## Executive Summary

All 7 plan tasks (40 checklist steps) were faithfully implemented, one commit per task, in the exact order and with the exact function signatures, constants, and field-parity choices the plan specified. `go build ./...`, `go test ./... -count=1`, `go test -race ./...`, and `go vet ./...` are all clean in `services/atlas-channel/atlas.com/channel`. The `// TODO apply Pick Pocket` line is gone and no other TODO was touched. The only gap is process-level, not functional: `plan.md`'s 40 checkboxes are all still `- [ ]` despite the work being fully committed — the plan file itself was never checked off. Task 7 Step 8 (live-tenant legacy-version acceptance check) is correctly unrunnable headlessly and is flagged below as an outstanding manual check per the audit brief, not a plan-adherence failure.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Generalize `mpEaterShouldProc` into `shouldProc` | DONE | `character_attack_common.go:216-221` defines `shouldProc`; `character_attack_mp_eater_test.go:8-27` has `TestShouldProc` (renamed, 7 cases, byte-identical to plan). Call site at `mpEaterTryProc` verified using `shouldProc`. Commit `10f5c9653`. |
| 2 | Pure helpers `pickPocketWhitelisted` / `pickPocketMesoAmount` | DONE | `character_attack_common.go:226-258`; whitelist map uses `skill3.*Id` constants (RogueDoubleStabId=4001334, BanditSavageBlowId=4201005, ChiefBanditAssaulterId=4211002, ChiefBanditBandOfThievesId=4211004, ShadowerAssassinateId=4221001, ShadowerTauntId=4221003, ShadowerBoomerangStepId=4221007 — all confirmed present in `libs/atlas-constants/skill/constants.go:3143-3187`). Tests `TestPickPocketWhitelisted` (12 subtests) and `TestPickPocketMesoAmount` (8 subtests) in `character_attack_pick_pocket_test.go:21-97`, all passing. Commit `1bc3b6daa`. |
| 3 | `SPAWN` producer in `drop` package | DONE | `kafka/message/drop/kafka.go:16` adds `CommandTypeSpawn = "SPAWN"`; `SpawnCommandBody` at `:38-52` matches atlas-drops' `CommandSpawnBody` (`services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:121-136`) field-for-field minus `EquipmentData`, verified by direct diff. `drop/producer.go:34-61` adds `SpawnMesoCommandProvider`. `drop/processor.go:19` adds `SpawnMeso` to the `Processor` interface, `:58-60` implements it on `*ProcessorImpl`. `drop/producer_test.go` (`TestSpawnMesoCommandProvider`) passes. As a necessary consequence (not itself a plan step, but required for the interface to still compile), `drop/mock/processor.go` was updated with a `SpawnMesoFunc` field — a correct, minimal knock-on change. Commit `e375d13ab`. |
| 4 | Per-attack state `pickPocketState` / `pickPocketResolveState` | DONE | `character_attack_common.go:263-328`, byte-for-byte matching the plan's specified gating order (whitelist → buff lookup → PICK_POCKET stat scan → maxmeso>0 → effect lookup → prop>0). All 8 planned subtests present and passing (`character_attack_pick_pocket_test.go:100-250`), including `TestPickPocketResolveState_NonWhitelistedSkillMakesNoLookups` which proves zero I/O for non-whitelisted skills. Commit `7f158ad78`. |
| 5 | Widen `onDamageApplied` to carry `DamageInfo` | DONE | `damageInfoEntryDeps.onDamageApplied` signature at `character_attack_common.go:114` is `func(di packetmodel.DamageInfo, totalDamage uint32)`; invocation at `:200-208` passes `di` alongside the summed/clamped total. The three existing drain hook tests (`character_attack_drain_test.go:111,141,160`) were updated to the new signature and pass. The new `TestOnDamageApplied_CarriesPerLineDamages` (`character_attack_pick_pocket_test.go:252`) pins the per-line `di.Damages()` capability Pick Pocket depends on. Commit `1e9f4189e`. |
| 6 | Per-monster proc `pickPocketTryProc` | DONE | `character_attack_common.go:334-364`, matches the plan's spec exactly: disabled-state short-circuit, monster-fetch-error → Debugf+skip, per-damage-line `shouldProc` roll, `pickPocketMesoAmount`, `x := mon.X() + rand.Intn(100)-50`, emit-error → Errorf+continue. All 5 planned tests present and passing (`character_attack_pick_pocket_test.go:288-402`): prop=1.0 emits per line, prop=0 emits nothing, disabled state makes no lookups, monster-fetch error skips, emit error continues remaining lines. Commit `d3685070b`. |
| 7 | Wire into `processAttack`, remove TODO, run full gate | DONE | `character_attack_common.go:582-612`: `dp := drop.NewProcessor(l, ctx)`, `ppState := pickPocketResolveState(...)` resolved once before the `deps` construction; the `onDamageApplied` closure keeps the pre-existing MP Eater arm (`:603-605`) and drain arm (`:606-608`) **unmodified in intent** (both now read `di.MonsterId()` per Task 5) and adds the Pick Pocket arm (`:609-611`) gated on `ppState.enabled`. `grep -n "TODO apply Pick Pocket"` returns nothing — confirmed removed; the other 20 unrelated TODO lines in the same block (`:659-680`) are untouched. Commit `92f4e5d3c`. |

**Completion Rate:** 7/7 tasks (100%) by code evidence. Plan-file checkbox tracking: 0/40 steps checked (see Skipped/Deferred section — this is a documentation-sync gap, not a functional gap).
**Skipped without approval:** 0
**Partial implementations:** 0 (Task 7 Step 8 is an intentionally-deferred manual/live-runtime check, not a partial code implementation — see below)

## Skipped / Deferred Tasks

- **Task 7, Step 8 ("Version acceptance check")** — requires exercising a whitelisted attack against a live legacy-version tenant (e.g. GMS v72) and confirming meso drops spawn and are lootable in a running environment. This cannot be executed headlessly in this audit (no live k8s/game-client session available in this pass). This is **not a plan-adherence failure**: the proc is implemented version-agnostically (runs on the already-decoded `packetmodel.AttackInfo`, downstream of every version gate, with no version-specific branching added anywhere in the diff — confirmed by inspection of the full diff, no `Major`/version-comparison code was added in this task), and `TestOnDamageApplied_CarriesPerLineDamages` plus the full `pickPocketTryProc`/`pickPocketResolveState` unit suite already exercise the version-independent code path. **Outstanding action:** a manual/live acceptance pass on a legacy-version tenant is still owed before the feature can be called fully verified end-to-end in a running environment, per PRD §10 and design §1a.
- **plan.md checkbox state** — all 40 `- [ ]` boxes in the committed `plan.md` remain unchecked even though every corresponding commit exists on the branch and all corresponding tests pass. This is a documentation-hygiene gap (the executor never went back to check off `plan.md` as work landed), not evidence of incomplete implementation — the code-level cross-check above independently confirms each task's deliverable exists and works.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel (`services/atlas-channel/atlas.com/channel`) | PASS | PASS | `go build ./...` clean (no output). `go test ./... -count=1` — all packages `ok` or `[no test files]`, zero `FAIL`. `go test -race ./... -count=1` — zero `FAIL`. `go vet ./...` — clean (no output). All Pick Pocket-specific tests (`TestShouldProc`, `TestPickPocketWhitelisted`, `TestPickPocketMesoAmount`, `TestPickPocketResolveState_*` ×8, `TestOnDamageApplied_CarriesPerLineDamages`, `TestPickPocketTryProc_*` ×5, `TestSpawnMesoCommandProvider`) individually verified passing via `-run` filter. |

Per the audit brief, `docker buildx bake atlas-channel` and the repo-root guards (`tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`) were already run green by the implementer at HEAD and were not re-run in this pass.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (contingent on the outstanding live-tenant legacy-version acceptance check being run before/shortly after merge, and on optionally checking off `plan.md`'s boxes for documentation hygiene)

## Action Items

1. Run the Task 7 Step 8 live-tenant acceptance check (whitelisted attack, e.g. basic attack or Double Stab, with PICK_POCKET buff active, on a legacy-version tenant such as GMS v72) and confirm meso drops spawn at the monster and are lootable by the attacker. This is the one PRD §10 criterion not verifiable by static/unit means.
2. Optionally, check off the 40 `- [ ]` boxes in `docs/tasks/task-149-pickpocket-meso-spawn/plan.md` to reflect the actually-completed state of the branch (cosmetic; does not block merge).
