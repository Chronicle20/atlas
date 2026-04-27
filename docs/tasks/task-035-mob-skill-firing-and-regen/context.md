# Mob Skill Firing + HP/MP Recovery — Execution Context

Companion to `plan.md`. Quick-reference for executing agents.

## Inputs

- `prd.md` — full requirements (FR-1 … FR-6, acceptance §10).
- `design.md` — concrete shapes + decisions log D1…D7.

## Code areas touched

| Path | Role |
|---|---|
| `services/atlas-data/atlas.com/data/monster/reader.go` | WZ → entity parse; add `hpRecovery`, `mpRecovery`. |
| `services/atlas-data/atlas.com/data/monster/rest.go` | REST shape; add `hp_recovery`, `mp_recovery` JSON fields. |
| `services/atlas-data/atlas.com/data/monster/reader_test.go` | Fixture already contains `<int name="hpRecovery" value="10000"/>` and `<int name="mpRecovery" value="50000"/>`; add assertions. |
| `services/atlas-data/atlas.com/data/monster/rest_test.go` | Round-trip via `MarshalResponse` / `jsonapi.Unmarshal` then `reflect.DeepEqual`. |
| `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go` | Consumer-side REST shape; add matching fields + Extract. |
| `services/atlas-monsters/atlas.com/monsters/monster/information/model.go` | Add `hpRecovery`, `mpRecovery` fields + getters. |
| `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` | Builder setters used by tests. |
| `services/atlas-monsters/atlas.com/monsters/monster/model.go` | Add `lastDamageTakenMs int64` field + `LastDamageTakenMs()` getter. |
| `services/atlas-monsters/atlas.com/monsters/monster/builder.go` | Mirror new field in `Clone`, `ModelBuilder`, `SetLastDamageTakenMs`, `Build`. |
| `services/atlas-monsters/atlas.com/monsters/monster/registry.go` | `storedMonster.LastDamageTakenMs` (omitempty); `toStored`/`fromStored`; new `applyRecoveryScript`; new `ApplyRecovery` method; existing `applyDamageScript` Lua writes `mon.lastDamageTakenMs = nowMs`. |
| `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go` | Existing miniredis scaffold via `TestMain`. Add tests for damage-side `lastDamageTakenMs` write and `ApplyRecovery` matrix. |
| `services/atlas-monsters/atlas.com/monsters/monster/picker.go` | Add `propEligibleSeen` local; min-merge `nextRepick` with `nowMs + MonsterSkillPickerSweepInterval.Milliseconds()` when sentinel and prop-eligible. |
| `services/atlas-monsters/atlas.com/monsters/monster/picker_test.go` | Add prop-fail-reroll tests. |
| `services/atlas-monsters/atlas.com/monsters/monster/picker_task.go` | Sweep skip when `!m.ControllerHasAggro()`. |
| `services/atlas-monsters/atlas.com/monsters/monster/picker_task_test.go` | Add aggro-gate tests; assert sweep skips/repicks based on `controllerHasAggro`. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | (1) Spawn picker call gated on `m.ControllerHasAggro()`; (2) damage trigger guard loosened to `firstHitObserved || HpPercentage changed`; (3) `postExecute` re-fetches and gates on aggro. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` | Tests for damage-trigger first-hit-only firing path. |
| `services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go` (NEW) | New `MonsterRecoveryTask` mirroring `MonsterAggroDecayTask` / `MonsterSkillPickerSweepTask`. |
| `services/atlas-monsters/atlas.com/monsters/monster/recovery_task_test.go` (NEW) | Unit tests for the recovery task’s decision matrix (HP idle gate, MP unconditional, clamps, dead-mob skip, both-zero skip). |
| `services/atlas-monsters/atlas.com/monsters/monster/rest.go` | Add `ControllerHasAggro` and `NextEligibleRepickAtMs` to `RestModel` + populate in `Transform`. |
| `services/atlas-monsters/atlas.com/monsters/main.go` | Register `MonsterRecoveryTask` alongside existing tasks. |

## Decisions to honor (from design.md §3)

- **D1** `lastDamageTakenMs` is a direct field on `monster.Model`, persisted in `storedMonster` with `omitempty`. Write inside the existing `applyDamageScript` Lua (one extra assignment).
- **D2** HP regen emits via `damagedStatusEventProvider(updated, updated.UniqueId(), updated.UniqueId(), false, DamageSourceHeal, updated.DamageSummary())`. MP regen emits nothing. Recovery itself goes through a new `applyRecoveryScript` Lua, NOT through `UpdateMonster` (which is a non-CAS overwrite and would race).
- **D3** Picker prop-fail re-pick uses `min(nextRepick, nowMs + sweepIntervalMs)` whenever `propEligibleSeen` is set — strict superset of FR-4.4. Cooldown-derived `nextRepick` still wins when shorter.
- **D4** Recovery does not special-case `info.RemoveAfter > 0`.
- **D5** MP regen runs independently of SEAL.
- **D6** REST tags use snake_case, Go fields use PascalCase.
- **D7** Boss exclusion is data-driven (zero recovery values in WZ); no `info.Boss()` check in code.

## Key types & symbols

- `MonsterSkillPickerSweepInterval = 1500 * time.Millisecond` (already defined; reuse).
- `AggroIdleThresholdMs = int64(10_000)` in `aggro.go` (reuse for both decay and HP-regen idle gate).
- `MonsterRecoveryInterval = 10 * time.Second` (NEW const; in `recovery_task.go`).
- `nextSkillDecision` is a private struct — getter is `Model.NextSkillDecision()` returning struct with private `nextEligibleRepickAtMs` field; the picker reads it via `d.nextEligibleRepickAtMs`. The struct is local to the package, so REST `Transform` can read it directly.
- `DamageSourceHeal = "HEAL"` in `kafka.go:31`.
- `Model.ControllerHasAggro() bool` already exists (`model.go:121`).
- `Model.HpPercentage() uint32` already exists (`model.go:274`).
- `Model.Alive()` already exists (returns `m.Hp() > 0`).

## Testing scaffolding

- atlas-monsters tests use miniredis booted in `registry_test.go` `TestMain`. All registry-touching tests share that global `testMiniRedis`.
- `newTestTenant(t *testing.T)` is defined in `cooldown_test.go:24`.
- `testField()` is defined in `model_test.go:14`.
- `newPickerLogger()` defined in `picker_test.go:45`; reuse for any logrus-FieldLogger needs.
- `fakeRand`, `fakeCooldown`, `skillsOnly`, `mobSkillTable`, `mskill` defined in `picker_test.go:18..78`; reuse.
- atlas-data tests use `httptest` + `jsonapi.Unmarshal` for round-trip; pattern in `rest_test.go:38`.

## Build & test gates

- `cd services/atlas-data/atlas.com/data && go build ./... && go test ./...`
- `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...`
- `cd libs/atlas-packet && go build ./... && go test ./...` (sanity, no expected changes)
- `cd libs/atlas-constants && go build ./... && go test ./...` (sanity)

## Out-of-scope reminders (from PRD §2)

- No boss multi-phase scripts (deferred to spec-task-4).
- No AREA_POISON mist execution (deferred to spec-task-3).
- No player HP/MP regen.
- No new client packets.
- No new Kafka event topics or commands.
