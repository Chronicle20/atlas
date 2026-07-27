# Plan Audit — task-151-brawler-mp-recovery

**Plan Path:** docs/tasks/task-151-brawler-mp-recovery/plan.md
**Audit Date:** 2026-07-27
**Branch:** task-151-brawler-mp-recovery
**Base Branch:** main (merge-base cdfb71aa3)

## Executive Summary

All 4 plan tasks are fully implemented and verified against the actual diff (`cdfb71aa3..HEAD`), not just the commit-message mapping. The plan's checkboxes in `plan.md` are left unchecked, but this is a docs-hygiene gap only — every file, function signature, and test named in the plan exists in the tree with matching (or superset) behavior. `go build`, `go vet`, and `go test -race` for `data/skill/effect` and `skill/handler/mprecovery` pass locally (re-run independently during this audit, not just trusted from `.superpowers/sdd/task-4-report.md`). `tools/redis-key-guard.sh` exits 0. Working tree is clean. No stubs, TODOs, or deferred work found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `Y()` getter on skill effect model | DONE | `data/skill/effect/model.go:157-160` adds `func (m Model) Y() int16 { return m.y }` exactly as specified. `model_test.go` `TestModelY` (commit 0924a599d) matches the plan's test verbatim. Fix commit 8d1efbbec additionally restores 3 pre-existing Shadow Stars tests (`TestModelBulletCount`, `TestModelBulletConsume`, `TestExtractBulletFields`) that a naive append would have clobbered — a correctness improvement over the plan's literal instructions, not a deviation from intent. |
| 2 | `Amounts` formula (pure function) | DONE | `skill/handler/mprecovery/formula.go` (commit d81e5a386) matches the plan's signature `func Amounts(maxHp uint16, x int16, y int16) (int16, int16)` and clamp/x<=0 semantics exactly. `formula_test.go` includes all 10 plan-specified table cases plus 2 additional clamp-ordering cases added in c72d9c18f ("mpGain from unclamped hpLost", "mpGain-only clamp: hpLost in range, mpGain overflows") — a superset, not a reduction. c38e8c62c reworded the doc comment from "Cosmic SpecialMoveHandler.java:118-124" to "WZ-verified v83 formula" — a citation-honesty fix consistent with CLAUDE.md's "No Cosmic citations in code comments" rule; formula/behavior unchanged. |
| 3 | MP Recovery handler with seams + registration | DONE | `skill/handler/mprecovery/mprecovery.go` (commit 9b4b9d44f) implements `init()` → `channelhandler.Register(skill2.BrawlerMPRecoveryId, Apply)`, package-level seams `loadCaster`/`changeHP`/`changeMP`, and `Apply` with the exact curried signature and control flow specified in the plan (load caster → `Amounts` → skip-if-zero → `changeHP` → skip-if-zero-mpGain → `changeMP`). `mprecovery_test.go` contains all 8 tests named in the plan verbatim (`TestMPRecoveryRegistered`, `TestMPRecoveryHappyPath`, `TestMPRecoveryCasterLoadError`, `TestMPRecoveryChangeHPError`, `TestMPRecoveryChangeMPError`, `TestMPRecoveryBadDataSkips`, `TestMPRecoveryZeroMpGainSkipsChangeMP`, plus `TestAmounts` from Task 2 in the same package). `skill2.BrawlerMPRecoveryId = Id(5101005)` confirmed live at `libs/atlas-constants/skill/constants.go:3199`. `character.Model.MaxHp()` confirmed at `character/model.go:136`; `ChangeHP`/`ChangeMP` confirmed at `character/processor.go:276,284`. |
| 4 | Production registration wiring + full verification | DONE | `skill/handler/registrations/registrations.go` (commit dccf26a0e) adds `_ "atlas-channel/skill/handler/mprecovery" // Brawler MP Recovery — task-151` at the correct alphabetical position (between `hide` and `mysticdoor`), matching the plan exactly. Confirmed the dispatch path is live: `skill/handler/common.go:143-148` (`UseSkill`) ends with `if h, ok := Lookup(skill2.Id(info.SkillId())); ok { h(l)(ctx)(wp, f, characterId, info, e) }`, so any `init()`-registered handler — including `mprecovery`'s — fires on every skill cast without per-skill wiring. Gates in `.superpowers/sdd/task-4-report.md` show build/vet/test-race/redis-guard all exit 0; independently re-run during this audit (see Build & Test Results) with the same result. |

**Completion Rate:** 4/4 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None found. All 4 tasks have direct code evidence, matching interfaces, and passing tests. The plan's "Out of Scope" section (Chakra 4211001, packet/opcode changes, atlas-data/libs changes, low-HP cast rejection, `docker buildx bake`) was correctly honored — no changes outside `services/atlas-channel/atlas.com/channel/` and no `go.mod` touch, consistent with the plan's Global Constraints.

Two commits go beyond the plan's literal text but both are within-task correctness/hygiene fixes, not scope creep or silent deferral:
- 8d1efbbec restores 3 tests the plan's literal "append after X()" instruction would have accidentally dropped (a bug the plan itself didn't anticipate; the executor caught and fixed it in the same task).
- c38e8c62c removes a "Cosmic SpecialMoveHandler.java" citation per CLAUDE.md's "No Cosmic citations in code comments" rule, replacing it with "WZ-verified" — consistent with project convention, not a plan task.
- c72d9c18f adds 2 extra clamp-ordering test cases to `TestAmounts`, strengthening rather than weakening Task 2's coverage.

None of these represent skipped or deferred plan work.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel (`go build ./...`) | PASS | — | Re-run from `services/atlas-channel/atlas.com/channel/`, exit 0, no output. |
| atlas-channel (`go vet ./...`) | — | PASS | Exit 0, no output. |
| atlas-channel `data/skill/effect` (`go test -race`) | — | PASS | `TestModelBulletCount`, `TestModelBulletConsume`, `TestExtractBulletFields`, `TestModelY` all PASS; `ok atlas-channel/data/skill/effect`. |
| atlas-channel `skill/handler/mprecovery` (`go test -race`) | — | PASS | All 8 tests PASS (`TestAmounts` + 7 handler tests); `ok atlas-channel/skill/handler/mprecovery`. |
| atlas-channel `skill/handler/{heal,healdispel,hide,mysticdoor,resurrection}` (`go test -race`) | — | PASS | Sibling per-skill handler packages unaffected by the change; all `ok`, confirming the shared registry wasn't broken. |
| `tools/redis-key-guard.sh` | — | PASS | Exit 0; no violation lines (informational scan-progress only). |

`docker buildx bake` was not run — correctly not required per CLAUDE.md gate 4 and the plan's explicit note, since no `go.mod` was touched by this task.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Docs hygiene only, non-blocking) `docs/tasks/task-151-brawler-mp-recovery/plan.md` checkboxes are all still `- [ ]` despite every step being completed. Recommend checking them off (or noting completion) before merge for future auditability, but this does not block the PR — the audit process explicitly treats it as a nit per the task instructions.
