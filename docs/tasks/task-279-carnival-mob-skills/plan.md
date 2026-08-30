# Monster Carnival Mob Skills (150–157) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make mob skill types 150–157 (the Monster Carnival family) reach the existing monster
temporary-stat machinery, and make the already-declared `SEAL_SKILL` status actually gate skill
use.

**Architecture:** Extend in place; add no new abstraction. Three switch/map extensions in
`libs/atlas-constants/monster/skill.go`, two dispatch-arm extensions plus one test-seam
extension in `services/atlas-monsters/.../monster/processor.go`, and one unexported helper in
`monster/picker.go` that both seal gates call. No new file except tests. No new constant, no new
interface, no new event type, no new packet, no schema change.

**Tech Stack:** Go 1.27, logrus, miniredis (test), `github.com/sirupsen/logrus/hooks/test`
(test), the repo's Builder pattern for test fixtures.

**Spec:** `docs/tasks/task-279-carnival-mob-skills/design.md` (PRD at `prd.md`)

## Global Constraints

- Module roots. `libs/atlas-constants` (module `github.com/Chronicle20/atlas/libs/atlas-constants`)
  and `services/atlas-monsters/atlas.com/monsters` (module `atlas-monsters`). All `go build` /
  `go test` commands run from one of those two directories.
- Import alias. Inside `atlas-monsters`, the constants package is imported as
  `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"`. Use that alias.
- No new constants. Every `TemporaryStatType` and `SkillType*` value this plan uses already
  exists in `libs/atlas-constants/monster/temporary_stat.go` and `skill.go`. Do not add any.
- Status-name symbols only. Never write the bare string literals `"SEAL"` or `"SEAL_SKILL"` in
  new production code; use `monster2.TemporaryStatTypeSeal` / `monster2.TemporaryStatTypeSealSkill`.
- Do not touch: `libs/atlas-packet`, `services/atlas-channel`, `services/atlas-data`,
  `services/atlas-buffs`, `services/atlas-messages`, seed data, skill 145's missing
  `SkillTypeToStatusName` arm, the `StatusSeal`/`TemporaryStatTypeSeal` duplication,
  `isBossAllowedStatus`, `applyAnimationDelayedEffect`.
- No test-only constructors. Use the existing Builders (`mobskill.NewBuilder()`,
  `information.NewBuilder()`, `monster.Clone(...)`) and the existing package-level hooks
  (`testMobSkillLookup`, `testInformationLookup`). Never add a `*_testhelpers.go`.
- WZ ground truth for 150–157 (design §1.4 / PRD §6). Ids 150,151,152,153,154,156 have 1 level;
  155 has 2; 157 has 1. `x` values: 150→40, 151→50, 152→50, 153→50, 154→50, 155 L1→30 /
  155 L2→**−990**, 156→50, 157→1. `time` (seconds in WZ, ingested by `atlas-data` as
  **milliseconds**): 1200 s for 150–154 L1, 155 L1, 156; 180 s for 155 L2 and 157. **No** entry
  declares `lt`, `rb`, `prop`, `mpCon`, `interval`, `hp`, or `count`.
- PRD acceptance criterion dropped by user decision: `docs/research/missing-features/monsters-and-bosses.md`
  does not exist in this repository and is **not** created by this plan. See `context.md`.

---

### Task 1: Constants — map, classify, and name the carnival family

**Files:**

- `libs/atlas-constants/monster/skill.go` — three edits: `SkillTypeToStatusName`,
  `IsAoeSkill`, `skillNameMap`
- `libs/atlas-constants/monster/skill_test.go` — new test cases appended
- `libs/atlas-constants/monster/temporary_stat.go` — read-only; the `TemporaryStatType`
  constants used by the mapping

Module root: `libs/atlas-constants`.

Patterns to copy: `libs/atlas-constants/monster/skill_test.go:14-29` (`TestReflectKindForSkill`
— the file's table-driven `cases := []struct{...}` shape).

**Interfaces:**

- Consumes: nothing from earlier tasks.
- Produces: `SkillTypeToStatusName(uint16) TemporaryStatType` now returns non-`""` for
  150–157; `IsAoeSkill(uint16) bool` now returns `true` for 150–157;
  `SkillNameToId(string) (uint16, bool)` now resolves the eight `CARNIVAL_*` names. Tasks 2–5
  depend on the first of these.

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-constants/monster/skill_test.go`. Four test functions, all table-driven in
the file's existing style (plain `for` over a `cases` slice, `t.Fatalf`/`t.Errorf` — the file
does not use `t.Run`).

`TestSkillTypeToStatusName_Carnival` — the FR-1.1 mapping plus the boundary ids:

| skillType | want |
|---|---|
| `SkillTypeCarnivalPAD` (150) | `TemporaryStatTypePowerUp` |
| `SkillTypeCarnivalMAD` (151) | `TemporaryStatTypeMagicUp` |
| `SkillTypeCarnivalPDR` (152) | `TemporaryStatTypePowerGuardUp` |
| `SkillTypeCarnivalMDR` (153) | `TemporaryStatTypeMagicGuardUp` |
| `SkillTypeCarnivalACC` (154) | `TemporaryStatTypeAccuracy` |
| `SkillTypeCarnivalEVA` (155) | `TemporaryStatTypeAvoidability` |
| `SkillTypeCarnivalSpeed` (156) | `TemporaryStatTypeSpeed` |
| `SkillTypeCarnivalSealSkill` (157) | `TemporaryStatTypeSealSkill` |
| `149` | `""` |
| `158` | `""` |
| `SkillTypePhysicalMagicCounter` (145) | `""` — design D1 deliberate non-change; do NOT add an arm for 145 |

`TestSkillTypeToStatusName_SharedStatArms` — pins design D1's arm folding (FR-1.2). Groups that
must all return the same value:

| ids | want |
|---|---|
| 100, 110, 150 | `TemporaryStatTypePowerUp` |
| 101, 111, 151 | `TemporaryStatTypeMagicUp` |
| 102, 112, 152 | `TemporaryStatTypePowerGuardUp` |
| 103, 113, 153 | `TemporaryStatTypeMagicGuardUp` |
| 115, 156 | `TemporaryStatTypeSpeed` |

`TestIsAoeSkill_CarnivalAndRegressions`:

| skillType | want |
|---|---|
| 150, 151, 152, 153, 154, 155, 156, 157 | `true` |
| 110, 111, 112, 113 (`*_Aoe`) | `true` |
| 114 (`SkillTypeHeal`) | `true` |
| 100, 101, 102, 103, 115, 120, 140, 143, 145, 200 | `false` |
| 149, 158 | `false` |

`TestSkillNameToId_Carnival` and `TestSkillTypeNames_IncludesCarnival`:

| name | want id | want ok |
|---|---|---|
| `"CARNIVAL_PAD"` | 150 | true |
| `"CARNIVAL_MAD"` | 151 | true |
| `"CARNIVAL_PDR"` | 152 | true |
| `"CARNIVAL_MDR"` | 153 | true |
| `"CARNIVAL_ACC"` | 154 | true |
| `"CARNIVAL_EVA"` | 155 | true |
| `"CARNIVAL_SPEED"` | 156 | true |
| `"CARNIVAL_SEAL_SKILL"` | 157 | true |
| `"CARNIVAL_NOPE"` | 0 | false |

`TestSkillTypeNames_IncludesCarnival` asserts `SkillTypeNames()` contains all eight names above
and that the returned slice is sorted ascending (`sort.StringsAreSorted(names)` is `true`).

`TestSkillCategory_Carnival` — regression pin on unchanged code: all eight ids return
`SkillCategoryCarnivalBuf`; 149 and 158 return `""`.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-constants && go test ./monster/... -run 'Carnival|SharedStatArms|AoeSkill' -v
```

Expected: FAIL. `SkillTypeToStatusName` returns `""` for 150–157, `IsAoeSkill` returns `false`,
`SkillNameToId` returns `(0, false)`.

- [ ] **Step 3: Extend `SkillTypeToStatusName`**

In `libs/atlas-constants/monster/skill.go:72-95`, fold four carnival ids into the existing arms
and add three new arms after `case SkillTypeMagicCounter:` to preserve ascending id order:

```go
func SkillTypeToStatusName(skillType uint16) TemporaryStatType {
	switch skillType {
	case SkillTypeWeaponAttackUp, SkillTypeWeaponAttackUpAoe, SkillTypeCarnivalPAD:
		return TemporaryStatTypePowerUp
	case SkillTypeMagicAttackUp, SkillTypeMagicAttackUpAoe, SkillTypeCarnivalMAD:
		return TemporaryStatTypeMagicUp
	case SkillTypeWeaponDefenseUp, SkillTypeWeaponDefenseUpAoe, SkillTypeCarnivalPDR:
		return TemporaryStatTypePowerGuardUp
	case SkillTypeMagicDefenseUp, SkillTypeMagicDefenseUpAoe, SkillTypeCarnivalMDR:
		return TemporaryStatTypeMagicGuardUp
	case SkillTypeSpeedUp, SkillTypeCarnivalSpeed:
		return TemporaryStatTypeSpeed
	case SkillTypePhysicalImmune:
		return TemporaryStatTypeWeaponAttackImmune
	case SkillTypeMagicImmune:
		return TemporaryStatTypeMagicAttackImmune
	case SkillTypeHardSkin:
		return TemporaryStatTypeHardSkin
	case SkillTypePhysicalCounter:
		return TemporaryStatTypeWeaponCounter
	case SkillTypeMagicCounter:
		return TemporaryStatTypeMagicCounter
	case SkillTypeCarnivalACC:
		return TemporaryStatTypeAccuracy
	case SkillTypeCarnivalEVA:
		return TemporaryStatTypeAvoidability
	case SkillTypeCarnivalSealSkill:
		return TemporaryStatTypeSealSkill
	default:
		return ""
	}
}
```

The carnival ids that share a stat with an existing skill go into the **existing** arm, not a
new one, so FR-1.2's "carnival PAD is not a distinct stat" is structurally true rather than a
coincidence of two arms returning the same constant. `default: return ""` is unchanged (FR-1.3).

- [ ] **Step 4: Extend `IsAoeSkill` with the FR-5.3 deviation comment**

Replace `libs/atlas-constants/monster/skill.go:99-108`:

```go
// IsAoeSkill returns true if the skill type is an AoE variant that affects nearby monsters.
func IsAoeSkill(skillType uint16) bool {
	switch skillType {
	case SkillTypeWeaponAttackUpAoe, SkillTypeMagicAttackUpAoe,
		SkillTypeWeaponDefenseUpAoe, SkillTypeMagicDefenseUpAoe,
		SkillTypeHeal:
		return true
	// Knowing divergence from the Cosmic reference (task-279 D3 / FR-5.3):
	// the reference consults lt/rb only for the *_M and HEAL_M arms, so it
	// does not treat the carnival family as AoE. Atlas does, for forward
	// compatibility with custom or later-version mob-skill data. This is
	// inert on all current WZ data — no MobSkill.img entry for 150-157
	// declares lt/rb, so the sole caller's
	// `IsAoeSkill(...) && sd.HasBoundingBox()` conjunction is always false.
	// Do not describe this as reference parity.
	case SkillTypeCarnivalPAD, SkillTypeCarnivalMAD,
		SkillTypeCarnivalPDR, SkillTypeCarnivalMDR,
		SkillTypeCarnivalACC, SkillTypeCarnivalEVA,
		SkillTypeCarnivalSpeed, SkillTypeCarnivalSealSkill:
		return true
	default:
		return false
	}
}
```

- [ ] **Step 5: Extend `skillNameMap`**

Add eight entries to `libs/atlas-constants/monster/skill.go:145-179`, placed contiguously after
`"PHYSICAL_MAGIC_COUNTER"` and before `"SUMMON"`. Run `gofmt` afterwards so the map's value
column realigns (the longest key becomes `"PHYSICAL_MAGIC_COUNTER"` still, so alignment should
not shift, but let `gofmt` decide).

```go
	"CARNIVAL_PAD":           SkillTypeCarnivalPAD,
	"CARNIVAL_MAD":           SkillTypeCarnivalMAD,
	"CARNIVAL_PDR":           SkillTypeCarnivalPDR,
	"CARNIVAL_MDR":           SkillTypeCarnivalMDR,
	"CARNIVAL_ACC":           SkillTypeCarnivalACC,
	"CARNIVAL_EVA":           SkillTypeCarnivalEVA,
	"CARNIVAL_SPEED":         SkillTypeCarnivalSpeed,
	"CARNIVAL_SEAL_SKILL":    SkillTypeCarnivalSealSkill,
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd libs/atlas-constants && gofmt -l . && go build ./... && go test ./...
```

Expected: `gofmt -l` prints nothing, build succeeds, all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-constants/monster/skill.go libs/atlas-constants/monster/skill_test.go
git commit -m "feat(atlas-constants): map, classify, and name carnival mob skills 150-157"
```

---

### Task 2: Dispatch — route `CARNIVAL_BUFF` through `executeStatBuff`

**Files:**

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — three edits at `:949`,
  `:1018`, `:1033`
- `services/atlas-monsters/atlas.com/monsters/monster/carnival_skill_test.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` — read-only; the
  setup patterns to copy

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy:

- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:2172-2238`
  (`TestUseSkill_...` — miniredis + `cooldownReg` swap, `testMobSkillLookup` swap, field/monster
  creation, `ProcessorImpl` struct literal).
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:880-910`
  (`TestExecuteStatBuff_ReflectStatus_...` — direct `executeStatBuff` call and
  `got.StatusEffects()` assertions).
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:952-970`
  (`applyImmunityForTest` — reuse it as-is to seed an arbitrary status; it is not
  immunity-specific despite the name).

**Interfaces:**

- Consumes: `monster2.SkillTypeToStatusName` from Task 1 (150→`POWER_UP`, 154→`ACCURACY`,
  157→`SEAL_SKILL`).
- Produces: `UseSkillGM` honors the package-level `testMobSkillLookup` hook exactly as
  `UseSkill` does. Tasks 3 and 5 rely on that.

**Environment notes the implementer must not rediscover:**

- `TestMain` (`monster/registry_test.go:27`) already installs a package-wide miniredis and
  initialises every registry, and `producertest.InstallNoop()` silences Kafka. Individual tests
  still swap `cooldownReg` to a per-test miniredis; follow that.
- `UseSkill` calls `information.NewProcessor(...).GetById(...)` **directly** (not through
  `testInformationLookup`) to read the animation delay. In tests that call fails, `animDelay`
  stays `0`, and `executeEffect`/`postExecute` run **synchronously** on the calling goroutine.
  That is what the existing `UseSkill` tests rely on; assert immediately after the call, no
  sleeps.
- `executeStatBuff` builds `SourceTypeMonsterSkill` effects, and `ApplyStatusEffect`'s elemental
  and boss-immunity checks run only for `SourceTypePlayerSkill`. So no information lookup is
  reached on the carnival path.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-monsters/atlas.com/monsters/monster/carnival_skill_test.go`.

`TestUseSkill_Carnival_AppliesMappedStatus` — table-driven with `t.Run`. Per case: swap
`cooldownReg` to a fresh miniredis, swap `testMobSkillLookup` to a `mobskill.NewBuilder()`
returning the case's `x`/`duration` for the requested id+level, create the monster with
`r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100, "", "")`, call
`p.UseSkill(m.UniqueId(), 1, skillId, skillLevel)`, then re-read via `r.GetMonster` and assert
exactly one status effect carrying the expected status name and value.

| subtest | skillId | level | builder `SetX` | builder `SetDuration` | expect status | expect value |
|---|---|---|---|---|---|---|
| `carnival_pad` | 150 | 1 | 40 | 1_200_000 | `POWER_UP` | 40 |
| `carnival_acc` | 154 | 1 | 50 | 1_200_000 | `ACCURACY` | 50 |
| `carnival_seal_skill` | 157 | 1 | 1 | 180_000 | `SEAL_SKILL` | 1 |

Assertions per case: `len(got.StatusEffects()) == 1`; `se.HasStatus(string(want))` is true;
`se.Statuses()[string(want)] == wantValue`; `se.SourceSkillId() == uint32(skillId)`;
`se.IsReflect()` is false.

`TestUseSkillGM_Carnival_AppliesMappedStatus` — identical table and identical assertions, but
calls `p.UseSkillGM(m.UniqueId(), skillId, skillLevel)`. This test cannot pass until Step 4
lands the `testMobSkillLookup` seam in `UseSkillGM`.

`TestUseSkill_Carnival_NoUnknownCategoryWarning` — for both entry points and all eight ids
(150–157), build the processor with a captured logger:

```go
l, hook := test.NewNullLogger()
l.SetLevel(logrus.DebugLevel)
```

(import `"github.com/sirupsen/logrus/hooks/test"`), run the cast, then assert no entry in
`hook.AllEntries()` has a `Message` containing `"unknown skill category"` and none has a
`Message` containing `"No status mapping for skill type"`. Clear the hook between ids with
`hook.Reset()`.

`TestExecuteStatBuff_Carnival_NoOppositeImmunityPrecancel_NotReflect` (FR-3.3): seed
`MAGIC_ATTACK_IMMUNE` with `applyImmunityForTest(t, p, m.UniqueId(), string(monster2.TemporaryStatTypeMagicAttackImmune), 1)`,
re-read the monster, then call `p.executeStatBuff(m, sd, byte(monster2.SkillTypeCarnivalPAD), 1)`
with `sd = mobskill.NewBuilder().SetSkillId(150).SetLevel(1).SetX(40).SetDuration(1_200_000).Build()`.
Assert:

- `got.HasStatusEffect(string(monster2.TemporaryStatTypeMagicAttackImmune))` is still `true`
  (the immunity was NOT pre-cancelled — `category != SkillCategoryImmunity`)
- `got.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp))` is `true`
- `len(got.StatusEffects()) == 2`
- the `POWER_UP` effect has `IsReflect() == false`, `ReflectKind() == ""`,
  `ReflectPercent() == 0`, `ReflectMaxDamage() == 0`

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run 'Carnival' -v
```

Expected: FAIL. The `UseSkill`/`UseSkillGM` cases apply no status because
`SkillCategoryCarnivalBuf` falls through to `default:`; the warning test sees "unknown skill
category"; `TestUseSkillGM_Carnival_...` additionally fails on a live HTTP attempt to atlas-data.

- [ ] **Step 3: Add the dispatch arm to both switches**

`services/atlas-monsters/atlas.com/monsters/monster/processor.go:949` (inside `UseSkill`'s
`executeEffect` closure):

```go
		case monster2.SkillCategoryStatBuff, monster2.SkillCategoryImmunity,
			monster2.SkillCategoryReflect, monster2.SkillCategoryCarnivalBuf:
			p.executeStatBuff(m, sd, skillId, skillLevel)
```

`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1033` (inside `UseSkillGM`):

```go
	case monster2.SkillCategoryStatBuff, monster2.SkillCategoryImmunity,
		monster2.SkillCategoryReflect, monster2.SkillCategoryCarnivalBuf:
		p.executeStatBuff(m, sd, skillId, skillLevel)
```

`default:` in both switches is untouched (FR-3.4). Add no new executor function (FR-3.2).

- [ ] **Step 4: Honor `testMobSkillLookup` in `UseSkillGM`**

Replace `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1018` so `UseSkillGM`
resolves skill data through the same seam `UseSkill` already uses (design D8). The hook is `nil`
in production, so the added branch is unreachable outside tests.

```go
	var sd mobskill.Model
	if testMobSkillLookup != nil {
		sd, err = testMobSkillLookup(uint16(skillId), uint16(skillLevel))
	} else {
		sd, err = mobskill.NewProcessor(p.l, p.ctx).GetByIdAndLevel(uint16(skillId), uint16(skillLevel))
	}
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve mob skill [%d] level [%d] for GM command.", skillId, skillLevel)
		return
	}
```

Also update the doc comment at `processor.go:89-90` — it currently says "When nil (production),
UseSkill calls mobskill.GetByIdAndLevel normally." Change `UseSkill` to `UseSkill and
UseSkillGM`.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && gofmt -l . && go build ./... && go test ./monster/... -run 'Carnival' -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/carnival_skill_test.go
git commit -m "feat(atlas-monsters): dispatch CARNIVAL_BUFF skills through executeStatBuff"
```

---

### Task 3: Pin values, duration, refresh-on-recast, and AoE application

No production change. Task 2 made these behaviors reachable; this task pins them so a later
refactor cannot break them silently (design D4, D5, and risks §9).

**Files:**

- `services/atlas-monsters/atlas.com/monsters/monster/carnival_value_test.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — read-only;
  `executeStatBuff` at `:1174-1265` is the code under test
- `services/atlas-monsters/atlas.com/monsters/monster/status.go` — read-only; `StatusEffect`
  accessors (`Statuses()`, `Duration()`, `ExpiresAt()`, `IsReflect()`)

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:880-910`
(direct `executeStatBuff` call with a `ProcessorImpl` struct literal).

**Interfaces:**

- Consumes: Task 2's dispatch arm; Task 1's mapping.
- Produces: nothing.

**Facts the implementer must not rediscover:**

- `executeStatBuff`'s AoE loop reads monsters from `p.ByFieldProvider(m.Field())`, which calls
  `GetMonsterRegistry().GetMonstersInMap(p.t, f)` directly — **not** `inFieldFn` (which returns
  *character* ids and is unused here). So seeding the AoE targets means calling
  `r.CreateMonster` in the same field with the desired x/y.
- The AoE containment test is `dx := other.X() - m.X(); dy := other.Y() - m.Y()` and requires
  `dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY()` (inclusive both ends).
- `sd.HasBoundingBox()` is `false` unless `mobskill.NewBuilder().SetBoundingBox(...)` was called.
- `Builder.AddStatusEffect` removes any existing effect of the same status type before appending
  (for every status except `VENOM`), which is what makes recast a refresh rather than a
  duplicate.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-monsters/atlas.com/monsters/monster/carnival_value_test.go`.

`TestExecuteStatBuff_Carnival_NegativeXSurvives` (FR-4.3) — skill 155 level 2, the Lich debuff:

| input | value |
|---|---|
| `sd` | `mobskill.NewBuilder().SetSkillId(155).SetLevel(2).SetX(-990).SetDuration(180_000).Build()` |
| call | `p.executeStatBuff(m, sd, byte(monster2.SkillTypeCarnivalEVA), 2)` |
| expect | exactly 1 effect; `se.Statuses()["AVOIDABILITY"] == -990` (not `0`, not `990`) |

`TestExecuteStatBuff_Carnival_DurationIsMilliseconds` (FR-4.4) — `atlas-data` ingests the WZ
`time=1200` seconds as `1_200_000` ms:

| input | value |
|---|---|
| `sd` | `...SetSkillId(150).SetLevel(1).SetX(40).SetDuration(1_200_000).Build()` |
| call | `p.executeStatBuff(m, sd, byte(monster2.SkillTypeCarnivalPAD), 1)` |
| expect | `se.Duration() == 20 * time.Minute` |

`TestExecuteStatBuff_Carnival_RecastRefreshesValueAndExpiry` (FR-4.1, design D4) — two casts of
skill 150 on the same monster, re-reading the model from the registry between them:

| step | `sd` | expect after |
|---|---|---|
| first cast | `SetX(40).SetDuration(60_000)` | 1 effect, `POWER_UP` = 40 |
| second cast | `SetX(99).SetDuration(120_000)` | still exactly 1 effect; `POWER_UP` == 99; `Duration() == 2 * time.Minute`; `ExpiresAt()` strictly after the first effect's `ExpiresAt()` |

Capture the first effect's `ExpiresAt()` into a local before the second cast so the comparison is
against a real recorded value.

`TestExecuteStatBuff_Carnival_NoBoundingBox_CasterOnly` (FR-5.2) — two monsters in the same
field, caster at `(0, 0)`, other at `(30, 10)`, `sd` built **without** `SetBoundingBox`:

| monster | position | expect `POWER_UP` |
|---|---|---|
| caster | (0, 0) | present |
| other | (30, 10) | absent |

`TestExecuteStatBuff_Carnival_WithBoundingBox_InBoxOnly` — three monsters in the same field,
`sd` built with `.SetBoundingBox(-50, -30, 50, 30)`:

| monster | position | dx, dy | expect `POWER_UP` |
|---|---|---|---|
| caster | (0, 0) | — | present |
| in-box | (30, 10) | 30, 10 | present |
| out-of-box | (200, 0) | 200, 0 | absent |

`r.CreateMonster`'s signature is
`CreateMonster(ctx, t, f, monsterId, x, y, fh, stance, team, hp, mp, spawnSourceType, spawnSourceId)`,
so pass x/y as the 5th and 6th arguments, e.g.
`r.CreateMonster(ctx, ten, f, 5100004, 30, 10, 0, 5, 0, 3000, 100, "", "")`.

- [ ] **Step 2: Run the tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run 'Carnival_(NegativeX|Duration|Recast|NoBoundingBox|WithBoundingBox)' -v
```

Expected: PASS on the first run — these pin behavior Task 2 already unlocked, and the design
verified each mechanism exists. If any FAILS, that is a real defect in the assumed behavior:
stop, diagnose against `executeStatBuff` and `monster/builder.go`'s `AddStatusEffect`, and report
before writing any production change. Do **not** add clamping, normalization, or duration
rescaling to make a test pass — FR-4.3 and FR-4.4 are prohibitions on exactly that.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/carnival_value_test.go
git commit -m "test(atlas-monsters): pin carnival buff value, duration, recast, and AoE behavior"
```

---

### Task 4: `skillSuppressingStatus` helper and the picker gate

**Files:**

- `services/atlas-monsters/atlas.com/monsters/monster/picker.go` — add the helper beside
  `isPickerRelevantStatus`; replace the bare `"SEAL"` check at `:124`
- `services/atlas-monsters/atlas.com/monsters/monster/picker_test.go` — new test cases appended
- `libs/atlas-constants/monster/temporary_stat.go` — read-only; `TemporaryStatTypeSeal` (`:16`)
  and `TemporaryStatTypeSealSkill` (`:32`)

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `services/atlas-monsters/atlas.com/monsters/monster/picker_test.go:94-108`
(`TestPicker_SealedMonster_ReturnsSentinel` — `Clone(m).AddStatusEffect(...).Build()` seeding
plus the `pickNextSkill(...)` call shape with `skillsOnly` / `mobSkillTable` / `fakeCooldown` /
`fakeRand`).

**Interfaces:**

- Consumes: `monster2.TemporaryStatTypeSeal`, `monster2.TemporaryStatTypeSealSkill`.
- Produces: `func skillSuppressingStatus(m Model) monster2.TemporaryStatType` — returns the
  blocking status name, or `""` when the monster is not suppressed. Task 5 calls it from
  `UseSkill`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-monsters/atlas.com/monsters/monster/picker_test.go`. Seed statuses the
same way `TestPicker_SealedMonster_ReturnsSentinel` does:

```go
m = Clone(m).AddStatusEffect(NewStatusEffect("MONSTER_SKILL", 0, 100, 1,
	map[string]int32{string(monster2.TemporaryStatTypeSealSkill): 1}, time.Minute, 0)).Build()
```

`TestPicker_SealSkillMonster_ReturnsSentinel` — monster carries `SEAL_SKILL`, skill list
`[]information.Skill{{Id: 100, Level: 1}}`, skill table
`{100*1000 + 1: mskill(t, 100, 1, 100, 0, 0, 0)}` (prop 100 — guaranteed to be picked but for
the gate), `&fakeRand{values: []int{0}}`. Expect `d.IsSentinel()` is `true`.

`TestPicker_SealMonster_StillReturnsSentinel` — the same setup but seeding
`monster2.TemporaryStatTypeSeal`. This is the design-D7 regression pin: broadening the gate must
not stop `SEAL` from blocking. Expect sentinel. (The existing
`TestPicker_SealedMonster_ReturnsSentinel` uses the bare `"SEAL"` literal; leave it as-is and add
this symbol-based one alongside.)

`TestPicker_SealSkillAndSeal_ReturnsSentinel` — both statuses seeded. Expect sentinel.

`TestPicker_HsalfSkill156NoProp_ReturnsSentinel` — the design §1.1 regression. Mob `9400593`
(Hsalf) is the only mob in `Mob.wz` referencing this family, and it declares skill 156 level 1,
whose `MobSkill.img` entry has **no** `prop`. Skill list `[]information.Skill{{Id: 156, Level: 1}}`,
skill table `{156*1000 + 1: mskill(t, 156, 1, 0, 0, 0, 0)}` — note `prop = 0`, matching the real
WZ data. `&fakeRand{values: []int{0}}` (a roll that would succeed if reached). No status effects
seeded. Expect `d.IsSentinel()` is `true`, pinning that `picker.go`'s `prop <= 0 → continue` — and
not luck — is what keeps a level-130 boss from acquiring a permanent +50 speed buff.

`TestSkillSuppressingStatus` — direct unit test of the helper:

| monster statuses seeded | want |
|---|---|
| none | `""` |
| `SEAL_SKILL` only | `monster2.TemporaryStatTypeSealSkill` |
| `SEAL` only | `monster2.TemporaryStatTypeSeal` |
| both | `monster2.TemporaryStatTypeSealSkill` (more specific wins) |
| `POWER_UP` only | `""` |

Build the monster with `newPickerTestMonster(t, 100, 50)` and seed with
`Clone(m).AddStatusEffect(...).Build()`.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run 'SealSkill|SkillSuppressingStatus|Hsalf' -v
```

Expected: FAIL to compile — `undefined: skillSuppressingStatus`.

- [ ] **Step 3: Add the helper to `picker.go`**

Place it in `services/atlas-monsters/atlas.com/monsters/monster/picker.go` immediately after
`isPickerRelevantStatus`, which is where the package already keeps its status-classification
predicates.

```go
// skillSuppressingStatus returns the name of the status blocking the monster
// from using skills, or "" if none. SEAL_SKILL is the reference gate
// (Cosmic Monster.java:1457); SEAL has no monster-side equivalent in the
// reference and is retained as a pre-existing, operator-reachable Atlas
// behavior (task-279 design D7). SEAL_SKILL is checked first so that when
// both are present the more specific status is the one reported.
func skillSuppressingStatus(m Model) monster2.TemporaryStatType {
	if m.HasStatusEffect(string(monster2.TemporaryStatTypeSealSkill)) {
		return monster2.TemporaryStatTypeSealSkill
	}
	if m.HasStatusEffect(string(monster2.TemporaryStatTypeSeal)) {
		return monster2.TemporaryStatTypeSeal
	}
	return ""
}
```

It returns the status rather than a `bool` so each caller can name `SEAL_SKILL` distinctly from
`SEAL` in its own log line (FR-7.2) without the helper knowing anything about logging.

- [ ] **Step 4: Replace the picker's bare `"SEAL"` gate**

`services/atlas-monsters/atlas.com/monsters/monster/picker.go:123-127` currently reads:

```go
	// Sealed monsters cannot fire any skill; emit sentinel.
	if m.HasStatusEffect("SEAL") {
		l.Debugf("Picker: monster [%d] is SEALed; no candidates.", m.UniqueId())
		return Decision{}
	}
```

Replace with:

```go
	// Skill-suppressed monsters cannot fire any skill; emit sentinel.
	if st := skillSuppressingStatus(m); st != "" {
		l.Debugf("Picker: monster [%d] has [%s]; no candidates.", m.UniqueId(), st)
		return Decision{}
	}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && gofmt -l . && go build ./... && go test ./monster/... -v
```

Expected: all PASS, including the pre-existing `TestPicker_SealedMonster_ReturnsSentinel`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/picker.go services/atlas-monsters/atlas.com/monsters/monster/picker_test.go
git commit -m "feat(atlas-monsters): gate the skill picker on SEAL_SKILL as well as SEAL"
```

---

### Task 5: The `UseSkill` executor gate

**Files:**

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — replace the bare `"SEAL"`
  check at `:865-869`
- `services/atlas-monsters/atlas.com/monsters/monster/seal_skill_test.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/picker.go` — read-only;
  `skillSuppressingStatus` from Task 4

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy:

- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:2172-2238` (`UseSkill`
  test setup).
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:1577-1624`
  (`TestUseBasicAttack_HappyPath_DeductsMpAndRegistersCooldown` — `attackCooldownReg` swap and
  `testInformationLookup` returning `information.NewBuilder().SetAttacks(...)`).
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:952-970`
  (`applyImmunityForTest`, reused to seed `SEAL_SKILL`).

**Interfaces:**

- Consumes: `skillSuppressingStatus` (Task 4); Task 2's dispatch arm for the end-to-end case.
- Produces: nothing.

**Scope boundary (design §4).** Implement the gate at `processor.go:866` only. Do **not** add a
suppression re-check inside `applyAnimationDelayedEffect`: that function is shared by every skill
category, so a check there would newly cause `SEAL` to cancel in-flight debuffs, heals, and
summons for skills 120–136 and 200 — a live behavior change no requirement asks for. The
reference consults `canUseSkill` at selection time, not after the animation. Likewise, add **no**
gate to `UseBasicAttack` (FR-6.6) and none to `UseSkillGM`, whose documented contract is to skip
validation.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-monsters/atlas.com/monsters/monster/seal_skill_test.go`.

`TestUseSkill_SealSkill_RejectsAndLogsDistinctly` (FR-6.3, FR-7.2) — build the processor with
`l, hook := test.NewNullLogger()` and `l.SetLevel(logrus.DebugLevel)`. Seed `SEAL_SKILL` via
`applyImmunityForTest(t, p, m.UniqueId(), string(monster2.TemporaryStatTypeSealSkill), 1)`, swap
`testMobSkillLookup` to return a skill-150 model with `SetX(40).SetDuration(1_200_000)`, then call
`p.UseSkill(m.UniqueId(), 1, byte(monster2.SkillTypeCarnivalPAD), 1)`.

| assertion | expected |
|---|---|
| `got.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp))` | `false` — the skill did not execute |
| `len(got.StatusEffects())` | `1` — only the seeded `SEAL_SKILL` |
| some `hook.AllEntries()` message | contains `"SEAL_SKILL"` |
| that same message | does **not** equal the `SEAL` wording — assert the rejection line names `SEAL_SKILL`, satisfying "distinct from the SEAL rejection" |

`TestUseSkill_Seal_StillRejects` — same setup but seeding
`string(monster2.TemporaryStatTypeSeal)`. Assert no `POWER_UP` applied and a logged message
containing `"SEAL"`. This is the D7 regression: the gate is strictly additive.

`TestUseBasicAttack_SealSkill_StillSucceeds` (FR-6.6) — copy the happy-path basic-attack setup:
swap `attackCooldownReg` to a fresh miniredis; swap `testInformationLookup` to return
`information.NewBuilder().SetAttacks([]information.AttackInfo{{Pos: 2, ConMP: 5, AttackAfter: 1500}}).Build()`;
create the monster with `mp = 100`; seed `SEAL_SKILL`; then call `p.UseBasicAttack(uniqueId, uint8(1))`
(0-indexed `attackPos` 1 maps to `AttackInfo.Pos` 2).

| assertion | expected |
|---|---|
| `got.Mp()` | `95` — MP was deducted, so the attack ran |
| `attackCooldownReg.IsOnCooldown(ctx, ten, uniqueId, uint8(1))` | `true` |

`TestUseSkill_Skill157ThenAnySkill_RejectedEndToEnd` — the end-to-end case. Swap
`testMobSkillLookup` to a table-driven stub keyed on the requested id:

| requested id | returned model |
|---|---|
| 157 | `SetSkillId(157).SetLevel(1).SetX(1).SetDuration(180_000)` |
| 150 | `SetSkillId(150).SetLevel(1).SetX(40).SetDuration(1_200_000)` |

Then:

1. `p.UseSkill(m.UniqueId(), 1, byte(monster2.SkillTypeCarnivalSealSkill), 1)` → re-read; assert
   `got.HasStatusEffect(string(monster2.TemporaryStatTypeSealSkill))` is `true`.
2. `p.UseSkill(m.UniqueId(), 1, byte(monster2.SkillTypeCarnivalPAD), 1)` → re-read; assert
   `got.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp))` is `false` and
   `len(got.StatusEffects()) == 1`.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run 'SealSkill|Seal_Still|Skill157' -v
```

Expected: FAIL. `UseSkill` still gates on the bare `"SEAL"` literal, so the `SEAL_SKILL` cases
execute the skill and apply `POWER_UP`. `TestUseSkill_Seal_StillRejects` and
`TestUseBasicAttack_SealSkill_StillSucceeds` should already PASS — that is expected; they are
regression pins, not drivers.

- [ ] **Step 3: Replace the executor's bare `"SEAL"` gate**

`services/atlas-monsters/atlas.com/monsters/monster/processor.go:865-869` currently reads:

```go
	// Check seal status - sealed monsters cannot use skills
	if m.HasStatusEffect("SEAL") {
		p.l.Debugf("Monster [%d] is sealed and cannot use skill [%d].", uniqueId, skillId)
		return
	}
```

Replace with:

```go
	// Skill-suppression gate. Mirrors the picker's gate so a decision that
	// went stale between pick and cast is rejected at cast time. This runs
	// before the animation delay, so it deliberately does not observe a
	// status that lands mid-flight (task-279 design §4).
	if st := skillSuppressingStatus(m); st != "" {
		p.l.Debugf("Monster [%d] has [%s] and cannot use skill [%d].", uniqueId, st, skillId)
		return
	}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && gofmt -l . && go build ./... && go test ./... 
```

Expected: `gofmt -l` prints nothing; all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/seal_skill_test.go
git commit -m "feat(atlas-monsters): gate UseSkill on SEAL_SKILL and name the blocking status"
```

---

### Task 6: Update the atlas-monsters domain doc

**Files:**

- `services/atlas-monsters/docs/domain.md` — update the `UseSkill` step list and the Skill Picker
  section

Module root: n/a (documentation only).

**Interfaces:**

- Consumes: the behavior landed by Tasks 1–5.
- Produces: nothing.

**Scope note.** The PRD's final acceptance criterion also named
`docs/research/missing-features/monsters-and-bosses.md` §8. That file does not exist in this
repository (`git ls-files | grep monsters-and-bosses` is empty; `docs/research/missing-features/`
contains only `items-and-consumables.md`). By explicit user decision at plan time, that criterion
is **dropped**. Do not create the file and do not invent a substitute path.

- [ ] **Step 1: Update the UseSkill flow description**

`services/atlas-monsters/docs/domain.md`, the numbered `UseSkill` list ending at `:277`. The
gate is described in the numbered steps preceding line 270; locate the item that mentions the
seal check and change it to say the monster is rejected if it carries **`SEAL` or `SEAL_SKILL`**,
and that the rejection log names which of the two blocked the cast. Add, in the same item, that
this gate runs before the animation delay and is therefore not re-evaluated after it.

Leave the sentence at `:279` ("`UseSkillGM` runs the same category dispatch without the
cooldown/MP/HP-threshold/probability/seal checks") **unchanged** — it remains accurate;
`UseSkillGM` gains no gate.

- [ ] **Step 2: Update the Skill Picker section**

Two edits in the `### Skill Picker` section:

- Item 1 (`:283`) currently reads "If the monster's template has no skills, or the monster is
  sealed, the sentinel decision is returned immediately." Change "the monster is sealed" to
  "the monster carries `SEAL` or `SEAL_SKILL`".
- Item 4 (`:286`) already lists `SEAL_SKILL` among the picker-relevant statuses. Add a clause
  noting that as of task-279 `SEAL_SKILL` also gates the picker itself, so a `SEAL_SKILL` apply
  or expire now changes the outcome rather than only triggering a re-pick that reproduces it
  (FR-6.5).

- [ ] **Step 3: Add the carnival family to the skill-category description**

Item 7 of the `UseSkill` list (`:276`) enumerates the dispatched categories. Add `CARNIVAL_BUFF`
(mob skill types 150–157) to the group routed through the stat-buff executor, alongside
stat-buff/immunity/reflect. State that the carnival family shares the stat-buff path and adds no
executor of its own.

- [ ] **Step 4: Verify no absolute paths and no invented claims**

Re-read the edited paragraphs. Every statement must be traceable to the code as landed in Tasks
1–5. Use repo-relative paths only.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/docs/domain.md
git commit -m "docs(atlas-monsters): document SEAL_SKILL gating and CARNIVAL_BUFF dispatch"
```

---

### Task 7: Full verification gate

**Files:**

- `tools/verify.sh` — read-only; the repo-wide gate

- [ ] **Step 1: Run the flagless verification gate**

Dispatch this to a `task-verifier` agent rather than running it in a large context.

```bash
tools/verify.sh
```

Expected: exit 0. `--quick` / `--no-docker` runs do **not** satisfy the "done means verified"
bar — they skip the bake and `-race`.

- [ ] **Step 2: Code review before the PR**

Run `superpowers:requesting-code-review`. `backend-guidelines-reviewer` applies (Go changes in
`libs/atlas-constants` and `services/atlas-monsters`); so does `plan-adherence-reviewer`. There
are no TypeScript changes, so `frontend-guidelines-reviewer` does not apply.

---

## Self-Review

**Spec coverage.** Every numbered change in design §5 is assigned: edits 1–3 → Task 1; edits 7,
8, 9 → Task 2; edits 4, 5 → Task 4; edit 6 → Task 5; domain doc → Task 6. Every test in design §8
is assigned: 1, 2, 3, 4, 5, 6 → Task 1; 7, 8, 9, 10 → Task 2; 11, 12, 13, 14, 15 → Task 3; 16, 17,
21 → Task 4; 18, 19, 20 → Task 5; 22 → Task 1 (the 145 row in
`TestSkillTypeToStatusName_Carnival`). Design §10 item 1 is resolved by user decision and
recorded in Task 6's scope note; items 2–4 need no work.

**Placeholder scan.** No TBD, no "similar to Task N", no "add appropriate error handling". Every
expected value — including `-990`, `20 * time.Minute`, `95`, and the eight `CARNIVAL_*` names — is
written out.

**Type consistency.** `skillSuppressingStatus(m Model) monster2.TemporaryStatType` is declared in
Task 4 and called with the same signature in Tasks 4 and 5. `testMobSkillLookup` keeps its
existing `func(skillId uint16, level uint16) (mobskill.Model, error)` type in Task 2. All
`TemporaryStatType*` and `SkillType*` symbols used are pre-existing and were read from
`libs/atlas-constants/monster/temporary_stat.go` and `skill.go` at plan time.
