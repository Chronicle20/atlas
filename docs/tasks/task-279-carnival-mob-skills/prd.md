# Monster Carnival Mob Skills (150–157) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
---

## 1. Overview

Mob skill types 150–157 — the Monster Carnival buff family (PAD, MAD, PDR, MDR, ACC, EVA,
SPEED, SEAL_SKILL) — are declared throughout `libs/atlas-constants/monster` but have no
execution path in any service. The constants exist
(`libs/atlas-constants/monster/skill.go:54–61`) and `SkillCategory` correctly classifies them
as `SkillCategoryCarnivalBuf` (`skill.go:237–241`, constant at `skill.go:13`), but nothing
consumes that category: a grep for `CarnivalBuf` across `services/` returns zero hits.

The result is a silent no-op with two independent causes, either of which alone would break
the skill:

1. **No dispatch arm.** `UseSkill`'s category switch
   (`services/atlas-monsters/atlas.com/monsters/monster/processor.go:949–959`) and
   `UseSkillGM`'s (`processor.go:1033–1043`) have arms for `StatBuff`, `Immunity`, `Reflect`,
   `Heal`, `Debuff`, and `Summon` only. `CarnivalBuf` falls through to
   `default: p.l.Warnf("Monster [%d] unknown skill category for skill [%d]")`.
2. **No status-name mapping.** `SkillTypeToStatusName` (`skill.go:71–95`) has no cases for
   150–157 and returns `""`. `executeStatBuff` opens with a `statusName == ""` guard that logs
   `"No status mapping for skill type [%d]"` and returns (`processor.go:1174–1179`). So even
   after fixing (1), the skill would still no-op without (2).

The packet layer is already complete and needs no change. `libs/atlas-packet/model/monster.go`
registers every stat this family requires in the mob temporary-stat bitfield — `ACCURACY`
(`:97`), `SPEED` (`:99`), `SEAL_SKILL` (`:119`), plus the `POWER_UP`/`MAGIC_UP`/
`POWER_GUARD_UP`/`MAGIC_GUARD_UP` entries already used by skills 100–113 — and `StatSet` /
`StatReset` (`libs/atlas-packet/monster/clientbound/stat.go`) encode the mask generically.
This is therefore a service-side mapping-and-dispatch gap confined to two Go modules.

## 2. Goals

Primary goals:

- Map all eight carnival skill types to their monster temporary stat, matching the reference
  server's mapping exactly.
- Route `SkillCategoryCarnivalBuf` through the existing stat-buff execution path in both
  `UseSkill` and `UseSkillGM`, so the eight skills apply a real, client-visible status effect
  instead of logging an "unknown skill category" warning.
- Close the `SEAL_SKILL` suppression gap so that skill 157 actually suppresses monster skill
  use, making it functional rather than cosmetic.
- Register the eight skills in the human-readable skill-name registry so seeds, GM tooling,
  and `SkillTypeNames()` can address them by name.

Non-goals:

- Monster Carnival game content of any kind: CP economy, CPQ maps, party/team model, Maple
  Coin rewards, Spiegelmann and Assistant NPC conversations (`2042000`–`2042009`), carnival
  shops, CP potions (`2022157`–`2022159`).
- Un-stubbing the decode-and-log carnival socket handler
  (`services/atlas-channel/atlas.com/channel/socket/handler/monster_carnival.go`) or emitting
  any of the eight registered carnival clientbound writers.
- Registering Monster Carnival as an `atlas-events` event type.
- Any change to `libs/atlas-packet` — the mob temporary-stat bitfield already carries every
  stat in scope.
- Any new autonomous mob behavior. See §4.5 for why the picker cannot select these skills.

## 3. User Stories

- As a GM, I want `UseSkillGM` on mob skills 150–157 to apply the corresponding monster
  temporary stat, so that I can validate carnival buff behavior in-client without CPQ content
  existing.
- As a player fighting a monster affected by SEAL_SKILL, I want that monster to stop using its
  skills for the duration, so that the debuff has an observable effect.
- As a service developer, I want a future Monster Carnival implementation to be able to cast
  these eight skills through the existing mob-skill path, so that CPQ work is content and
  orchestration only, not mob-skill plumbing.
- As an operator reading logs, I want carnival skill casts to stop emitting
  "unknown skill category" warnings, so that the warning retains diagnostic value.

## 4. Functional Requirements

### 4.1 Skill-type to temporary-stat mapping

**FR-1.1** `SkillTypeToStatusName` (`libs/atlas-constants/monster/skill.go:71`) MUST return the
following for each carnival skill type. The mapping is taken from the reference server,
`Cosmic src/main/java/server/life/MobSkill.java:205–208` and `:250–253`, where the carnival
types share their arms with the corresponding non-carnival buffs:

| Skill | Const | Cosmic `MonsterStatus` | Atlas `TemporaryStatType` |
|---|---|---|---|
| 150 | `SkillTypeCarnivalPAD` | `WEAPON_ATTACK_UP` | `TemporaryStatTypePowerUp` |
| 151 | `SkillTypeCarnivalMAD` | `MAGIC_ATTACK_UP` | `TemporaryStatTypeMagicUp` |
| 152 | `SkillTypeCarnivalPDR` | `WEAPON_DEFENSE_UP` | `TemporaryStatTypePowerGuardUp` |
| 153 | `SkillTypeCarnivalMDR` | `MAGIC_DEFENSE_UP` | `TemporaryStatTypeMagicGuardUp` |
| 154 | `SkillTypeCarnivalACC` | `ACC` | `TemporaryStatTypeAccuracy` |
| 155 | `SkillTypeCarnivalEVA` | `AVOID` | `TemporaryStatTypeAvoidability` |
| 156 | `SkillTypeCarnivalSpeed` | `SPEED` | `TemporaryStatTypeSpeed` |
| 157 | `SkillTypeCarnivalSealSkill` | `SEAL_SKILL` | `TemporaryStatTypeSealSkill` |

All eight target constants already exist in `libs/atlas-constants/monster/temporary_stat.go`
(lines 10, 11, 12, 18, 19, 20, 21, 32) and all eight are registered in the packet-side bitfield.

**FR-1.2** Skills 150 and 152 MUST reuse the same status names as skills 100/110 and 102/112
respectively. Carnival PAD is not a distinct stat from `POWER_UP`; it is the same client stat
reached by a different skill id, exactly as in the reference.

**FR-1.3** `SkillTypeToStatusName` MUST continue to return `""` for every skill type not
explicitly mapped. No default-case behavior change.

### 4.2 Skill-name registry

**FR-2.1** `skillNameMap` (`skill.go:145–179`) MUST gain eight entries using a `CARNIVAL_`
prefix:

```
"CARNIVAL_PAD"        → SkillTypeCarnivalPAD
"CARNIVAL_MAD"        → SkillTypeCarnivalMAD
"CARNIVAL_PDR"        → SkillTypeCarnivalPDR
"CARNIVAL_MDR"        → SkillTypeCarnivalMDR
"CARNIVAL_ACC"        → SkillTypeCarnivalACC
"CARNIVAL_EVA"        → SkillTypeCarnivalEVA
"CARNIVAL_SPEED"      → SkillTypeCarnivalSpeed
"CARNIVAL_SEAL_SKILL" → SkillTypeCarnivalSealSkill
```

**FR-2.2** The prefix is required, not stylistic. `skillNameMap` is a flat `string → uint16`
map; the reference server's bare names `SPEED` and `SEAL_SKILL` would sit ambiguously beside
the existing `SPEED_UP` (110) and `SEAL` (120) entries. The prefixed form also matches the
existing Go constant names `SkillTypeCarnival*`.

**FR-2.3** `SkillNameToId` and `SkillTypeNames` MUST resolve and list all eight new names with
no signature change. `SkillTypeNames` returns a sorted slice, so the eight names will appear
contiguously under `CARNIVAL_`.

### 4.3 Dispatch

**FR-3.1** `SkillCategoryCarnivalBuf` MUST be added to the existing stat-buff case arm in
`UseSkill` (`processor.go:949`) and `UseSkillGM` (`processor.go:1033`), alongside
`SkillCategoryStatBuff`, `SkillCategoryImmunity`, and `SkillCategoryReflect`, dispatching to
`executeStatBuff`.

**FR-3.2** No dedicated `executeCarnivalBuff` function is to be written. Reusing
`executeStatBuff` inherits the MP-cost check and deduction, cooldown registration, HP-threshold
gate, animation delay, and post-anim-delay alive guard without duplication.

**FR-3.3** `executeStatBuff` MUST NOT treat the carnival category as `Immunity` or `Reflect`.
Its `oppositeImmunity` pre-cancel is keyed on `category == SkillCategoryImmunity`
(`processor.go:1193`) and its reflect branch on `category == SkillCategoryReflect`; both remain
false for `CarnivalBuf`, so carnival skills take the plain `NewStatusEffect` path. This
requires no code change, but MUST be asserted by test.

**FR-3.4** The `default:` arm's "unknown skill category" warning MUST remain for genuinely
unmapped categories.

### 4.4 Stacking and value semantics

**FR-4.1** Re-applying an already-active carnival buff MUST refresh it — overwrite both value
and duration. This is the existing `SkillCategoryStatBuff` behavior and requires no change: the
already-active reject gate at `processor.go:922` is keyed on `Immunity` and `Reflect` only.

**FR-4.2** Carnival buffs MUST NOT be added to the already-active reject gate at
`processor.go:922`.

**FR-4.3** The stat value passed to the status effect is the skill's `X()` verbatim. It MUST
NOT be clamped, floored at zero, or sign-normalized. This is load-bearing: mob skill 155
(`CarnivalEVA`) has a **level 2** whose `x` is `-990` — a negative avoidability, i.e. a debuff.
Clamping would silently convert a large debuff into a no-op. (Source: `Skill.wz/MobSkill.img`,
skill 155 level 2, `x=-990`, `time=180`, `info=리치` / "Lich".)

**FR-4.4** Duration MUST continue to be read as `time.Duration(sd.Duration()) * time.Millisecond`.
The WZ `time` field is authored in seconds and converted to milliseconds during ingest in
`atlas-data` (see `services/atlas-data/atlas.com/data/mobskill/reader_test.go:11`); the
existing `executeStatBuff` conversion is already correct and must not be changed.

### 4.5 AoE targeting

**FR-5.1** `IsAoeSkill` (`skill.go:99`) MUST return `true` for skill types 150–157, so that
`executeStatBuff`'s bounding-box loop is reachable for the carnival family when the skill
declares a box.

**FR-5.2** The existing `IsAoeSkill(...) && sd.HasBoundingBox()` conjunction in
`executeStatBuff` MUST be preserved unchanged. A carnival skill with no bounding box applies to
the caster only.

**FR-5.3 (deliberate reference deviation, must be recorded in code comment).** This diverges
from the reference server, where the carnival arms feed `applyMonsterBuffs` with a single
monster and only the `*_M` and `HEAL_M` arms consult `lt`/`rb`
(`Cosmic MobSkill.java:205–208, 250–253` vs `:264–275`). It also has **no observable effect on
current data**: none of the eight WZ entries declares `lt` or `rb` — every level of 150–157
carries only `x`, `time`, `mob/`, and (for 157) `effect/`, so `HasBoundingBox()` is
unconditionally false and the AoE loop never executes. FR-5.1 is forward-compatibility for
custom or later-version skill data, chosen deliberately over reference parity. The
implementation MUST NOT claim reference parity for this behavior.

### 4.6 SEAL_SKILL suppression

**FR-6.1** A monster carrying the `SEAL_SKILL` temporary stat MUST NOT use mob skills. Today
nothing enforces this: both the picker gate (`picker.go:124`) and the executor gate
(`processor.go:866`) test `HasStatusEffect("SEAL")` only. Reference:
`Cosmic Monster.java:1457` — `if (toUse == null || isBuffed(MonsterStatus.SEAL_SKILL))`.

**FR-6.2** The gate in `pickNextSkill` (`picker.go:124`) MUST reject a monster carrying
`SEAL_SKILL` as well as one carrying `SEAL`.

**FR-6.3** The gate in `UseSkill` (`processor.go:866`) MUST likewise reject on `SEAL_SKILL`, so
that a skill already picked and in flight is dropped if `SEAL_SKILL` lands during the animation
delay.

**FR-6.4** Both gates MUST reference `monster2.TemporaryStatTypeSealSkill` /
`monster2.TemporaryStatTypeSeal` rather than the current bare `"SEAL"` string literal.

**FR-6.5** No change is required to `pickerRelevantStatuses` (`picker.go:69–76`), which already
lists both `TemporaryStatTypeSeal` and `TemporaryStatTypeSealSkill`. Before this task,
`SEAL_SKILL`'s presence there was inert — a status declared to flip picker eligibility that no
gate consulted. FR-6.2 makes that declaration true.

**FR-6.6** SEAL_SKILL MUST NOT suppress the monster's basic attack (`UseBasicAttack`,
`processor.go:1051`). The reference gate is on skill selection only.

### 4.7 Observability

**FR-7.1** Casting a carnival skill MUST NOT emit the "unknown skill category" warning.

**FR-7.2** A skill-use attempt rejected by the `SEAL_SKILL` gate MUST log at debug level,
naming `SEAL_SKILL` distinctly from `SEAL`, so operators can tell the two rejections apart.

## 5. API Surface

No REST endpoint is added, removed, or modified.

`atlas-data`'s existing mob-skill endpoints already serve skill types 150–157 unchanged:

- `GET /data/mob-skills` (`mobskill/resource.go:24`)
- `GET /data/mob-skills/{skillId}` (`:25`)
- `GET /data/mob-skills/{skillId}/{level}` (`:26`)

These are generic over skill id and require no per-type registration. No JSON:API document
shape changes.

## 6. Data Model

No new entity, no schema migration, no new field.

The mob-skill records for types 150–157 are ingested from `Skill.wz/MobSkill.img` by the
existing `atlas-data` reader and stored in the existing mob-skill table. Confirmed present in
the WZ data with these values:

| Skill | Levels | `x` | `time` (s) | `info` | Extras |
|---|---|---|---|---|---|
| 150 | 1 | 40 | 1200 | 몬스터카니발 | — |
| 151 | 1 | 50 | 1200 | 몬스터카니발 | — |
| 152 | 1 | 50 | 1200 | 몬스터카니발 | — |
| 153 | 1 | 50 | 1200 | 몬스터카니발 | — |
| 154 | 1 | 50 | 1200 | 몬스터카니발 | — |
| 155 | 2 | 30 / **−990** | 1200 / 180 | — / 리치 | level 2 is a Lich debuff |
| 156 | 1 | 50 | 1200 | 몬스터카니발 | — |
| 157 | 1 | 1 | 180 | 리치 | has `effect/` node |

No entry declares `lt`, `rb`, `prop`, `mpCon`, `interval`, `hp`, or `count`. Two consequences
carried into the requirements: FR-5.3 (bounding box always absent) and FR-8.4 (`prop` always
absent).

The `TemporaryStatType` value set (`libs/atlas-constants/monster/temporary_stat.go`) already
contains all eight targets; no constant is added.

## 7. Service Impact

**`libs/atlas-constants`** (module `libs/atlas-constants/go.mod`) — `monster/skill.go` only:
eight cases added to `SkillTypeToStatusName`, eight entries added to `skillNameMap`, eight
types added to `IsAoeSkill`. No new constant, no signature change, no new file.

**`services/atlas-monsters`** (module `services/atlas-monsters/atlas.com/monsters/go.mod`):
- `monster/processor.go` — `SkillCategoryCarnivalBuf` added to two case arms (`:949`, `:1033`);
  the `SEAL` gate at `:866` broadened to `SEAL_SKILL` (FR-6.3).
- `monster/picker.go` — the `SEAL` gate at `:124` broadened to `SEAL_SKILL` (FR-6.2).
- `monster/processor_test.go`, `monster/picker_test.go` — new coverage per §10.
- `docs/domain.md` — the picker-relevant-status note at `:287` becomes accurate for
  `SEAL_SKILL` once FR-6.2 lands; update the surrounding skill-gating description.

**Not touched:** `libs/atlas-packet` (bitfield already complete), `services/atlas-channel`
(consumes status effects generically), `services/atlas-data` (endpoints generic over skill id),
`services/atlas-buffs` (monster stats are not character buffs), all seed data.

## 8. Non-Functional Requirements

**NFR-1 Multi-tenancy.** Every touched path is already tenant-scoped through `p.t` /
`tenant.Model` — `GetMonsterRegistry().GetMonster(p.t, ...)`, `GetCooldownRegistry()`,
`MonsterTemporaryStatTypeByName(t)`. No new tenant-scoping work; no path may be introduced that
resolves a stat or registry entry without a tenant.

**NFR-2 Version compatibility.** Mob temporary-stat bit positions are tenant/version-resolved
in `libs/atlas-packet/model/monster.go` (`MonsterTemporaryStatTypeByName(t)`, `legacyMobStatMask(t)`).
Because this task adds no bitfield entry, all currently supported versions are unaffected. The
implementation MUST NOT hard-code a bit position or mask.

**NFR-3 No performance change.** The work is a map lookup and two additional case labels on
paths that already run per skill cast. The FR-6 gates add one `HasStatusEffect` call per
picker iteration and per skill use — the same order as the existing `SEAL` check.

**NFR-4 No security surface.** No new external input is parsed; skill ids arrive on paths that
already validate them.

**NFR-5 Guidelines conformance.** Changes follow the `backend-dev-guidelines` skill; models
stay immutable and no test-only constructor is introduced (Builder pattern for any test setup
needing it).

## 9. Open Questions

**OQ-1 — No in-game caster exists (accepted, not blocking).** None of the eight WZ entries
declares `prop`, and `pickNextSkill` skips any skill with `prop <= 0`
(`picker.go:188–190`). The autonomous picker therefore can never select 150–157. With CPQ
content out of scope (§2), the only reachable caster after this task is `UseSkillGM`. This is
understood and accepted: the deliverable is a correct, tested execution path for a future CPQ
implementation to drive, not new autonomous mob behavior. Acceptance criteria are written
against `UseSkillGM` and unit tests accordingly.

**OQ-2 — Skill 155 level 2 is not carnival content.** Level 2 (`x=-990`, `info=리치`/"Lich")
appears to be a Lich mob debuff reusing the type id, not a carnival buff. It is served by the
same `SkillTypeToStatusName` mapping and is correct under FR-4.3 (no clamping), but it is worth
confirming at design time that no Lich mob's WZ skill list references `155/2` with a `prop` that
would make it picker-selectable — which would mean this task changes live mob behavior, not just
GM-reachable behavior. Resolve before implementation.

**OQ-3 — Client rendering of ACCURACY/AVOIDABILITY.** These two stats are registered in the mob
temporary-stat bitfield but, as far as this spec verified, are not currently set by any other
mob skill. Whether the client renders a visible indicator for them (as opposed to silently
applying them) is unverified. Not blocking: the server-side contract is the deliverable.

**OQ-4 — `SEAL` vs `SEAL_SKILL` semantics.** This task makes `SEAL_SKILL` suppress skill use
(FR-6). Whether the pre-existing `SEAL` gate should *also* remain is assumed yes (no regression
intended), but the reference gates on `SEAL_SKILL` alone at `Monster.java:1457`. Keeping both is
the conservative choice taken here; flag if design finds `SEAL` gating to be an Atlas-specific
divergence worth revisiting separately.

## 10. Acceptance Criteria

Mapping and registry:

- [ ] `SkillTypeToStatusName` returns the exact stat from the FR-1.1 table for each of the eight
      ids; a table-driven test asserts all eight.
- [ ] `SkillTypeToStatusName` still returns `""` for an unmapped id (e.g. 149).
- [ ] `SkillNameToId` resolves all eight `CARNIVAL_*` names to the correct ids.
- [ ] `SkillTypeNames()` includes all eight names.
- [ ] `IsAoeSkill` returns `true` for 150–157 and its pre-existing results are unchanged for
      every other id.

Dispatch:

- [ ] `UseSkill` with a carnival skill id reaches `executeStatBuff` and applies a status effect
      carrying the mapped stat name; asserted for at least ids 150, 154, and 157.
- [ ] `UseSkillGM` with a carnival skill id applies the same status effect.
- [ ] Neither path logs "unknown skill category" for any of the eight ids.
- [ ] A carnival cast produces a plain `NewStatusEffect`, not a reflect effect, and triggers no
      opposite-immunity pre-cancel (FR-3.3).

Values and stacking:

- [ ] A carnival buff applied with `X() = -990` stores `-990` — not `0`, not `990` (FR-4.3).
- [ ] Re-casting an active carnival buff refreshes value and duration rather than being rejected
      (FR-4.1).
- [ ] Duration is `sd.Duration()` milliseconds; a skill with `time=1200` seconds ingested as
      `1200000` ms yields a 20-minute effect, not 20 minutes × 1000.

AoE:

- [ ] A carnival skill with no bounding box applies to the caster only.
- [ ] A carnival skill with a synthetic bounding box applies to in-box monsters and skips
      out-of-box monsters.
- [ ] The FR-5.3 deviation is recorded in a code comment at the `IsAoeSkill` change site.

SEAL_SKILL:

- [ ] `pickNextSkill` returns no decision for a monster carrying `SEAL_SKILL`.
- [ ] `UseSkill` rejects a skill on a monster carrying `SEAL_SKILL`, logging a message distinct
      from the `SEAL` rejection.
- [ ] `UseBasicAttack` still succeeds for a monster carrying `SEAL_SKILL` (FR-6.6).
- [ ] Casting skill 157 on monster A, then attempting a skill on A, is rejected end-to-end.

Gates:

- [ ] `go build ./...` and `go test ./...` pass in both touched modules.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` and `task-reviewer` run before the PR is opened.
- [ ] `services/atlas-monsters/docs/domain.md` reflects the new `SEAL_SKILL` gating.
- [ ] `docs/research/missing-features/monsters-and-bosses.md` §8 updated to reflect the closed
      gap, including the OQ-1 caveat that no autonomous caster exists.

## Appendix A — Evidence

Every claim of absence below states the command run. All line numbers are pinned against this
worktree at branch point `bda6566f3`.

- Zero consumers: `grep -rn "CarnivalBuf" services/` → no matches.
- Dispatch gap: `processor.go:949–959` (UseSkill switch), `processor.go:1033–1043` (UseSkillGM
  switch) — no `CarnivalBuf` arm in either.
- Mapping gap: `skill.go:71–95` — `SkillTypeToStatusName` has no case above
  `SkillTypeMagicCounter` (144); `executeStatBuff` empty-name guard at `processor.go:1174–1179`.
- Classification present: `skill.go:237–241` returns `SkillCategoryCarnivalBuf`; constant at
  `skill.go:13`.
- Packet support present: `libs/atlas-packet/model/monster.go:97` (`ACCURACY`), `:99` (`SPEED`),
  `:119` (`SEAL_SKILL`); generic mask encode at `:242–256`; `StatSet`/`StatReset` at
  `libs/atlas-packet/monster/clientbound/stat.go:21–110`.
- SEAL_SKILL declared-but-unenforced: `picker.go:71` lists it in `pickerRelevantStatuses`; the
  only gates are `picker.go:124` and `processor.go:866`, both testing `"SEAL"`.
- Picker cannot select these skills: `picker.go:188–190` (`prop <= 0` → `continue`) combined
  with the absence of `prop` in all eight WZ entries.
- Reference mapping (paths relative to a local Cosmic checkout, `<cosmic-root>`):
  `<cosmic-root>/src/main/java/server/life/MobSkill.java:205–208, 250–253`; reference SEAL_SKILL
  gate at `<cosmic-root>/src/main/java/server/life/Monster.java:1457`; reference id list at
  `<cosmic-root>/src/main/java/server/life/MobSkillType.java:39–46`.
- WZ data: `Skill.wz/MobSkill.img` (read from a local WZ extraction outside this repo) — ids
  150–157 all present, field inventory and values as tabulated in §6. Cross-checked against a
  second independent extraction of the same img; the id set is identical in both.
- Baseline: `go test ./...` in `services/atlas-monsters/atlas.com/monsters` and
  `libs/atlas-constants` — both clean at branch creation, no failures.
