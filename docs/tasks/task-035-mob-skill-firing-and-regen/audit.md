# Task 035 — Implementation Audit Notes

## Build & test gates (Task 15)

| Gate | Result |
|---|---|
| `services/atlas-data` build + test | ✅ PASS |
| `services/atlas-monsters` build + test | ✅ PASS |
| `libs/atlas-packet` build + test | ✅ PASS (sanity, no expected changes) |
| `libs/atlas-constants` build + test | ✅ PASS (sanity, no expected changes) |

## Manual verification

PRD §10.1 manual gameplay verification deferred to post-merge QA. The §10.2 automated coverage matrix is fully green via the unit tests added in Tasks 1–14.

## Commit chain

`95e837bea → 6fd160ed6 → 51737d361 → b4b84a296 → 21a39b47c → 58c850b4a → 7a8916bfd → 82986dc51 → e8ddbd27d → b1cd664ff → 026517d04 → 0acc39421 → 86a658ab5 → 05971614e`

(14 commits — one per implementation task; Task 15 is validation-only.)

---

# Plan Adherence Audit — task-035-mob-skill-firing-and-regen

**Plan Path:** `docs/tasks/task-035-mob-skill-firing-and-regen/plan.md`
**Audit Date:** 2026-04-27
**Branch:** `task-035-mob-skill-firing-and-regen`
**Base Branch:** `main`
**Auditor:** plan-adherence-reviewer (read-only)

## Executive Summary

All 15 plan tasks were faithfully implemented. The branch contains 15 commits (14 feature commits + 1 docs commit), one per implementation task plus the validation-only Task 15 captured as a docs commit. Every Functional Requirement in PRD §4 (FR-1.* through FR-6.*) maps to identifiable code in the diff, and every entry in the §10.2 acceptance matrix is covered by a concrete unit test. Build and test gates pass for all four required modules (`atlas-data`, `atlas-monsters`, `atlas-packet`, `atlas-constants`). Recommendation: READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | atlas-data parses `hpRecovery`/`mpRecovery` from WZ | DONE | `services/atlas-data/atlas.com/data/monster/rest.go:18-19`; reader at `reader.go:62-63`; test at `reader_test.go:1272-1277`. Commit `95e837bea`. |
| 2 | atlas-data REST round-trip test for recovery fields | DONE | `services/atlas-data/atlas.com/data/monster/rest_test.go:64-67`. Commit `6fd160ed6`. |
| 3 | `information.Model` recovery accessors | DONE | `information/model.go:16-17,90-95`; `information/rest.go:16-17,100-101`; builder setters at `information/builder.go:7-8,22-29,40-41`; test at `information/rest_test.go`. Commit `51737d361`. |
| 4 | `monster.Model.lastDamageTakenMs` field + builder | DONE | `monster/model.go:54,282-283`; builder at `monster/builder.go:36,62,114-118,210`; round-trip test at `monster/model_test.go:71-79`. Commit `b4b84a296`. |
| 5 | Persist `lastDamageTakenMs`; stamp inside `applyDamageScript` | DONE | `monster/registry.go:50` (`storedMonster`), `:149` (toStored), `:233` (fromStored), `:451` (Lua write); test at `registry_test.go:1247-1271`. Commit `21a39b47c`. |
| 6 | Picker `propEligibleSeen` + sweep min-merge | DONE | `monster/picker.go:128-131,201-211,216-224`; tests cover all four matrix variants in `picker_test.go`. Commit `58c850b4a`. |
| 7 | Sweep skips monsters without aggro | DONE | `monster/picker_task.go:70-72`; tests in `picker_task_test.go` (skip-when-false, repick-when-true, plus updated existing). Commit `7a8916bfd`. |
| 8 | Spawn picker call no-ops without aggro | DONE | `monster/processor.go:134-142`; `TestSpawnPickerGuardOnAggro` plus best-effort `TestCreate_DoesNotInvokeSpawnPickerWhenNoAggro` in `processor_test.go`. Commit `82986dc51`. |
| 9 | Post-`UseSkill` repick re-checks aggro | DONE | `monster/processor.go:570-585` (re-fetches monster, gates on `ControllerHasAggro()`); supporting tests in `processor_test.go`. Commit `e8ddbd27d`. |
| 10 | Damage trigger fires on first hit even when HP unchanged | DONE | `monster/processor.go:318-325` — guard now reads `firstHitObserved \|\| HP changed`; `TestDamageRepickGuard_FiresOnFirstHitMiss` table covers the matrix. Commit `b1cd664ff`. |
| 11 | `applyRecoveryScript` Lua + `ApplyRecovery` registry method | DONE | `monster/registry.go:503-545` (Lua), `:553-589` (Go method using `AggroIdleThresholdMs`); tests `TestApplyRecovery_AppliesMpUnconditionally`, `_ClampsAtMax`, `_HpGatedByIdleWindow`, `_SkipsDeadMob`. Commit `026517d04`. |
| 12 | `MonsterRecoveryTask` (new file) | DONE | `monster/recovery_task.go:1-127`; tests in `recovery_task_test.go` covering 4 scenarios. Run loop short-circuits dead/full and skips both-zero before applyFn. Commit `0acc39421`. |
| 13 | Register recovery task in `main.go` | DONE | `services/atlas-monsters/atlas.com/monsters/main.go:88` — `tasks.Register(l, tdm.Context())(monster.NewMonsterRecoveryTask(...))`. Commit `86a658ab5`. |
| 14 | Expose `controllerHasAggro` and `nextEligibleRepickAtMs` on monsters REST | DONE | `monster/rest.go:32-33,96-97`; tests `TestTransform_IncludesAggroAndRepickFields`, `TestTransform_OmitsZeroNextEligibleRepick`. Commit `05971614e`. |
| 15 | Final build, test, and acceptance check | DONE | All four build/test gates pass (see results table); §10.2 acceptance matrix items each have a concrete test in the diff; §10.1 manual gameplay deferred to post-merge QA per plan Step 5; docs commit `9c1b7eddd`. |

**Completion Rate:** 15/15 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Acceptance §10.2 Matrix Coverage

Every row in PRD §10.2 is backed by an automated test that lives in the diff:

| Acceptance row | Test |
|---|---|
| Picker prop-fail reschedules at sweep cadence | `TestPicker_AllPropFailReschedulesAtSweepCadence` (`picker_test.go`) |
| Sweep wins min over cooldown | `TestPicker_AllPropFailMergesWithLongCooldown` |
| Cooldown wins min over sweep | `TestPicker_AllPropFailLosesToShorterCooldown` |
| No prop-eligible → no sweep merge | `TestPicker_AllCooldownGated_NoPropEligible_NoSweepMerge` |
| Damage trigger fires on first-hit miss | `TestDamageRepickGuard_FiresOnFirstHitMiss` (table-driven) |
| Damage trigger does not fire on second-hit miss | same table |
| Sweep skips when aggro=false | `TestPickerSweep_SkipsWhenAggroFalse` |
| Sweep repicks when aggro=true | `TestPickerSweep_RepicksWhenAggroTrue` |
| Spawn picker no-ops without aggro | `TestSpawnPickerGuardOnAggro` (+ smoke `TestCreate_DoesNotInvokeSpawnPickerWhenNoAggro`) |
| Post-UseSkill repick no-ops on aggro decay | post-execute closure inspected at `processor.go:570-585` + `TestPostExecuteAggroGate_LogicTable` |
| Recovery applies MP unconditionally | `TestApplyRecovery_AppliesMpUnconditionally` |
| Recovery clamps at max | `TestApplyRecovery_ClampsAtMax` |
| Recovery HP idle-gated | `TestApplyRecovery_HpGatedByIdleWindow` |
| Recovery skips dead mobs | `TestApplyRecovery_SkipsDeadMob`, `TestRecoveryTask_SkipsDeadMob` |
| Recovery skips both-zero | `TestRecoveryTask_SkipsBothZero` |
| Recovery skips full HP+full MP | `TestRecoveryTask_SkipsFullHpAndFullMp` |
| Recovery applies & emits HP | `TestRecoveryTask_AppliesMpAndEmitsHp` |
| REST round-trips aggro/repick | `TestTransform_IncludesAggroAndRepickFields`, `TestTransform_OmitsZeroNextEligibleRepick` |
| atlas-data REST round-trips recovery | `rest_test.go` round-trip assertion (line 64) |

## Skipped / Deferred Tasks

None. Plan §10.1 manual gameplay verification is explicitly deferred to post-merge QA per plan Task 15 Step 5 — this is by design, not a gap.

The PRD's `Step 1` of Task 8 is documented as "best-effort" and the plan calls out the limitation explicitly (Create() depends on a real `atlas-data` HTTP call); Step 4's `TestSpawnPickerGuardOnAggro` is the primary regression guard and is present at `processor_test.go`. Code matches plan intent.

## Build & Test Results

| Module | Build | Tests | Notes |
|---|---|---|---|
| `services/atlas-data/atlas.com/data` | PASS | PASS | `monster` package + all sibling packages green. |
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS | `monster` package suite ~3.0s; `monster/information` and `kafka/consumer/monster` also green. |
| `libs/atlas-packet` | PASS | PASS | Sanity gate; no expected diff. |
| `libs/atlas-constants` | PASS | PASS | Sanity gate; no expected diff. |

## Plan-vs-Code Spot Checks

- **FR-2.1 spawn aggro guard** (`processor.go:138`): `if m.ControllerHasAggro() { ... RepickAndEmit(spawn) }` matches the plan body verbatim including the rationale comment.
- **FR-2.3 post-UseSkill aggro gate** (`processor.go:570-585`): re-fetches via `GetMonsterRegistry().GetMonster(p.t, uniqueId)`, returns Debug-log on either "gone" or "lost aggro" branch. Matches plan exactly.
- **FR-3.1 damage guard loosening** (`processor.go:321`): expression is `if !killed && (firstHitObserved || last.Monster.HpPercentage() != oldHpPercentage)` — exactly the plan's loosened guard.
- **FR-4.4 propEligibleSeen + min-merge** (`picker.go:128-131,201-224`): tracks `propEligibleSeen`, computes `sweepIntervalMs := MonsterSkillPickerSweepInterval.Milliseconds()`, only min-merges when `chosen.SkillId == 0 && propEligibleSeen`. Matches design D3.
- **FR-5.5 atomic CAS** (`registry.go:503-545`): Lua reads `mon.lastDamageTakenMs`, gates HP on `(nowMs - lastDamage) > idleThresholdMs`, clamps both stats, returns `{hpApplied, mpApplied, monster}` JSON envelope. The Go wrapper at `:553-589` decodes via `fromStored(env.Monster)` so the returned Model is canonical.
- **FR-5.6/5.7 emit only on hpApplied** (`recovery_task.go:113-124`): `applyFn` is invoked even if only MP recovers, but `emitFn` only fires when `hpApplied == true`. Matches the plan and PRD.
- **FR-5.8 lastDamageTakenMs persistence** (`registry.go:50,149,233,451`): direct field on `storedMonster`, written in Lua via `m.lastDamageTakenMs = nowMs` after the damage entries assignment. Matches design D1.
- **FR-1.4 omitempty** (`rest.go:33`): `NextEligibleRepickAtMs int64 \`json:"nextEligibleRepickAtMs,omitempty"\`` — correct tag.
- **Task 13 wiring** (`main.go:88`): `tasks.Register(l, tdm.Context())(monster.NewMonsterRecoveryTask(l, tdm.Context(), monster.MonsterRecoveryInterval))` — exact plan body.

## Minor Observations (non-blocking)

1. Commits `51737d361` (Task 3) and `58c850b4a` (Task 6) carry a `Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>` trailer. The user noted this was unintentional. Subjects and contents otherwise match the plan exactly. Not blocking; flagged only as observation.
2. Per-monster errors in `recovery_task.go:99-100,115-116,121-122` log at Debug. The plan (FR-6.3) explicitly authorizes this level. The reviewing user noted a Warn-vs-Debug judgment call from the backend reviewer — the plan is authoritative here, no action required.
3. The recovery task Run loop does not propagate the loop tenant into a context for `infoFn` directly; instead `NewMonsterRecoveryTask` constructs a `tenant.WithContext(tk.ctx, t)` inside its closure (`recovery_task.go:51-53`). This matches the plan and the `recoveryInfoFn` signature (`tenant.Model, uint32`).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required for plan adherence. (Optional cleanup: amend the two Sonnet 4.6 trailers if the user prefers to keep history clean, but the plan does not require it.)

---

# Backend Guidelines Audit — task-035-mob-skill-firing-and-regen

- **Date:** 2026-04-27
- **Auditor:** backend-guidelines-reviewer (adversarial)
- **Build:** PASS — `services/atlas-monsters` and `services/atlas-data` both compile clean and tests pass.
- **Tests:** atlas-monsters/monster (3.124s), atlas-monsters/monster/information (0.004s), atlas-monsters/kafka/consumer/monster (0.007s), atlas-data/monster (0.077s) — all green.
- **Overall:** PASS — every applicable check has file:line evidence proving compliance. Two non-blocking observations recorded.

## Scope

Three changed Go domain packages were audited:
1. `services/atlas-monsters/atlas.com/monsters/monster` — Redis-backed domain (no GORM).
2. `services/atlas-monsters/atlas.com/monsters/monster/information` — sub-domain (REST client side, no GORM).
3. `services/atlas-data/atlas.com/data/monster` — read-only WZ projector (no GORM).

`atlas-monsters` does NOT use GORM (state lives in Redis). DOM checks tied to GORM (entity.go ToEntity/Make, tenant callbacks, lazy provider queries, administrator.go) are N/A by design. The `Registry` (registry.go) plays the analogous role for write paths — Lua atomic CAS is the equivalent of administrator-level mutation. This was already the architecture before task-035.

## Build & Test Results

```
atlas-monsters: ok atlas-monsters/monster 3.124s
                ok atlas-monsters/monster/information 0.004s
                ok atlas-monsters/kafka/consumer/monster 0.007s
atlas-data:     ok atlas-data/monster 0.077s
```

## Domain Checklist Results

### atlas-monsters/monster (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go fluent setters | PASS | `monster/builder.go:41` (ModelBuilder), `:188` (Build); new `SetLastDamageTakenMs` at `:116-119` returns `*ModelBuilder` consistent with all other setters; `Clone()` at `:12-38` propagates `lastDamageTakenMs` at `:36` |
| DOM-02 | `ToEntity()` method | N/A | No GORM in atlas-monsters; `storedMonster` + `toStored()` at `monster/registry.go:99-151` is the analog and round-trips `lastDamageTakenMs` at `:149` |
| DOM-03 | `Make(Entity)` | N/A | See DOM-02; `fromStored()` at `monster/registry.go:153-235` round-trips |
| DOM-04 | Transform function | PASS | `monster/rest.go:61` (Transform), exposes new `ControllerHasAggro` and `NextEligibleRepickAtMs` at `:96-97` |
| DOM-05 | TransformSlice (or equivalent) | PASS | `monster/rest.go:62` uses `model.SliceMap(TransformDamageEntry)`; resource handler at `resource.go:34` uses `model.Map(Transform)` for the singular GET (only handler in this resource) |
| DOM-06 | Processor accepts FieldLogger | PASS | `monster/processor.go:60-66` (`ProcessorImpl.l logrus.FieldLogger`), `:69` (`NewProcessor(l logrus.FieldLogger, ctx context.Context)`) |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `monster/resource.go:28` (`NewProcessor(d.Logger(), d.Context())`); grep across all changed files shows zero `logrus.StandardLogger()` references |
| DOM-08 | POST/PATCH use RegisterInputHandler | N/A | Only handler in resource.go is GET (`monster/resource.go:21`); no POST/PATCH endpoints touched by task-035 |
| DOM-09 | Transform errors handled | PASS | `monster/resource.go:34-39` checks `err` from `model.Map(Transform)` and returns 500; no `_, _ :=` pattern |
| DOM-10 | Test DB has tenant callbacks | N/A | No GORM; tests use miniredis at `monster/registry_test.go:23-38`. Recovery and picker tasks use `tenant.WithContext(...)` for per-tenant scoping (`recovery_task.go:56,61`, `picker_task.go:40,44`) |
| DOM-11 | Providers use lazy evaluation | PASS | New code preserves the lazy pattern: `processor.ByIdProvider` returns `func() (Model, error)` at `monster/processor.go:86`; recovery task fetch closure at `recovery_task.go:55-58` defers info lookup |
| DOM-12 | No `os.Getenv()` in handlers/picker/task | PASS | grep across `resource.go`, `processor.go`, `picker.go`, `picker_task.go`, `recovery_task.go`: zero matches |
| DOM-13 | No cross-domain logic in handlers | PASS | `monster/resource.go:25-44` only calls `p.GetById` and `Transform` — no cross-domain calls |
| DOM-14 | Handlers don't call providers directly | PASS | `monster/resource.go:29` calls `p.GetById` (processor method), not `GetMonsterRegistry()` directly |
| DOM-15 | No direct entity creation in handlers | PASS | No GORM; resource.go does no Redis writes — all writes flow through `Registry.*` (analog to administrator), invoked only from processor |
| DOM-16 | administrator.go for writes | N/A | Redis-backed; `Registry` in `monster/registry.go:237-252` plays that role. New write-path additions (`ApplyRecovery` at `:553`, `lastDamageTakenMs` serialization at `:149`) live there, not in processor or handler |
| DOM-17 | Domain error → HTTP status mapping | PASS | `monster/resource.go:31` maps not-found to 404; `:37` maps Transform error to 500 |
| DOM-18 | JSON:API interface on REST models | PASS | `monster/rest.go:48-59` — GetID, SetID, GetName; no jsonapi struct tags |
| DOM-19 | Request models flat | N/A | No new request models added by task-035; only response model `RestModel` extended with `ControllerHasAggro` (`rest.go:32`) and `NextEligibleRepickAtMs` (`rest.go:33`), both flat fields |
| DOM-20 | Table-driven tests | PASS | `processor_test.go:462-494` (TestAttackerInField), `processor_test.go:727-749` (TestDamageRepickGuard_FiresOnFirstHitMiss) — explicit `tests := []struct{...}` + `t.Run`. Other tests use one-off cases by intent (per-trigger isolation), tolerated by guidelines |

### atlas-monsters/monster/information (sub-domain)

This package has model.go + rest.go + builder.go but no resource.go (REST client, not server).

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `information/builder.go:5` (ModelBuilder), `:33` (Build); new `SetHpRecovery`/`SetMpRecovery` at `:22,27` return `*ModelBuilder` consistent with the existing fluent style |
| DOM-04 | Extract (Transform analog) | PASS | `information/rest.go:82` — populates new `hpRecovery`/`mpRecovery` at `:100-101` |
| DOM-18 | JSON:API interface | PASS | `information/rest.go:69-80` (GetName/GetID/SetID) |
| DOM-19 | Flat request structure | N/A | No new request models |
| DOM-20 | Tests | PARTIAL | `information/rest_test.go:5-23` covers Extract for the new fields (single test case, not table-driven, but the surface is two integers — single case is adequate) |

### atlas-data/data/monster (sub-domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | Read function | PASS | `data/monster/reader.go:32` (Read); new fields wired at `:62-63` |
| DOM-18 | JSON:API interface | PASS | `data/monster/rest.go:45-60` |
| DOM-20 | Reader/REST tests | PASS | `data/monster/reader_test.go:32-33` exercises both fields via XML; `data/monster/rest_test.go:64-67` asserts round-trip |

## Sub-Domain Checklist Results

### MonsterRecoveryTask (new file: `monster/recovery_task.go`)

| Check | Status | Evidence |
|-------|--------|----------|
| Tenant scoping correct | PASS | `recovery_task.go:56,61` use `tenant.WithContext(tk.ctx, t)` for both info-lookup and emit, mirroring `picker_task.go:40,44`. `Run()` iterates `GetMonsterRegistry().GetMonsters()` (tenant-keyed map) and binds `tenant.Model t` per outer loop at `:82` |
| Per-tenant info cache correct | PASS | `recovery_task.go:80,84,95-104` — `infoCache` keyed by `uuid.UUID` (`tenant.Id()`); template-id cache only reused within the same tenant. Cross-tenant collision impossible by construction |
| No db/redis writes outside Registry | PASS | All state mutation goes through `tk.applyFn` → `(*Registry).ApplyRecovery` (`registry.go:553`) atomically via Lua |
| Atomic CAS pattern matches `applyDamageScript` | PASS | `applyRecoveryScript` (`registry.go:503-545`) follows the same shape as `applyDamageScript` (`:415-462`): `redis.call('GET')` → cjson.decode → mutate → `redis.call('SET', cjson.encode())` → return JSON envelope. Both rely on Lua-side atomicity (no Watch/MULTI loop). The dead-mob early-return at `:516-518` is a deliberate addition (forbidding healing the dead) consistent with `decayDamageEntriesScript`'s early checks |
| `lastDamageTakenMs` round-trip | PASS | `storedMonster.LastDamageTakenMs` uses `json:",omitempty"` (`registry.go:50`); legacy blobs default to 0 via Lua `mon.lastDamageTakenMs or 0` at `registry.go:524` |

### MonsterSkillPickerSweepTask (modified)

| Check | Status | Evidence |
|-------|--------|----------|
| Aggro gate added | PASS | `picker_task.go:70-72` skips monsters with `!m.ControllerHasAggro()` before the skill-cache lookup. Confirmed by `picker_task_test.go:104-132` (no-aggro sweep no-ops) and `:134-167` (with-aggro sweep does fire) |
| Tenant scoping unchanged | PASS | `picker_task.go:40,44` use `tenant.WithContext(tk.ctx, t)` |

### Picker (`picker.go`) — propEligibleSeen + sweep merge

| Check | Status | Evidence |
|-------|--------|----------|
| `propEligibleSeen` set only after every gate passes | PASS | `picker.go:204` is reached only after the cooldown / hp / mp / reflect-immunity / `prop > 0` gates. A `prop <= 0` skill doesn't taint the flag (continue at `:197`) |
| min-merge with cooldown-derived nextRepick correct | PASS | `picker.go:218-222` updates `nextRepick` only if the sweep candidate `< nextRepick` or `nextRepick == 0`. Verified by `picker_test.go:290-312` (sweep wins), `:314-342` (sweep beats long cooldown), `:344-368` (short cooldown beats sweep), `:370-390` (no prop-eligible → no sweep merge) |
| Sentinel zero-id check unchanged | PASS | `picker.go:218` guards on `chosen.SkillId == 0` |

### Processor — aggro gates (spawn, post-UseSkill, damage)

| Check | Status | Evidence |
|-------|--------|----------|
| Spawn picker gated by aggro | PASS | `processor.go:138-142` — calls `RepickAndEmit(..., RepickReasonSpawn)` only if `m.ControllerHasAggro()`. Test: `processor_test.go:601-637`, `:639-652` |
| Post-UseSkill closure re-fetches monster (no TOCTOU on stale closure) | PASS | `processor.go:570-585` — `postExecute` does `current, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)` and reads `current.ControllerHasAggro()` from the freshly-loaded monster, NOT from the captured `m`. The earlier alive-check in `applyAnimationDelayedEffect` also re-fetches at `:601-613` |
| Damage repick fires on first-hit miss | PASS | `processor.go:289,299-301,321-325` — `firstHitObserved` flag tracked across damage lines; guard `!killed && (firstHitObserved \|\| newPct != oldPct)` permits a 0-damage attack that flipped `controllerHasAggro` to still trigger a repick. Test: table-driven `TestDamageRepickGuard_FiresOnFirstHitMiss` at `processor_test.go:727-749` |
| `RepickAndEmit` exposed on Processor interface | PASS | `processor.go:51` (interface), `picker.go:234` (implementation) |

## Security / Multi-tenancy Review

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SEC-01 | Tenant header parsing not bypassed | PASS | All processor entry points obtain tenant via `tenant.MustFromContext(ctx)` at `processor.go:73`. The recovery task and sweep task receive tenant in the iterator from `Registry.GetMonsters()`, never from a request header |
| SEC-02 | No cross-tenant key collisions on new state | PASS | `monsterKey` (`registry.go:254`) and `mapIndexKey` (`:258`) include `t.Id().String()`. `lastDamageTakenMs` is stored inside the per-monster blob, so it inherits tenant scoping. Recovery task's per-tenant info cache (`recovery_task.go:80,84,95`) is keyed by `tenant.Id()` so two tenants spawning the same template id won't share entries |
| SEC-03 | No hardcoded secrets/keys | PASS | grep across modified files: zero matches for raw keys, passwords, JWT secrets. `EnvEventTopicMonsterStatus` constant resolves to env var name `EVENT_TOPIC_MONSTER_STATUS` (`kafka.go:12`), not a hardcoded topic |
| SEC-04 | Lua injection surface | PASS | `applyRecoveryScript` argv (`registry.go:557-562`) consists of integers serialized via `strconv.FormatUint`/`strconv.FormatInt`. No string interpolation into the Lua source. Same shape as the existing `applyDamageScript`, no regression |

## Specific Question Answers

1. **Immutable model + builder consistency** — PASS. `Model.lastDamageTakenMs` is private int64 (`model.go:54`) with public `LastDamageTakenMs()` getter (`:282`). `ModelBuilder.SetLastDamageTakenMs(v int64) *ModelBuilder` (`builder.go:116-119`) returns the receiver. `Clone()` propagates the field (`builder.go:36`). `Build()` includes it (`builder.go:210`). `information.Model.hpRecovery`/`mpRecovery` likewise private (`information/model.go:16-17`) with `HpRecovery()`/`MpRecovery()` getters (`:90-95`) and builder setters at `information/builder.go:22-29`. Consistent with project pattern.

2. **Lua atomic-CAS pattern matches `applyDamageScript`** — PASS. Same GET → cjson.decode → mutate → SET → cjson.encode envelope; same return-string-and-Go-decode round-trip; same tolerance for legacy/missing fields via `or 0`. See `registry.go:503-545` vs `:415-462`.

3. **`MonsterRecoveryTask` tenant scoping** — PASS. Uses `tenant.WithContext(tk.ctx, t)` for both `infoFn` and `emitFn` closures (`recovery_task.go:56,61`), mirroring the picker sweep. Per-tenant info cache (`recovery_task.go:80,84,95`) prevents cross-tenant template bleed.

4. **Aggro-gate TOCTOU** — PASS. The post-UseSkill closure re-fetches via `GetMonsterRegistry().GetMonster(p.t, uniqueId)` at `processor.go:573` and gates on the freshly loaded `current.ControllerHasAggro()` at `:578`. Spawn-side gate uses the just-created monster's flag (no time gap, freshly returned from `CreateMonster`). Sweep-side gate at `picker_task.go:70` reads from the snapshot returned by `GetMonsters()` — there is a tiny sweep-tick window but this is intrinsic to a periodic sweep, and the picker re-evaluation (`RepickAndEmit` → `GetMonster` at `picker.go:235`) provides a second-line defense. No tighter consistency model is achievable without holding a Redis lock across the whole sweep, which would defeat the cadence design.

5. **Logging level (Debug vs Warn) for the recovery task** — Non-blocking. The recovery task logs per-monster failures at `Debug` (`recovery_task.go:99,115,121`); the picker logs analogous failures at `Warn` (`processor.go:140,237,322` etc.). Confirmed intentional per the prompt and the plan.

## Summary

### Blocking (must fix)

- None. Every applicable check has file:line evidence proving compliance.

### Non-Blocking

- Recovery task logs per-monster failures at `Debug`; picker logs analogous failures at `Warn`. Asymmetry is intentional per plan and user guidance — surfaced for the record only.
- `picker.go:130` declares `var nextRepick int64` and uses `nextRepick == 0` as "unset" sentinel. Theoretically a skill whose computed expiry equals epoch ms 0 would be ambiguous, but `nowMs > 0` makes that unreachable in practice. Not a defect.
- `monster/rest.go:30-33` formatting diverges from the surrounding column alignment (extra spaces line up the new fields). Cosmetic; gofmt accepts it.

### Files relevant to this audit

- `services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go`
- `services/atlas-monsters/atlas.com/monsters/monster/registry.go`
- `services/atlas-monsters/atlas.com/monsters/monster/picker.go`
- `services/atlas-monsters/atlas.com/monsters/monster/picker_task.go`
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
- `services/atlas-monsters/atlas.com/monsters/monster/builder.go`
- `services/atlas-monsters/atlas.com/monsters/monster/model.go`
- `services/atlas-monsters/atlas.com/monsters/monster/rest.go`
- `services/atlas-monsters/atlas.com/monsters/monster/resource.go`
- `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go`
- `services/atlas-monsters/atlas.com/monsters/monster/information/model.go`
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go`
- `services/atlas-data/atlas.com/data/monster/reader.go`
- `services/atlas-data/atlas.com/data/monster/rest.go`
- `services/atlas-monsters/atlas.com/monsters/main.go` (task registration)
