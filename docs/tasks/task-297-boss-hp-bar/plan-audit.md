# Plan Audit — task-297-boss-hp-bar

**Plan Path:** docs/tasks/task-297-boss-hp-bar/plan.md
**Audit Date:** 2026-09-04
**Branch:** task-297-boss-hp-bar
**Base Branch:** main (diff range 31a791e3a..5b6ed61a1)

## Executive Summary

All 8 plan tasks were implemented and their code matches the plan text closely, including
both pre-accepted deviations (Task 7's `ControlCharacterId(999)` fixture, and Task 8's
divergence finding on `template_gms_12_1.json` carrying zero `FieldEffect`-family writer
entries). `go build ./...` is clean and every task's own `go test` scope passes *without*
`-race`. However, the `-race` runs mandated by Task 5's and Task 6's own Step 3 verification
(`go test -race ./kafka/consumer/monster/...`) fail on a real, reproducible data race: two
`routine.Go` goroutines inside the pre-existing `handleStatusEventDamaged` body write to the
same shared `err` variable (`consumer.go:288` and `:308`). This bug predates task-297 but was
never previously exercised under `-race` because no prior test invoked
`handleStatusEventDamaged` directly; Task 5's new `TestHandleStatusEventDamaged_BossHpGauge`
("boss damage broadcasts" case, which combines `Boss: true` with
`DamageSourceMonsterAttack`) is the first test that runs both goroutines concurrently and
trips it. This is a real regression risk introduced into the test suite by task-297 (the new
test now fails under race detection) and blocks the plan's own Step 3 gate for Tasks 5 and 6.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Widen `data/monster` projection + mock | DONE | `data/monster/rest.go`, `model.go`, `mock/processor.go` (commit 88dc77803); `rest_test.go` has `TestExtract`, `TestTransformRoundTrip`, new `TestUnmarshalDataPayload` exactly as specified |
| 2 | TTL cache for `data/monster.GetById` | DONE | `data/monster/cache.go` (192 lines, no `recordCache`/metrics calls, correct env names/bounds); `cache_test.go` has all 7 named subtests (`PositiveHitAvoidsSecondFetch` … `ConcurrentAccess`); `processor.go` `NewProcessor` returns `Processor` with `var _ Processor = (*ProcessorImpl)(nil)`; `main.go:320` adds `datamonster.EvictTenant(tid)` |
| 3 | `monster/bosshp` package | DONE | `bosshp.go` matches the plan's `Gauge`/`Resolver`/`AnnounceOperator` signatures verbatim; `bosshp_test.go` has `TestResolve` (5 cases) and `TestBossHpBodyBytes` (both sub-cases, byte sequence and `b[0]==99` sentinel match plan exactly) |
| 4 | Carry `MaxHp` on live mirror | DONE | `live_mirror.go` adds `MaxHp uint32` and seeds it in `LiveEntryFromModel`; `live_mirror_test.go` adds `TestLiveEntryFromModel_SeedsMaxHp` matching the plan's exact fixture and assertions |
| 5 | Damage hook (`bossHpBroadcaster` + `DAMAGED`) | PARTIAL | Code matches plan (`consumer.go:268→monsterGetByIdFn`, `bossHpBroadcaster` var, call site inside `handleStatusEventDamaged`); all 6 named test cases present in `consumer_test.go`. **But** the plan's own Step 3 (`go test -race ./kafka/consumer/monster/...`) fails: `TestHandleStatusEventDamaged_BossHpGauge/boss_damage_broadcasts_(FR-4)` and `/echo-suppressed_source_still_broadcasts_(FR-6)` both trip a genuine data race at `consumer.go:288` vs `:308` (shared `err` var written by two concurrent `routine.Go` closures). See Skipped/Deferred section |
| 6 | Death hooks (`KILLED`/`DESTROYED`) | DONE (build/race in this file also affected — see note) | `consumer.go` diff matches plan exactly (hoisted `t`, `Lookup`-before-evict sequence in both handlers); `TestHandleStatusEventDeath_BossHpGaugeEmpties` has all 4 named subtests with correct expected records; pre-existing `TestHandleStatusEventDestroyedAndKilled_RemoveMirrorEntry` unchanged and still passes. Task 6 code itself introduces no new race; the `-race` failure is inherited from Task 5 in the same package/test binary |
| 7 | Field-entry hook (`bossHpSenderFn`) | DONE | `consumer.go` (map package) matches plan: `doorAnnounce` reused for existing Spawn/Control announces, `bossHpSenderFn` var, call site after Control branch; `consumer_test.go` has all 6 named cases including the accepted `ControlCharacterId(999)` deviation, explicitly commented in the test to explain the fixture choice; `go test -race ./kafka/consumer/map/...` passes clean |
| 8 | Record `gms_12`/`gms_92` follow-up doc | DONE | `docs/tasks/task-297-boss-hp-bar/follow-up-field-effect-writer-gms12-gms92.md` created; independently re-verified `grep` evidence matches the doc's claims exactly (9/11 templates carry `FieldEffect`; `gms_92` carries only `FieldEffectWeather` at line 2540; `gms_12` has zero FieldEffect-family hits); doc explicitly records the divergence from design.md OQ1 (gms_12 has *no* fallback writer at all, contrary to the design's claim); `design.md` was confirmed untouched since its original commit (980430696) |

**Completion Rate:** 8/8 tasks (100%) code-complete against plan text
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 5, due to inherited test-suite race failure; Task 6 flagged only because it shares the failing test binary)

Note: every task/step checkbox in `plan.md` is still `- [ ]` (unchecked), including the
"Final verification" section — this is a plan-hygiene gap (the plan document was never
marked up to reflect completed work) but is not evidence of incomplete implementation; code
evidence above establishes each task was actually done.

## Skipped / Deferred Tasks

None of the 8 tasks were skipped or deferred. The one substantive gap is a test-suite
regression surfaced by Task 5's new test, not a missing implementation:

- **Task 5 / Task 6 — `-race` failure in `kafka/consumer/monster`.** `handleStatusEventDamaged`
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:263-303`, a
  function the plan explicitly says not to otherwise touch — "Do not touch the existing
  `announcer` goroutine or the `MonsterDamage` goroutine") declares `err` once at line 273
  (`m, err := monsterGetByIdFn(...)`) and then two independent `routine.Go` closures both
  assign to that same captured `err` — the health-announce goroutine at `consumer.go:288`
  (`err = _map.NewProcessor(...).ForSessionsInMap(...)`) and, when
  `shouldEchoDamagePacket(e.Body.DamageSource)` is true, the damage-echo goroutine at
  `consumer.go:308` (`err = _map.NewProcessor(...).ForSessionsInMap(...)`). Both goroutines
  can run concurrently for `DamageSourceMonsterAttack`. This is pre-existing production code,
  unmodified by task-297's diff (confirmed by diffing `ee733852d` (task 2, before task 5)
  against the pre-task-297 base — the shared-`err` shape is identical). No test before
  task-297 called `handleStatusEventDamaged` directly (`TestShouldEchoDamagePacket` only
  tests the standalone helper), so the race was latent. Task 5's new
  `TestHandleStatusEventDamaged_BossHpGauge` is the first test to invoke the handler with
  `Boss: true` and `DamageSource: monster2.DamageSourceMonsterAttack` synchronously, which is
  exactly the combination that fires both goroutines and trips `-race`. Reproduced twice,
  isolated with `-run TestHandleStatusEventDamaged_BossHpGauge` (no contention from any
  concurrently-running `verify.sh`, since the failure is confined to this one test function
  and reproduces in isolation).
  - Impact: the plan's own Step 3 command for both Task 5 and Task 6
    (`go test -race ./kafka/consumer/monster/...`) does not exit 0, so neither task's stated
    acceptance gate is currently met, and the flagless `tools/verify.sh` (which runs with
    `-race`, per project convention) will not pass this package either.
  - This is pre-existing code the task was not supposed to modify, but the task's chosen test
    fixture is what exposes it in CI going forward; either the fixture needs adjusting to
    avoid exercising the concurrent-goroutine path, or (more correctly, since this is a real
    production bug) the shared `err` needs to be given goroutine-local scope. Neither line was
    changed in this diff, so no fix is present.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel (`data/monster`) | PASS | PASS | `go test -race ./data/monster/... ./socket/handler/...` — ok |
| atlas-channel (`monster/bosshp`) | PASS | PASS | `go test ./monster/bosshp/...` — ok |
| atlas-channel (`monster`) | PASS | PASS | `go test ./monster/...` — ok |
| atlas-channel (`kafka/consumer/monster`) | PASS | **FAIL** | `go test -race ./kafka/consumer/monster/...` — `TestHandleStatusEventDamaged_BossHpGauge` fails with `WARNING: DATA RACE` at consumer.go:288/308; also observed an apparently-unrelated `TestBridleFailReason/SPECIES_MISMATCH` race failure in the same `-race` run (not investigated further — out of task-297's diff, and `t.Parallel`/shared-state ordering across the package's other tests is a plausible independent cause, but was not confirmed) |
| atlas-channel (`kafka/consumer/map`) | PASS | PASS | `go test -race ./kafka/consumer/map/...` — ok |
| atlas-channel (whole module) | PASS | — | `go build ./...` clean |

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE
- **Recommendation:** NEEDS_FIXES

## Action Items

1. Fix the data race in `handleStatusEventDamaged`
   (`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:263-315`):
   give each `routine.Go` closure its own local error variable instead of writing through the
   shared outer `err`, so the health-announce and damage-echo goroutines no longer race when
   both fire for the same `DAMAGED` event. Re-run
   `go test -race ./kafka/consumer/monster/...` to confirm the fix and that
   `TestHandleStatusEventDamaged_BossHpGauge` passes clean under `-race`.
2. Separately confirm whether `TestBridleFailReason/SPECIES_MISMATCH`'s race failure in the
   same run is real or an artifact of running many `-race` tests in one process; it sits
   outside task-297's diff and was not further diagnosed here.
3. Once (1) is fixed, dispatch `task-verifier` for the flagless `tools/verify.sh` run per the
   plan's Final Verification section (not yet run, and not run by this audit as the
   concurrent verify.sh process already in flight was not to be disturbed).
