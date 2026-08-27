# Plan Audit — task-122-attack-path-snapshots (Tasks 8-14)

**Plan Path:** docs/tasks/task-122-attack-path-snapshots/plan.md
**Audit Date:** 2026-08-27
**Branch:** task-122-attack-path-snapshots
**Base Branch:** main (merge-base `2537d8a6a`)
**Scope:** Plan Tasks 8-14 only (sibling shard covers 1-7)

## Executive Summary

All seven tasks in this range (8-14) are fully and faithfully implemented against the plan's acceptance criteria, with no silently-dropped sub-requirements found. Task 12's shadow-verification metric — flagged by its own first-pass review as a "declared but never recorded" risk (the same failure mode Task 3 hit) — was genuinely fixed in a follow-up commit that now asserts `snapshotDivergenceTotal` via `testutil.ToFloat64`, confirmed present and correct at HEAD. Task 13's `Values` population was verified complete across every non-empty-`Updates` `statChangedProvider` call site in `character/processor.go` (18 sites populated, 2 correctly left `nil` for empty-Updates error paths), and every key used matches atlas-channel's `statValueKeys` table exactly, including the disclosed `AVAILABLE_SP` omission. Module-local `go build`/`go test` pass clean in both `atlas-channel` and `atlas-character` for all task-8-14-scoped packages.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 8 | Buff consumer snapshot handlers | DONE | `handleSnapshotBuffApplied`/`handleSnapshotBuffExpired` implemented and registered in `InitHandlers` (`kafka/consumer/buff/consumer.go:88-97`, `:513-548`); `UpsertBuff`/`RemoveBuff` wired; 3 tests incl. no-op-on-unbackfilled-character (`consumer_test.go:41,84,107`). |
| 9 | Skill-data in-process TTL cache | DONE | `data/skill/cache.go`, `metrics.go` created per spec (env kill-switch, positive/negative TTL, no singleflight, `EvictTenant`); `Processor.GetById` routes through `getByIdCached` (`processor.go:31`); `main.go:315` wires `dataskill.EvictTenant(tid)` in the evictor block. 7 required test cases present (`cache_test.go`). |
| 10 | Live-mirror X/Y extension + monster movement position feed | DONE | `LiveEntry.X/Y int16` added (`live_mirror.go:35-36`), `LiveEntryFromModel` maps them, `UpdatePosition` is update-only (`:178-198`); `movement/processor.go:238` feeds the mirror synchronously right after the fold, before the async goroutine — matches the "synchronous, wire-invariant-preserving" placement rule. Tests present for both mapping and update-only semantics, plus `TestForMonster_FeedsMirrorPosition`. |
| 11 | Attack-path adoption | DONE | `attackMonsterByIdFn` seam + `buildMonsterResolver` (memoized, mirror-first, REST-fallback-with-backfill) implemented (`character_attack_common.go:104-135`); `damageInfoEntryDeps.getMonster` retyped to `func(uint32) (monster.LiveEntry, error)`; `mpEaterTryProc` retyped and reads `mon.MaxMp/Mp` from the resolver (`:603-661`); `processAttack` reads the character via `sp := snapshot.NewProcessor(...); c, err := sp.Get(...)` (`:875-876`), with `cp` retained only for HP/MP command emission as specified. Projectile buffs are routed through `sp.GetBuffs` via a `loadBuffs` closure parameter rather than a `ProjectileProcessorImpl` struct field as literally specified — functionally equivalent to the plan's intent (buffs served from the snapshot), and covered by `TestWriterEquivalence_SnapshotComposedModel` and `TestSnapshotStalenessWindow_SkillLevelChangesBetweenAttacks`. `drainTryHeal`/`pickPocketTryProc`/`mortalBlowDeps` remain REST-backed via a separate `buildMonsterModelResolver` — this is controller ruling R7 (recorded in `.superpowers/sdd/plan/task-11-review.md`), an explicit, reasoned, mid-execution scope narrowing (monster HP is not mirrored — out of this plan's sizing), not a silent drop. |
| 12 | Shadow verification (env-gated sampling) | DONE | `character/snapshot/shadow.go` implements `maybeShadow`/`shadowCompare`/`compareProjection` exactly per spec, including the `±100px` position tolerance band and the env-gated `CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE` kill-switch (default off). Wired at both `Get` fast-path and full-hit slow-path returns (`processor.go:61,151`), verified structurally mutually exclusive (no double-sampling) per `task-12-review.md` §2. The metric-completeness gap the first-pass review flagged as BLOCKING — "no test asserts `snapshotDivergenceTotal` actually incremented" (`task-12-review.md` §5) — was fixed: `TestShadow_SamplesAndCountsDivergence` (`shadow_test.go:33-68`) now reads `testutil.ToFloat64(snapshotDivergenceTotal...)` before/after and asserts the delta is exactly 1. Confirmed present and correct by direct read at HEAD. Shadow goroutine routes through `routine.Go` (fixed in a later commit, `6a24a2bd5`). |
| 13 | atlas-character — populate Values on every STAT_CHANGED emission | DONE | Every non-empty-`Updates` `statChangedProvider` call site in `character/processor.go` was checked by direct enumeration (26 call sites total): 2 correctly retain `nil` with `[]stat.Type{}` (AP-distribute error paths at what are now lines 1067/1131 — matches plan's explicit "leave nil" instruction), the remaining 24 all carry a populated `map[string]interface{}`. Spot-verified the composite/dynamic sites (level-up growth, job-change growth, ResetStats, RebalanceAP) — every conditionally-appended `stat.Type` has a matching key in `values`. All key names (`job`, `hair`, `face`, `skin`, `experience`, `level`, `meso`, `fame`, `available_ap`, `hp`, `max_hp`, `mp`, `max_mp`, `strength`, `dexterity`, `intelligence`, `luck`) match atlas-channel's `statValueKeys` table (`character/snapshot/registry.go:246-264`) exactly. `AVAILABLE_SP` is intentionally absent from that table (disclosed exception, confirmed accepted). 10 test functions in `stat_values_test.go` drive the hot paths named in the plan plus level-up/job-change/reset/rebalance. |
| 14 | Full verification gate | DONE | `context.md` updated with component→handler map, metric names, env-var defaults, and a genuine re-swept "R11" zero-REST audit (not a narrow 3-grep check) that found and disclosed a 4th REST read (`energyReannounceAuthoritative`) beyond the plan's original three greps, reasoned as pre-existing/out-of-scope rather than silently whitelisted. `tools/redis-key-guard.sh` was run task-locally per report; `-race`/docker-bake were deferred to the repo-wide `tools/verify.sh` gate (per this audit's own instructions, already confirmed flagless-green at HEAD). |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

Note: the plan.md checkboxes for Tasks 8-14 are all still literally `- [ ]` (unchecked) at HEAD — a documentation-hygiene gap, not a completion gap; every step's evidence is present in the landed diff and reports as detailed above.

## Skipped / Deferred Tasks

None. No task in this range is SKIPPED or PARTIAL.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel (`character/snapshot`, `data/skill`, `kafka/consumer/buff`, `monster`, `movement`, `socket/handler`) | PASS | PASS | `go build ./...` clean; `go test ./character/snapshot/... ./data/skill/... ./kafka/consumer/buff/... ./monster/... ./movement/... ./socket/handler/... -count=1` all `ok`. |
| atlas-character (`character`) | PASS | PASS | `go build ./...` clean; `go test ./character/... -count=1` `ok` (10.4s). |

(Flagless `tools/verify.sh`, which additionally covers `-race` and docker bake across the whole repo, was already confirmed green at HEAD per the dispatch instructions and was not re-run by this audit.)

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; defer to the sibling 1-7 shard's findings for overall branch verdict)

## Action Items

None required for Tasks 8-14.
