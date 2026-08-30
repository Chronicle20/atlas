# Monster Carnival Mob Skills (150–157) — Design

Version: v1
Status: Draft
Created: 2026-08-28
Input: `docs/tasks/task-279-carnival-mob-skills/prd.md` (v1, approved)
Branch point: `bda6566f3`

---

## 0. Summary

The PRD describes a mapping-and-dispatch gap, not a new subsystem. Every mechanism the
carnival family needs — the mob temporary-stat bitfield, `executeStatBuff`, the status
registry, the cooldown registry, the picker's status-relevance set — already exists and is
exercised by skills 100–145. The work is to make eight already-classified skill ids reach
those mechanisms, plus one genuine behavior addition (`SEAL_SKILL` suppression).

Design posture, therefore, is **extend in place, add no new abstraction**. No new file, no new
constant, no new interface, no new event type, no new package. The only structural addition is
a single unexported helper (D6) that exists solely so two call sites cannot drift apart.

Three things in the PRD needed resolution or correction before implementation. All three are
resolved here against primary evidence:

- **OQ-1 / OQ-2 are now closed** with WZ evidence (§1). One mob in the game references the
  family; it is not the one the PRD worried about, and it still cannot fire the skill.
- **FR-6.3's stated rationale is wrong** about where the gate sits relative to the animation
  delay (§4). The requirement is kept; the rationale is corrected and the scope question it
  raises is decided explicitly.
- **`UseSkillGM` has no injectable skill lookup**, so PRD acceptance criterion
  "`UseSkillGM` with a carnival skill id applies the same status effect" is not testable as the
  code stands (§7, D8). A one-line seam extension fixes it.

---

## 1. Evidence resolved at design time

### 1.1 OQ-2 — Does any mob reference skill 155 level 2 (the Lich `x=-990` debuff)?

**No.** Resolved by sweeping every mob definition, not spot-checking.

```
$ cd <wz-root>/Mob.wz && grep -l 'name="skill" value="15[0-7]"' *.xml
9400593.img.xml
$ grep -l 'name="skill" value="15[0-7]"' *.xml | wc -l
1
```

Exactly one mob in the entire `Mob.wz` extraction references any of skills 150–157, and its
reference is:

```xml
<imgdir name="skill">
  <imgdir name="0">
    <int name="skill" value="156"/>
    <int name="action" value="1"/>
    <int name="level" value="1"/>
    <int name="effectAfter" value="1600"/>
  </imgdir>
</imgdir>
```

Mob `9400593` is `Hsalf` (`String.wz/Mob.img.xml:3111–3112`), a level-130 boss
(`boss=1`, `maxHP=233000000`). It declares **156 level 1** — `CarnivalSpeed`, `x=50`,
`time=1200` — not 155 level 2.

**Consequences:**

- OQ-2 is closed with no residual risk. Skill `155/2` is referenced by no mob's skill list. It
  is reachable only through `UseSkillGM`. FR-4.3's no-clamping rule remains correct and remains
  worth testing, but it cannot change live mob behavior.
- OQ-1 is **narrowed but confirmed**. There *is* a mob carrying a carnival skill, which the
  PRD did not know. It still cannot fire it: `MobSkill.img` entry `156/level/1` declares only
  `x`, `time`, `mob/`, and `info`. With no `prop`, `pickNextSkill` hits `prop <= 0 → continue`
  (`picker.go:188–190`) and the skill is never selected. Hsalf's `156` is dead template data
  under the current picker.
- This makes the picker's `prop` gate load-bearing for a behavior claim, so §8 requires a
  regression test pinning it: a monster whose skill list contains 156 with `prop=0` must yield
  the sentinel decision. Without that test, a future "default prop to 100" change would
  silently give a level-130 boss a permanent +50 speed buff with no failing test.

### 1.2 FR-1.1 — Reference mapping re-derived, not trusted

Read directly from `<cosmic-root>/src/main/java/server/life/MobSkill.java:205–208, 250–253`:

```java
case ATTACK_UP, ATTACK_UP_M, PAD        -> stats.put(MonsterStatus.WEAPON_ATTACK_UP, x);
case MAGIC_ATTACK_UP, MAGIC_ATTACK_UP_M, MAD -> stats.put(MonsterStatus.MAGIC_ATTACK_UP, x);
case DEFENSE_UP, DEFENSE_UP_M, PDR      -> stats.put(MonsterStatus.WEAPON_DEFENSE_UP, x);
case MAGIC_DEFENSE_UP, MAGIC_DEFENSE_UP_M, MDR -> stats.put(MonsterStatus.MAGIC_DEFENSE_UP, x);
...
case ACC        -> stats.put(MonsterStatus.ACC, x);
case EVA        -> stats.put(MonsterStatus.AVOID, x);
case SPEED      -> stats.put(MonsterStatus.SPEED, x);
case SEAL_SKILL -> stats.put(MonsterStatus.SEAL_SKILL, x);
```

The FR-1.1 table is confirmed verbatim, including the shared-arm structure for 150–153 (FR-1.2).

### 1.3 FR-6.1 — Reference gate re-derived

`<cosmic-root>/src/main/java/server/life/Monster.java:1456–1458`:

```java
public boolean canUseSkill(MobSkill toUse, boolean apply) {
    if (toUse == null || isBuffed(MonsterStatus.SEAL_SKILL)) {
        return false;
    }
```

Confirmed. Note the reference gates on `SEAL_SKILL` **only** — there is no monster-side `SEAL`
gate in the reference at all, because in the reference `SEAL` is a character `Disease`, not a
`MonsterStatus`. This is the basis of the OQ-4 decision in §5.

### 1.4 WZ field inventory for 150–157

Read from `<wz-root>/Skill.wz/MobSkill.img.xml` (offsets 487235–491400). The PRD's §6 table is
confirmed exactly, including that no entry declares `lt`, `rb`, `prop`, `mpCon`, `interval`,
`hp`, or `count`. Skill 157 alone carries an `effect/` node; 155 alone has two levels.

---

## 2. Alternatives considered

### A. Mapping placement — extend the switch vs. introduce a table

`SkillTypeToStatusName` (`skill.go:71–95`) is a hand-written switch. The alternative was to
replace it with a `map[uint16]TemporaryStatType`, which would make the eight additions data
rather than code and would remove the risk of a fall-through mistake.

**Rejected.** The switch is the file's established idiom (`SkillTypeToDiseaseName`,
`SkillCategory`, `ReflectKindForSkill` are all switches), and shared arms — `case
SkillTypeWeaponAttackUp, SkillTypeWeaponAttackUpAoe:` — express FR-1.2's "same stat, different
id" relationship more legibly than eight independent map rows would. Converting the function is
unrelated refactoring; the PRD's §7 explicitly scopes this to "eight cases added". Extend the
switch.

### B. Dispatch — new `executeCarnivalBuff` vs. reuse `executeStatBuff`

FR-3.2 already forbids a dedicated function. Verified that reuse is in fact correct rather than
merely convenient by reading `executeStatBuff` (`processor.go:1174–1260`):

- Its `oppositeImmunity` pre-cancel is guarded by `category == SkillCategoryImmunity`
  (`:1193`) — false for `CarnivalBuf`, so `oppositeImmunity` stays `""` and the pre-cancel
  block is skipped entirely.
- Its reflect branch is guarded by `category == SkillCategoryReflect` (`:1218`) — false, so
  the plain `NewStatusEffect` path is taken.
- Everything else in the function (`statuses` map, ms duration, `applyBuff`, the AoE loop) is
  category-agnostic.

So the carnival category traverses `executeStatBuff` on the plainest possible path. Reuse
confirmed. FR-3.3's "requires no code change but MUST be asserted by test" is the right call —
these two guards are exactly the kind of invariant a future refactor breaks silently.

### C. `SEAL_SKILL` gate placement (see §4 for the decision)

Three candidate placements were considered: (i) at the two existing gate sites only, (ii) also
inside `applyAnimationDelayedEffect`, (iii) inside `executeStatBuff`/each executor. Decided in
§4.

### D. Where the eight `CARNIVAL_*` names buy anything

Traced the consumer to confirm FR-2.1 is not decorative. `skillNameMap` feeds `SkillNameToId`,
whose only production caller is the `@mobstatus` GM command
(`services/atlas-messages/.../command/monster/commands.go:56`):

```go
} else if id, ok := monster2.SkillNameToId(strings.ToUpper(input)); ok {
    skillId = id
}
```

which produces `UseSkillFieldCommandProvider` → `USE_SKILL_FIELD` → `handleUseSkillFieldCommand`
(`kafka/consumer/monster/consumer.go:373–387`) → `p.UseSkillGM(...)` **for every monster in the
map**. So `@mobstatus CARNIVAL_PAD` is the concrete OQ-1 caster, and the unknown-skill error
path already lists `SkillTypeNames()` back to the GM, which is why FR-2.3 matters. No change to
atlas-messages is required for this to work.

---

## 3. Design decisions

### D1 — Extend `SkillTypeToStatusName` with four shared arms and four singles

Add to the existing switch, placed after the `SkillTypeMagicCounter` arm to preserve ascending
id order:

```go
case SkillTypeWeaponAttackUp, SkillTypeWeaponAttackUpAoe, SkillTypeCarnivalPAD:
    return TemporaryStatTypePowerUp
```

i.e. the four carnival ids that share a stat with an existing skill (150/151/152/153) are
folded into the **existing** arms rather than given new ones. This makes FR-1.2 structurally
true — "carnival PAD is not a distinct stat" — instead of true only by coincidence of two arms
returning the same constant. The remaining four get their own arms:

| Added to | Ids | Returns |
|---|---|---|
| existing `WeaponAttackUp` arm | + 150 | `TemporaryStatTypePowerUp` |
| existing `MagicAttackUp` arm | + 151 | `TemporaryStatTypeMagicUp` |
| existing `WeaponDefenseUp` arm | + 152 | `TemporaryStatTypePowerGuardUp` |
| existing `MagicDefenseUp` arm | + 153 | `TemporaryStatTypeMagicGuardUp` |
| new arm | 154 | `TemporaryStatTypeAccuracy` |
| new arm | 155 | `TemporaryStatTypeAvoidability` |
| folded into existing `SpeedUp` arm | + 156 | `TemporaryStatTypeSpeed` |
| new arm | 157 | `TemporaryStatTypeSealSkill` |

156 folds into the existing `case SkillTypeSpeedUp:` arm for the same reason. The `default:
return ""` is untouched (FR-1.3).

**Deliberate non-change:** skill 145 (`SkillTypePhysicalMagicCounter`) is classified
`SkillCategoryReflect` but has no `SkillTypeToStatusName` arm, so it hits the
`statusName == ""` guard and no-ops today. That is a pre-existing defect of the same shape as
this task's, discovered while reading the switch. It is **out of scope** and must not be fixed
here — it is a reflect skill with different semantics (the reference sets four stats plus a
reflection value, `MobSkill.java:243–249`), not a one-line addition. Record it; do not touch it.

### D2 — Add `SkillCategoryCarnivalBuf` to the existing stat-buff arm in both switches

```go
case monster2.SkillCategoryStatBuff, monster2.SkillCategoryImmunity,
     monster2.SkillCategoryReflect, monster2.SkillCategoryCarnivalBuf:
    p.executeStatBuff(m, sd, skillId, skillLevel)
```

at `processor.go:949` (`UseSkill`) and `processor.go:1033` (`UseSkillGM`). Two call sites, same
edit. `default:` is untouched (FR-3.4).

### D3 — `IsAoeSkill` gains 150–157, with the deviation recorded at the site

FR-5.1 as written. Verified `IsAoeSkill` has exactly one caller repo-wide
(`processor.go:1253`), inside the `IsAoeSkill(...) && sd.HasBoundingBox()` conjunction, so
widening it cannot leak into any other behavior:

```
$ grep -rn --include='*.go' "IsAoeSkill" services libs
services/atlas-monsters/.../monster/processor.go:1253
libs/atlas-constants/monster/skill.go:98,99
```

Because no WZ entry for 150–157 declares `lt`/`rb` (§1.4), `HasBoundingBox()` is
unconditionally false for this family and the change is **inert on all current data**. It is
forward-compatibility for custom or later-version data, and it is a knowing divergence from the
reference, where only the `*_M` and `HEAL_M` arms consult `lt`/`rb`. Per FR-5.3 the code
comment at the change site must say so plainly and must not claim reference parity.

The alternative — leave `IsAoeSkill` alone — was considered and rejected only because the PRD
decided it. Noting the tradeoff honestly: leaving it alone would be closer to the reference and
equally inert today. The PRD's forward-compat argument is reasonable; the comment is what makes
the choice reviewable later.

### D4 — No change to the already-active reject gate

FR-4.1/FR-4.2 require refresh-on-recast. Verified this is already guaranteed two layers down
rather than merely un-blocked: `Registry.ApplyStatusEffect` → `Model.ApplyStatus` →
`Builder.AddStatusEffect` (`monster/builder.go:181–204`), which for every non-`VENOM` status
calls `b.RemoveStatusEffectByType(statusType)` before appending. Recasting a carnival buff
therefore evicts the prior effect and installs the new value and duration. No code change; the
behavior is asserted by test (§8).

### D5 — Values and duration pass through unchanged

`statuses := map[string]int32{string(statusName): sd.X()}` and
`time.Duration(sd.Duration()) * time.Millisecond` (`processor.go:1181–1185`) already satisfy
FR-4.3 and FR-4.4. `sd.X()` is `int32`, so `-990` survives with no clamp on the path. The
requirement is a **prohibition on adding** normalization, plus test coverage; there is nothing
to build.

### D6 — One helper for the two `SEAL`/`SEAL_SKILL` gates

The gate logic lands in two files that must not drift. Rather than duplicating a two-status
condition, add an unexported helper in the `monster` package:

```go
// skillSuppressingStatus returns the name of the status blocking the monster
// from using skills, or "" if none. SEAL_SKILL is the reference gate
// (Cosmic Monster.java:1457); SEAL is retained as a pre-existing Atlas
// behavior (task-279 design D7).
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

It returns the blocking status rather than a `bool` precisely so each caller can satisfy FR-7.2
by naming `SEAL_SKILL` distinctly from `SEAL` in its own log line, without the helper knowing
anything about logging. Both call sites become:

```go
if st := skillSuppressingStatus(m); st != "" {
    p.l.Debugf("Monster [%d] has [%s] and cannot use skill [%d].", uniqueId, st, skillId)
    return
}
```

`SEAL_SKILL` is checked first so that when both are present the more specific status is the one
reported.

**Placement:** `picker.go`, beside `pickerRelevantStatuses` and `isPickerRelevantStatus`, which
is where the package already keeps status-classification predicates. `processor.go` is 1600+
lines and adding to it is the worse of the two.

This also satisfies FR-6.4 — both sites move off the bare `"SEAL"` literal
(`processor.go:866`, `picker.go:124`) onto `monster2.TemporaryStatType*`.

**Note on a duplicate constant set:** `libs/atlas-constants/monster` carries *two* names for the
same wire string — `status.go:8` `StatusSeal = "SEAL"` and `temporary_stat.go:16`
`TemporaryStatTypeSeal = "SEAL"`. FR-6.4 picks `TemporaryStatType*`, which is correct: it is
what `pickerRelevantStatuses` (`picker.go:70–71`) already uses, so the gate and the relevance
set will reference the same symbols. The duplication itself is pre-existing and out of scope —
do not consolidate it here.

### D7 — Keep the `SEAL` gate alongside `SEAL_SKILL` (resolves OQ-4)

The reference has no monster-side `SEAL` gate at all (§1.3). Removing Atlas's would be closer to
reference parity. **Rejected — keep both**, for a reason stronger than caution:

Atlas's monster `SEAL` gate is *reachable*. `@mobclear`'s `validStatuses` list
(`services/atlas-messages/.../command/monster/commands.go:26–34`) includes `"SEAL"` as a
monster status, and `libs/atlas-packet/model/monster.go:103` registers `SEAL` in the mob
temporary-stat bitfield. Whether or not a production path currently *applies* it, removing the
gate would be a behavior change to an operator-visible surface, made as a side effect of an
unrelated task. That is exactly the kind of quiet scope creep the PRD's non-goals exist to
prevent. OQ-4 is answered: keep both, and record here that dropping `SEAL` is a defensible
separate change nobody has justified yet.

### D8 — Extend the `testMobSkillLookup` seam to `UseSkillGM`

**This is the one place the PRD's acceptance criteria are not satisfiable against the code as
written.** `testMobSkillLookup` (`processor.go:89–91`) is honored only by `UseSkill`
(`processor.go:873–877`). `UseSkillGM` calls the REST processor unconditionally:

```go
sd, err := mobskill.NewProcessor(p.l, p.ctx).GetByIdAndLevel(uint16(skillId), uint16(skillLevel))
```

(`processor.go:1018`). So the criterion "`UseSkillGM` with a carnival skill id applies the same
status effect" cannot be tested — the call would attempt real HTTP.

Fix: give `UseSkillGM` the identical seam check `UseSkill` already has, so the two entry points
resolve skill data the same way. Three lines, symmetric with existing code, and it removes a
latent asymmetry rather than adding a test-only construct. It does not introduce a
`*_testhelpers.go` constructor and does not violate the Builder-pattern rule — it extends an
existing production-nil hook to a second caller.

Considered and rejected: testing `executeStatBuff` directly and asserting the GM switch by
inspection. That would leave the GM dispatch arm — one of the two edits this task exists to
make — with no test at all.

### D9 — `atlas-messages` GM status lists: leave alone, record the gap

`validStatuses` (`commands.go:26–34`) governs `@mobclear <status>` only. It does **not** contain
`SEAL_SKILL`, `ACCURACY`, or `AVOIDABILITY`, so after `@mobstatus CARNIVAL_SEAL_SKILL` a GM
cannot clear the status by name and must wait out the 180 s duration (or use bare `@mobclear`).

Not fixed here. It is a GM-ergonomics gap in a service the PRD does not scope, the durations
involved are short, and expanding that list is a one-line change any follow-up can make with
its own review. Recorded so it is a known limitation rather than a surprise during manual
verification.

Related, also not fixed: `isBossAllowedStatus` (`processor.go:1571–1585`) omits `ACCURACY`,
`AVOIDABILITY`, and `SEAL_SKILL`. This is **inert for this task** — the boss filter runs only
`if effect.SourceType() == SourceTypePlayerSkill` (`processor.go:1494`), and every carnival
effect is `SourceTypeMonsterSkill`. Hsalf (a boss, §1.1) would therefore receive its own
`156` buff if it could ever cast it. Do not touch the boss list.

---

## 4. The `SEAL_SKILL` gate — FR-6.3's rationale corrected, and its scope decided

FR-6.3 says the `UseSkill` gate should be broadened "so that a skill already picked and in
flight is dropped if `SEAL_SKILL` lands during the animation delay."

**That is not what broadening `processor.go:866` does.** Reading the function, the ordering is:

```
866   seal gate                     ← runs FIRST, synchronously
872   fetch skill definition
883   cooldown / HP / MP gates
916   MP deduction, cooldown registration
945   executeEffect := func(){...}  ← closure, not yet run
978   if animDelay > 0 { go { sleep(animDelay); applyAnimationDelayedEffect(...) } }
```

The gate at `:866` runs strictly **before** the animation delay begins. It cannot observe a
status that lands during that delay. The post-delay re-fetch happens in
`applyAnimationDelayedEffect` (`processor.go:988–1005`), which today re-checks presence and
`Alive()` and nothing else.

**Decision: implement the gate at `:866` as specified, and do NOT add a post-delay re-check.**

Reasoning:

1. Every FR-6 acceptance criterion is satisfied by the `:866` gate. The criterion is
   "`UseSkill` rejects a skill on a monster carrying `SEAL_SKILL`" — a pre-delay condition.
   Nothing in §10 tests the in-flight case.
2. `applyAnimationDelayedEffect` is shared by *all* skill categories. Adding a suppression
   check there would newly cause `SEAL` — not just `SEAL_SKILL` — to cancel in-flight debuffs,
   heals, and summons, which no requirement asks for and which is a live behavior change to
   skills 120–136 and 200. That is scope the PRD did not authorize.
3. The reference does not do it either. `canUseSkill` is consulted at selection time
   (`Monster.java:1456`), not after the animation.

So: FR-6.3's *requirement* is implemented; its *rationale* is wrong and is replaced by "the
executor gate mirrors the picker gate, so a decision that went stale between pick and cast is
rejected at cast time." The in-flight case is an explicit, recorded non-goal. If it is wanted
later it is a separate change with its own justification, because it must decide the `SEAL`
question in point 2.

FR-6.6 needs no work: `UseBasicAttack` (`processor.go:1051`) has no seal gate today and gains
none. A test pins it.

FR-6.5 needs no work either: `pickerRelevantStatuses` (`picker.go:69–76`) already lists both
statuses. This task makes that pre-existing declaration meaningful — before it, `SEAL_SKILL`
triggered a re-pick that could not change the outcome. Worth stating in the domain doc.

---

## 5. Change inventory

Nine edits across two modules. No new file except tests.

**`libs/atlas-constants` — `monster/skill.go` only**

| # | Site | Change |
|---|---|---|
| 1 | `SkillTypeToStatusName` (`:71`) | 4 ids folded into existing arms, 4 new arms (D1) |
| 2 | `IsAoeSkill` (`:99`) | 150–157 added + FR-5.3 deviation comment (D3) |
| 3 | `skillNameMap` (`:145`) | 8 `CARNIVAL_*` entries (FR-2.1) |

**`services/atlas-monsters/atlas.com/monsters`**

| # | Site | Change |
|---|---|---|
| 4 | `monster/picker.go` (new helper) | `skillSuppressingStatus` (D6) |
| 5 | `monster/picker.go:124` | bare `"SEAL"` → helper + named log (FR-6.2, FR-6.4) |
| 6 | `monster/processor.go:866` | bare `"SEAL"` → helper + named log (FR-6.3, FR-6.4, FR-7.2) |
| 7 | `monster/processor.go:949` | `+ SkillCategoryCarnivalBuf` (D2) |
| 8 | `monster/processor.go:1033` | `+ SkillCategoryCarnivalBuf` (D2) |
| 9 | `monster/processor.go:1018` | honor `testMobSkillLookup` (D8) |

**Docs**

- `services/atlas-monsters/docs/domain.md` — the `UseSkillGM` sentence at `:278` currently says
  it skips the "seal" check, which stays true; the picker description at `:283` item 1 ("or the
  monster is sealed") and item 4's picker-relevant-status list need to reflect that `SEAL_SKILL`
  now actually gates, per FR-6.5.
- `docs/research/missing-features/monsters-and-bosses.md` §8 — **this file does not exist in
  this worktree.** `ls docs/research/missing-features/` returns only
  `items-and-consumables.md`, and `git ls-files | grep monsters-and-bosses` is empty. The PRD's
  final acceptance criterion cites a path that is not in the repository. Flagged for the user
  in §10; the planning phase must either drop that criterion or name the real file.

**Explicitly not touched:** `libs/atlas-packet`, `services/atlas-channel`,
`services/atlas-data`, `services/atlas-buffs`, `services/atlas-messages`, all seed data, skill
145's missing mapping, the `StatusSeal`/`TemporaryStatTypeSeal` duplication,
`isBossAllowedStatus`, `applyAnimationDelayedEffect`.

---

## 6. Data flow

```
@mobstatus CARNIVAL_PAD                     (atlas-messages, GM-gated)
  └─ SkillNameToId("CARNIVAL_PAD") → 150    ← edit 3
     └─ USE_SKILL_FIELD command (Kafka)
        └─ handleUseSkillFieldCommand        (atlas-monsters, per monster in field)
           └─ UseSkillGM(uniqueId, 150, 1)
              ├─ mobskill lookup             ← edit 9 (test seam)
              └─ SkillCategory(150) = CARNIVAL_BUFF
                 └─ switch arm               ← edit 8
                    └─ executeStatBuff
                       ├─ SkillTypeToStatusName(150) = POWER_UP   ← edit 1
                       ├─ statuses{POWER_UP: sd.X()}, dur = ms
                       ├─ category ≠ Immunity → no opposite pre-cancel
                       ├─ category ≠ Reflect  → NewStatusEffect
                       ├─ ApplyStatusEffect → registry (evicts same-type, D4)
                       │    └─ MONSTER_STATUS event → atlas-channel → StatSet packet
                       └─ IsAoeSkill(150)=true && HasBoundingBox()=false → loop skipped ← edit 2
```

The autonomous path (`pickNextSkill` → `NEXT_SKILL_DECIDED` → `UseSkill`) is unchanged in shape;
it simply cannot reach 150–157 because no such skill declares `prop` (§1.1). The `SEAL_SKILL`
gate (edits 4–6) sits at the head of both `pickNextSkill` and `UseSkill`.

No new topic, no new event type, no new packet. `atlas-channel` renders the result through the
existing generic mob temporary-stat mask path.

---

## 7. Testability

Established package pattern, confirmed by reading existing tests: construct
`ProcessorImpl` as a struct literal with stubbed function fields, seed the real monster registry
via `r.CreateMonster`, override the package-level `testMobSkillLookup` / `testInformationLookup`
hooks with `defer`-restore, call the method, assert on `GetMonster(...).StatusEffects()` /
`HasStatusEffect(...)` (`processor_test.go:876–1135`, `:2180–2280`). Cooldown tests wrap
`miniredis`. Carnival tests follow this exactly; no new harness.

Two seams are needed and only one is new:

- `testMobSkillLookup` in `UseSkillGM` — edit 9 (D8). New.
- `mobskill.NewBuilder()` for synthetic skill data including a negative `X` and a synthetic
  bounding box — already exists and is already used this way.

`executeStatBuff` can also be called directly, which is how the existing immunity/reflect tests
assert `NewStatusEffect`-vs-reflect routing. FR-3.3's assertions use that entry point.

---

## 8. Test plan

Derived from §10 of the PRD, plus two tests §1 showed are needed.

**`libs/atlas-constants/monster` (pure, table-driven):**

1. `SkillTypeToStatusName` — all eight ids against the FR-1.1 table; plus 149 and 158 → `""`.
2. `SkillTypeToStatusName` — 100/110/150 all return `POWER_UP`; 102/112/152 all return
   `POWER_GUARD_UP`; 115/156 both return `SPEED` (FR-1.2, and it pins D1's arm folding).
3. `IsAoeSkill` — true for 150–157; unchanged for the pre-existing true set
   (110–113, 114) and for a representative false set.
4. `SkillNameToId` — all eight `CARNIVAL_*` names → correct ids; unknown name → `false`.
5. `SkillTypeNames()` — contains all eight; still sorted.
6. `SkillCategory` — all eight → `SkillCategoryCarnivalBuf` (regression pin; unchanged code).

**`services/atlas-monsters` — dispatch:**

7. `UseSkill` with 150, 154, 157 → registry carries `POWER_UP` / `ACCURACY` / `SEAL_SKILL`.
8. `UseSkillGM` with the same three → same result (requires edit 9).
9. No "unknown skill category" warning for any of the eight — assert via a captured
   `logrus` hook rather than by inspection, for both entry points.
10. FR-3.3: `executeStatBuff` with 150 while `MAGIC_ATTACK_IMMUNE` is active → the immunity
    survives (no opposite pre-cancel), and the produced effect is a plain effect, not a reflect
    effect (`ReflectKind()`/reflect fields unset).

**Values and stacking:**

11. Skill 155 level 2 with `X() = -990` → stored status value is exactly `-990` (FR-4.3).
12. Recast an active carnival buff with a different `X` and duration → exactly one effect of
    that type, carrying the new value and the new expiry (FR-4.1, D4).
13. `Duration()` of `1200000` ms → 20 minutes, not 20 000 minutes (FR-4.4).

**AoE:**

14. Carnival skill, no bounding box, two monsters in field → only the caster is buffed.
15. Carnival skill with a synthetic `lt`/`rb`, three monsters → in-box buffed, out-of-box not.

**`SEAL_SKILL`:**

16. `pickNextSkill` on a monster carrying `SEAL_SKILL` → sentinel `Decision{}`.
17. `pickNextSkill` on a monster carrying `SEAL` → sentinel (regression, D7).
18. `UseSkill` on a monster carrying `SEAL_SKILL` → no effect applied, and the debug line names
    `SEAL_SKILL` (FR-7.2) — assert the log text, since "distinct from the SEAL rejection" is the
    requirement.
19. `UseBasicAttack` on a monster carrying `SEAL_SKILL` → still succeeds (FR-6.6).
20. End-to-end: cast 157 on monster A, then `UseSkill` any skill on A → rejected.

**From §1 (not in the PRD):**

21. **Hsalf regression.** A monster whose skill list is `{id: 156, level: 1}` with the real WZ
    skill data (no `prop`) → `pickNextSkill` returns the sentinel. This pins the fact that
    §1.1's "no autonomous caster" claim is enforced by the `prop <= 0` gate and not by accident.
22. Skill 145 (`PHYSICAL_AND_MAGIC_COUNTER`) still returns `""` from `SkillTypeToStatusName`
    (pins D1's deliberate non-change, so the omission is visibly intentional to the next reader).

---

## 9. Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Widening `IsAoeSkill` changes behavior somewhere unexamined | Low | Single caller repo-wide, verified by grep; inert on all current data (no `lt`/`rb`) |
| Broadening the seal gate suppresses skills that used to fire | Low | Gate is strictly additive — `SEAL` continues to block exactly what it blocked (D7); test 17 pins it |
| Folding carnival ids into shared arms (D1) breaks an existing mapping | Low | Test 2 asserts old and new ids together |
| `-990` gets clamped by a well-meaning later change | Medium | Test 11 asserts the exact value; FR-4.3 rationale is in the design and should be echoed at the site |
| Edit 9 changes production behavior | None | Hook is nil in production; the added branch is unreachable outside tests, matching `UseSkill` |
| A future picker change makes `prop`-less skills selectable, giving Hsalf a permanent buff | Medium | Test 21 |

---

## 10. Open items for the planning phase

1. **`docs/research/missing-features/monsters-and-bosses.md` does not exist** in this worktree
   (§5). The PRD's last acceptance criterion targets it. The plan must either drop that
   criterion, create the document, or name the file the user actually meant. **Needs a user
   decision — do not guess a path.**
2. **OQ-3 (client rendering of `ACCURACY`/`AVOIDABILITY`) remains open and is not blocking.**
   Both are registered in the mob temporary-stat bitfield
   (`libs/atlas-packet/model/monster.go:97–98`) and encode through the generic mask, so the
   server-side contract is deliverable and testable regardless. Whether the client draws an
   indicator is an in-client observation nobody can make from the repo; it should be checked
   during manual verification, not blocked on.
3. OQ-1 and OQ-2 are **closed** by §1 and need no further work.
4. OQ-4 is **closed** by D7: keep both gates.

---

## Appendix — commands run at design time

```
grep -l 'name="skill" value="15[0-7]"' <wz-root>/Mob.wz/*.xml          → 9400593.img.xml (1 file)
sed -n '1,40p'  <wz-root>/Mob.wz/9400593.img.xml                       → skill 156 level 1, boss=1
grep -A3 '"9400593"' <wz-root>/String.wz/Mob.img.xml                   → name "Hsalf"
python3 (scan MobSkill.img.xml for ids 150-157)                        → field inventory, §1.4
sed -n '190,285p' <cosmic-root>/src/main/java/server/life/MobSkill.java → FR-1.1 mapping
sed -n '1450,1465p' <cosmic-root>/src/main/java/server/life/Monster.java → FR-6.1 gate
grep -rn --include='*.go' "IsAoeSkill" services libs                   → 1 caller
grep -rn --include='*.go' -e 'SealSkill' -e 'SEAL_SKILL' services libs → no gate consumes it
grep -rn "testMobSkillLookup" --include='*.go' .                       → UseSkill only, not UseSkillGM
grep -n -A25 "func (b \*Builder) AddStatusEffect" monster/builder.go   → same-type eviction (D4)
ls docs/research/missing-features/                                     → items-and-consumables.md only
```
