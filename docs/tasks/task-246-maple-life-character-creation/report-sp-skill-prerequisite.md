# Report: bug-sp-skill-prerequisite

Task: task-246-maple-life-character-creation
Branch: task-246-maple-life-character-creation (committed directly, per ruling 2)

## Implemented

- `factory/maple_life.go`
  - Added `prerequisiteFor(spSkillId uint32) (uint32, bool)`, a switch keyed on
    `skill.WarriorImprovedMaxHpIncreaseId` -> `skill.WarriorImprovedHpRecoveryId`
    and `skill.MagicianImprovedMaxMpIncreaseId` -> `skill.MagicianImprovedMpRecoveryId`.
    Returns `false` for any other id (Bowman/etc.) -- nothing invented.
  - `toPreset`: when `e.SpSkillId != 0 && in.SP > 0`, builds a two-entry
    `skills` slice -- prerequisite at level 5 first (if `prerequisiteFor`
    reports one), then `e.SpSkillId` at `in.SP` -- and accumulates `spCost`
    (`in.SP`, +5 when a prerequisite was granted).
  - `remainingSP := spendSPPool(e.SP, spCost)` replaces the old
    `spendSPPool(e.SP, in.SP)`.
  - `spendSPPool`'s `nSP` parameter widened from `byte` to `int` (its only
    caller now passes `spCost`, which can be up to 15 -- fits either type, but
    `int` keeps the addition explicit and total per the brief's overflow
    note).
  - The `stats.Hp`/`stats.Mp` contribution block is untouched, as directed.
- `factory/processor.go` (`resolveMapleLifePreset`, the SP pool-sufficiency
  guard): computes `needed := int(in.SP)`, adds 5 when
  `prerequisiteFor(entry.SpSkillId)` reports one and `in.SP > 0`, and checks
  `needed > pool[0]` in place of the old `int(in.SP) > pool[0]`.
- `factory/maple_life_test.go`
  - Bumped the Warrior and Magician fixture SP pools in
    `mapleLifeTenantConfig` from `"9,...`"/`"7,...` to `"61,...` each, so the
    default request's `SP=5` plus the new +5 prerequisite cost fits (matching
    the bug report's own pool of 61) and so the new SP=4/SP=10 fixtures below
    have room.
  - `TestCreateMapleLife`'s `"sp above the pool"` case now needs a shrunk pool
    to still trip the pool guard (since the fixture pool grew to 61); replaced
    it with `"sp above the pool once the prerequisite's 5 is reserved"`,
    backed by a new `mapleLifeSmallPoolConfig()` helper (`SP: "8,0,...`" on
    the Warrior class, request `SP=5` needing 10). Added a sibling
    `"sp above the hard cap of 10"` case (`SP=11`) so the pre-existing
    `in.SP > 10` cap still has direct coverage.
  - `TestCreateMapleLifeSagaPayload` (Warrior, `SP=5`, base request): now
    asserts two `CreateSkill` steps -- `{1000000, 5}` then `{1000001, 5}` --
    and `payload.SP == "51,0,0,0,0,0,0,0,0,0"` (61 - 5 - 5).
  - `TestCreateMapleLifeSagaPayload_SPZero`: `payload.SP` expectation updated
    to `"61,0,0,0,0,0,0,0,0,0"` (fixture pool change only; still zero skills).
  - Added `TestCreateMapleLifeSagaPayload_WarriorPartialSP` (`SP=4`):
    asserts `[{1000000,5},{1000001,4}]` and `SP == "52,0,0,0,0,0,0,0,0,0"`
    (61 - 4 - 5), exactly the brief's worked example.
  - Added `TestCreateMapleLifeSagaPayload_MagicianFullSP` (`SP=10`): asserts
    `[{2000000,5},{2000001,10}]` and `SP == "46,0,0,0,0,0,0,0,0,0"`
    (61 - 10 - 5), exactly the brief's worked example.
  - `TestCreateMapleLifeSagaPayload_MagicianMP` and
    `TestCreateMapleLifeSagaPayload_NoSkillClass` (Bowman, `SpSkillId == 0`)
    needed no changes -- neither asserts the SP pool string, and the Bowman
    path is provably untouched (`prerequisiteFor` returns `false` for it).

## Tested

```
cd services/atlas-character-factory/atlas.com/character-factory
go build ./...
go test ./...
```

`go build ./...`: clean, no output.

`go test ./...`: all packages pass, output pristine aside from the module's
pre-existing kafka-writer startup log lines (unrelated to this change, same
as before).

`go test ./factory/... -run "MapleLife" -v`: all 20 subtests/tests under that
filter pass, including the two new saga-payload tests and the reshaped
`TestCreateMapleLife` table (guard-related cases renamed/added as above).

## Files changed

- `services/atlas-character-factory/atlas.com/character-factory/factory/maple_life.go`
- `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go`
- `services/atlas-character-factory/atlas.com/character-factory/factory/maple_life_test.go`

## Self-review

- Confirmed `spendSPPool` has exactly one caller (`toPreset`), so widening its
  signature was safe with no other call site to update.
- Confirmed `prerequisiteFor` is the single source of truth used by both
  `toPreset` (skill grant) and `resolveMapleLifePreset` (pool guard), so the
  two can never drift out of sync.
- Verified the Bowman path (`SpSkillId == 0`) is unaffected: `prerequisiteFor`
  is only reached inside the `e.SpSkillId != 0 && in.SP > 0` branch in
  `toPreset`, and the guard's `hasPrereq` check is `false` for id `0`.
- Verified `SP == 0` (opt-out) still grants nothing and spends nothing: the
  `in.SP > 0` guard on the skills block short-circuits before
  `prerequisiteFor` is even called.
- No seed data, template, packet, or Kafka-contract file touched, per the
  brief's "no ... change" list.
- Constants used are exactly the two identifiers named in the ruling:
  `skill.WarriorImprovedHpRecoveryId` / `skill.MagicianImprovedMpRecoveryId`,
  alongside the pre-existing `skill.WarriorImprovedMaxHpIncreaseId` /
  `skill.MagicianImprovedMaxMpIncreaseId` already imported in this file.

## Issues or concerns

None. The "Not yet answered" section in the bug file (whether the client's
own pre-submit SP preview subtracts the prerequisite's 5) is explicitly
flagged as non-blocking by the brief and left untouched.

## Resolution

Fixed as specified. Ready for review.
