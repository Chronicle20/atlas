# Priest Doom (Task 047) — Wrong-Channel Postmortem

Created: 2026-05-03 (post-PR #377)
Author: implementer
Trigger: live test on the deployed branch failed — Doom did not transform mobs and Magic Rock was not consumed.

---

## TL;DR

The PRD/design/plan for task-047 assumed Doom flows through atlas-channel's
**magic-attack** packet handler (`character_attack_magic.go` → `processAttack`
→ `processDamageInfoEntry`'s empty-damage branch). It does not. The v83
client sends Doom over the **buff (SPECIAL_MOVE) opcode** only; the
magic-attack handler is never invoked for Doom. Cosmic confirms this
(`SpecialMoveHandler.java:138` → `StatEffect.applyTo` → `applyMonsterBuff`
at `StatEffect.java:1180-1204`, server-authoritative bounding-box mob
selection). All channel-side Doom production code on this branch therefore
sits on dead paths for the live skill.

The empirical confirmation: with a breakpoint on
`CharacterMagicAttackHandle`, casting Doom does not hit it. Loki shows
`Character [N] using skill [2311005]` (the buff dispatcher) and nothing
else.

---

## How the wrong assumption entered the design

PRD §4.1 stated: "The magic-attack packet handler ... routes a packet whose
`SkillId() == 2311005` through `processAttack` (the existing common path).
... the empty-damage monster-status apply branch in
`character_attack_common.go` already covers Doom's behaviour."

This was inferred from atlas-channel's existing code shape (the empty-damage
branch of `processAttack` calls `mp.ApplyStatus` when `len(damages) == 0
&& len(se.MonsterStatus()) > 0`, which superficially fits Doom). It was
**not** verified against either Cosmic source or live packet capture before
the design was written. The reconnaissance pass mentioned in the PRD
("plumbing for this skill is already largely assembled") read like a
positive signal but was actually pattern-matching, not evidence.

The plan's Task 0 pre-flight verified that the constants and entry points
existed; it did not verify that the v83 client actually *uses* those entry
points for Doom. That gap is the root of the failure.

---

## What the v83 client actually does

For Doom, the client sends one packet over the **SPECIAL_MOVE** opcode
(atlas-channel's `CharacterUseSkillHandle`). The packet body carries:

- `updateTime` (uint32)
- `skillId` (uint32) — `2311005`
- `skillLevel` (byte)
- trailing 5 bytes: `castX` (int16), `castY` (int16), 1 byte (delay/direction)

Notably absent: a list of affected monster ids. Cosmic's
`SpecialMoveHandler.handlePacket` reads only the cast position; the server
then does its own bounding-box mob selection in `StatEffect.applyMonsterBuff`
using the caster's current position + the skill's `lt`/`rb` rectangle, and
applies the status to up to `mobCount` of the monsters in that rectangle.

This is **server-authoritative** target selection. atlas-channel's existing
`applyToMobs` function (`skill/handler/common.go:56`) trusts the client's
mob list (`info.AffectedMobIds()`); for Doom that list is empty (the v83
client doesn't send it, AND `isMobAffectingBuff(PriestDoomId)` is false in
`libs/atlas-packet/model/skill_usage_info.go` so the decoder wouldn't read
it even if the bytes were there), so `applyToMobs` early-returns and the
DOOM apply never fires.

`itemCon` (Magic Rock) consumption was placed in
`processAttack`'s HP/MP cost gate. Since `processAttack` is never called
for Doom, the Magic Rock is never burned either.

---

## Branch state vs in-game state

What's in the merged branch (PR #377) and what it actually does in-game:

| Commit | Change | In-game effect for Doom |
|---|---|---|
| 78f27eb20 | atlas-data Doom effect-mapping reader test | n/a — test only, but pins a contract that is still correct |
| 4c79dc3d9 | atlas-monsters `information.ModelBuilder.SetBoss/SetResistances` | n/a — test affordance, still useful |
| 96d2e213f | atlas-monsters `testInformationLookup` routes through `ApplyStatusEffect` | n/a — test affordance, still correct |
| 163251790 | atlas-monsters DOOM short-circuit + 3 Doom apply tests | **Engages once an upstream emits `ApplyStatus({DOOM:1})`. Today nothing does, so unobservable.** |
| 03c84901c | atlas-channel `processDamageInfoEntry` helper extraction | Pure refactor on the magic-attack path. Doom never reaches it. |
| e05a1983a | atlas-channel Doom-gated magic-reflect probe in empty-damage branch | **Dead for Doom — wrong handler.** |
| ecb2535a8 | atlas-channel 4 helper tests for Doom cast / reflect / spread | Tests pass but exercise the wrong-path helper. |
| 746c24714 | atlas-channel `Doom: caster=…` Debugf in `monster.Processor.ApplyStatus` | Will fire whenever any code path calls `ApplyStatus` with `{DOOM:1}`. Today nothing does. |
| 4a3312d6d | atlas-channel `itemCon` consume inside `processAttack`'s HP/MP cost gate | **Dead for Doom and for every other `itemCon` skill — `itemCon` is a buff-path concern; magic-attack-path skills don't use `itemCon`.** |
| 98b38112b | DOM-21 — route `"DOOM"` / `2311005` literals through `libs/atlas-constants` | Stylistic; correct regardless of path. |
| 0599ef965 | Delegate `findItemSlotInInventory` to `compartment.FindFirstByItemId` | Dead with the parent change. |
| 9f1b14a00 | Inline `findItemSlotInInventory` into `processAttack`; drop helper + tests | Dead with the parent change. |

---

## What stays useful (no action needed)

These pieces are correct in their own right and engage on the corrected
path. **No revert.**

- `services/atlas-data/atlas.com/data/skill/reader_test.go`:
  `TestReader_PriestDoom_MapsDoomStatus` (78f27eb20). Pins
  `MonsterStatus["DOOM"] = 1` and `Duration > 0` for skill 2311005. The
  contract is independent of which channel-side handler consumes it.
- `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go`:
  `SetBoss` / `SetResistances` on `ModelBuilder` (4c79dc3d9).
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go`:
  - `testInformationLookup` extension to `ApplyStatusEffect` (96d2e213f).
  - Explicit DOOM short-circuit at the top of `isElementallyImmune` (163251790,
    98b38112b). Engages once the new buff-path handler emits `ApplyStatus`.
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`:
  three Doom tests (`BypassesElementalImmunity`, `RejectedOnBoss`,
  `ReapplyReplacesExisting`). They test `ApplyStatusEffect` directly and
  do not assume any particular upstream caller.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`:
  `processDamageInfoEntry` extraction (03c84901c). Pure refactor; useful
  seam regardless of whether Doom reaches it. **Keep.**
- `services/atlas-channel/atlas.com/channel/monster/processor.go`:
  `Doom: caster=…` Debugf in `Processor.ApplyStatus` (746c24714, 98b38112b).
  Fires on the new path too. **Keep.**

---

## What gets reverted (wrong path)

These changes target `processAttack`/`processDamageInfoEntry` for Doom,
which is unreachable. They are reverted in this revision.

- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`:
  - The Doom-gated magic-reflect probe inserted into
    `processDamageInfoEntry`'s empty-damage branch (e05a1983a).
  - The `if itemId := se.ItemConsume(); itemId > 0 { ... }` block inside
    `processAttack`'s HP/MP cost gate (4a3312d6d, 9f1b14a00). The
    *replacement* — generic `itemCon` consumption — is moved to
    `skill/handler/common.go:UseSkill`'s cost block, which is the correct
    home for buff-path skills (and is the only place every `itemCon`
    skill flows through today).
  - Imports added solely for the above (`atlas-channel/consumable`, the
    `inventoryconst`/`itemconst`/`charcon`/`slot` aliases when no other
    code in this file uses them).
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go`:
  - The four `TestProcessDamageInfoEntry_Doom_*` tests + supporting
    `damageEntryFakes`, `applyStatusCall`, `newDoomEffect`,
    `newDoomAttackInfo` helpers (ecb2535a8). They test the wrong-path
    helper. Replacement tests for the new Doom handler land alongside
    the new code under `skill/handler/doom/`.
  - `skillconst` import line if no remaining test uses it.
- `services/atlas-channel/atlas.com/channel/monster/processor.go`:
  no revert (the Debugf stays — it works on the corrected path).

---

## What gets added (corrected path)

See the revised `design.md` and `plan.md` for the architecture and the
task list. Headlines:

1. **`libs/atlas-packet`**: nothing. The buff packet for Doom carries no
   mob list; we do not need cast position decoding either (Cosmic uses
   the caster's current position for monster-buff bounding boxes).
2. **`atlas-monsters`**: a new "monsters in field within rectangle" REST
   query (and underlying processor query). The rectangle is given by
   `(x1, y1, x2, y2)`; results are limited by an optional `limit`
   parameter (used to honor the skill's `mobCount`).
3. **`atlas-channel/monster`**: a thin client wrapper for the new query.
4. **`atlas-channel/skill/handler/doom/`**: a new per-skill handler
   registered for `PriestDoomId`. Mirrors Cosmic's `applyMonsterBuff`:
   compute the bounding box from caster position + facing + `e.LT()` /
   `e.RB()`, query monsters in that rectangle (capped to `e.MobCount()`),
   apply DOOM via `mp.ApplyStatus` to each. Per-mob magic-reflect probe
   skips reflect targets without emitting reflect damage (Doom does no
   damage, mirroring the previous design intent).
5. **`atlas-channel/skill/handler/common.go`**: extend the `UseSkill`
   cost block to charge `e.ItemConsume()` (Magic Rock for Doom; summon
   items for summon skills; etc.). This is the generic path the user
   asked for, now placed where `itemCon` skills actually flow.

---

## Lessons / process notes

1. **Verify the cast packet path before designing around it.** A
   five-minute breakpoint or Loki query at PRD time would have caught
   this. The plan's "Task 0 pre-flight" was pure code-existence
   verification; it should have included a "what packet does the v83
   client actually send for this skill?" step.
2. **Cross-reference the canonical reference implementation (Cosmic)
   for non-trivial dispatcher questions.** The PRD references
   `StatEffect.java:1531` (where DOOM is in `isMonsterBuff`) but did
   not trace the call chain that consumes that flag back to the
   handler. `isMonsterBuff` is consumed in `applyMonsterBuff`, which
   is invoked from `applyTo`, which is invoked from
   `SpecialMoveHandler` — the buff opcode, not the magic-attack
   opcode. Tracing one more hop would have surfaced the wrong-channel
   assumption.
3. **`itemCon` belongs in the buff-path cost block, not the
   magic-attack cost block.** Every `itemCon` skill in v83 (Doom,
   Mystic Door, summons, mists, …) is dispatched via SPECIAL_MOVE.
   No magic-attack-packet skill consumes inventory items. Plumbing
   `itemCon` into `processAttack`'s cost gate was wrong on day one.
4. **Test affordances built for the wrong path are not "wasted" code,
   but they shouldn't be retained on a feature branch under the
   pretense of covering the feature.** The four
   `TestProcessDamageInfoEntry_Doom_*` tests pass and cover the
   helper they target; they just don't cover Doom. Removing them is
   honesty, not cleanup-for-its-own-sake.
