# task-279 — Implementation Context

Companion to `plan.md`. Everything here was verified against the worktree at branch
`task-279-carnival-mob-skills`, head `0b0437f48`.

## Scope in one line

Eight already-classified mob skill ids (150–157, the Monster Carnival family) never reach the
existing monster temporary-stat machinery, and the already-declared `SEAL_SKILL` status never
gates anything. Nine small edits across two modules close both gaps. No new file except tests.

## Key files

| Path | Role | Touched by |
|---|---|---|
| `libs/atlas-constants/monster/skill.go` | `SkillTypeToStatusName` (`:71`), `IsAoeSkill` (`:99`), `skillNameMap` (`:145`), `SkillCategory` (`:214`, already returns `CARNIVAL_BUFF`) | Task 1 |
| `libs/atlas-constants/monster/temporary_stat.go` | All eight target `TemporaryStatType` values already exist; **read-only** | — |
| `libs/atlas-constants/monster/skill_test.go` | 30 lines today; the plan's table-driven style copies `TestReflectKindForSkill` (`:14`) | Task 1 |
| `services/atlas-monsters/.../monster/processor.go` | 2181 lines. Seal gate `:866`; `UseSkill` dispatch switch `:949`; `UseSkillGM` skill fetch `:1018` and dispatch switch `:1033`; `executeStatBuff` `:1174-1265` | Tasks 2, 5 |
| `services/atlas-monsters/.../monster/picker.go` | `pickerRelevantStatuses` `:69-76` (already lists `SEAL_SKILL`); `isPickerRelevantStatus`; seal gate `:124` | Task 4 |
| `services/atlas-monsters/.../monster/processor_test.go` | The setup patterns every new test copies | read-only |
| `services/atlas-monsters/docs/domain.md` | `UseSkill` step list, `UseSkillGM` sentence `:279`, Skill Picker items `:283`/`:286` | Task 6 |

Module roots (the `go build` / `go test` cwd): `libs/atlas-constants` and
`services/atlas-monsters/atlas.com/monsters`.

## Decisions carried from design.md

- **D1 — arm folding.** 150/151/152/153/156 join the *existing* `SkillTypeToStatusName` arms
  rather than getting their own, so "carnival PAD is not a distinct stat" (FR-1.2) is
  structurally true. 154/155/157 get new arms.
- **D3 — `IsAoeSkill` widening is inert.** No WZ entry for 150–157 declares `lt`/`rb`, and the
  function's only repo-wide caller (`processor.go:1253`) is the conjunction
  `IsAoeSkill(...) && sd.HasBoundingBox()`. It is forward-compat, and a knowing divergence from
  the Cosmic reference. The code comment at the site must say so and must not claim parity.
- **D6/D7 — one helper, both statuses.** `skillSuppressingStatus` returns the blocking status
  name (not a `bool`) so each caller logs which status blocked. `SEAL_SKILL` is checked first.
  Atlas's monster-side `SEAL` gate has no reference equivalent, but it is operator-reachable via
  `@mobclear`'s `validStatuses` and registered in the mob temporary-stat bitfield, so removing it
  would be an unrelated behavior change. Keep both.
- **D8 — `UseSkillGM` test seam.** `testMobSkillLookup` was honored only by `UseSkill`. Extending
  it to `UseSkillGM` is what makes the PRD criterion "`UseSkillGM` with a carnival skill id
  applies the same status effect" testable at all. The hook is `nil` in production.
- **Design §4 — the gate placement.** FR-6.3's stated rationale (catching a `SEAL_SKILL` that
  lands *during* the animation delay) is wrong: the gate at `:866` runs strictly before the delay
  begins. The requirement is implemented as specified; the in-flight case is an explicit non-goal,
  because `applyAnimationDelayedEffect` is shared by every category and a check there would newly
  let `SEAL` cancel in-flight debuffs, heals, and summons.

## Open item resolved at plan time

The PRD's final acceptance criterion targets `docs/research/missing-features/monsters-and-bosses.md`
§8. **That file does not exist in this repository** — `docs/research/missing-features/` contains
only `items-and-consumables.md`, and `git ls-files | grep monsters-and-bosses` returns nothing.
Asked the user; the decision was to **drop the criterion**. Documentation scope is limited to
`services/atlas-monsters/docs/domain.md` (Task 6). Do not create the research doc.

## Deliberate non-changes — do not "fix" these

- **Skill 145** (`SkillTypePhysicalMagicCounter`) is classified `REFLECT` but has no
  `SkillTypeToStatusName` arm, so it no-ops today. Pre-existing defect of the same shape as this
  task's, but with different semantics (the reference sets four stats plus a reflection value).
  Task 1 pins it returning `""` so the omission is visibly intentional.
- **Duplicate constant set.** `status.go:8` `StatusSeal = "SEAL"` and `temporary_stat.go:16`
  `TemporaryStatTypeSeal = "SEAL"` are two names for one wire string. FR-6.4 picks
  `TemporaryStatType*` because `pickerRelevantStatuses` already uses it. Do not consolidate.
- **`isBossAllowedStatus`** (`processor.go:1571-1585`) omits `ACCURACY`, `AVOIDABILITY`, and
  `SEAL_SKILL`, but is inert here — the boss filter runs only for `SourceTypePlayerSkill`, and
  every carnival effect is `SourceTypeMonsterSkill`.
- **`@mobclear`'s `validStatuses`** in atlas-messages does not list `SEAL_SKILL`, `ACCURACY`, or
  `AVOIDABILITY`, so a GM cannot clear those by name after `@mobstatus CARNIVAL_*`. Known
  GM-ergonomics gap; out of scope (design D9).
- **`applyAnimationDelayedEffect`**, `UseSkillGM`'s validation-free contract, and `UseBasicAttack`
  gain no suppression gate.

## Evidence that shapes the tests

- **No autonomous caster.** Exactly one mob in `Mob.wz` references skills 150–157: `9400593`
  (Hsalf, a level-130 boss), declaring **156 level 1**. That `MobSkill.img` entry has no `prop`,
  so `picker.go`'s `prop <= 0 → continue` never selects it. Task 4's
  `TestPicker_HsalfSkill156NoProp_ReturnsSentinel` pins this, so a future "default prop to 100"
  change cannot silently hand a boss a permanent +50 speed buff.
- **Reachable caster.** `@mobstatus CARNIVAL_PAD` (atlas-messages) → `SkillNameToId` →
  `USE_SKILL_FIELD` → `UseSkillGM` for every monster in the field. No atlas-messages change is
  needed for this to work once Task 1's `skillNameMap` entries land.
- **`-990` is real data.** Skill 155 **level 2** carries `x = -990` (a Lich debuff). `sd.X()` is
  `int32` and `executeStatBuff` passes it straight through. FR-4.3 is a prohibition on adding
  clamping, not a feature to build.
- **Durations are milliseconds.** `atlas-data`'s reader converts the WZ `time` seconds to ms
  (`services/atlas-data/.../mobskill/reader_test.go:16`), and `executeStatBuff` does
  `time.Duration(sd.Duration()) * time.Millisecond`. WZ `time=1200` → `1_200_000` ms → 20 min.

## Test-harness facts (saves the implementer a discovery pass)

- `TestMain` (`monster/registry_test.go:27`) installs a package-wide miniredis, initialises every
  registry, and calls `producertest.InstallNoop()`. Individual tests still swap `cooldownReg` /
  `attackCooldownReg` to a per-test miniredis; follow the existing pattern.
- Processors are built as `&ProcessorImpl{l:..., ctx:..., t:..., emit:..., inFieldFn:...}` struct
  literals, never via `NewProcessor`.
- `executeStatBuff`'s AoE loop reads `p.ByFieldProvider(...)`, which hits
  `GetMonsterRegistry().GetMonstersInMap` directly — **not** `inFieldFn` (that one returns
  *character* ids). Seed AoE targets with `r.CreateMonster(..., x, y, ...)` in the same field.
- `UseSkill` reads its animation delay via `information.NewProcessor(...).GetById(...)` directly,
  bypassing `testInformationLookup`. In tests that call fails, `animDelay` stays `0`, and the
  effect runs **synchronously** — assert immediately, no sleeps. This is pre-existing and the
  existing `UseSkill` tests already depend on it.
- `applyImmunityForTest` (`processor_test.go:952`) seeds an arbitrary status despite its name;
  reuse it rather than writing a second seeder.
- `logrus/hooks/test` is used elsewhere in the repo (`atlas-map-actions`, `atlas-reward-pools`)
  and needs no `go.mod` change in atlas-monsters — it ships in the logrus module.

## Task sizing

Seven tasks, each ≤ 4 files and confined to one module. No task was deliberately left large.
Tasks 1 → 2 → (3, 4) → 5 → 6 → 7 is the dependency order; Tasks 3 and 4 are independent of each
other. Task 3 is test-only (it pins behavior Task 2 unlocks) and Task 7 is the verification and
review gate.
