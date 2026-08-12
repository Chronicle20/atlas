# Monster Magnet — Design

Task: task-215-monster-magnet
PRD: [`prd.md`](prd.md)
Status: Draft
Created: 2026-08-12

---

## 0. Summary

Monster Magnet is delivered on the **same opcode as `USE_SKILL` /
`CharacterUseSkillHandle`**, but its body is written by a *different client
function* (`CUserLocal::TryDoingMonsterMagnet`) than every other skill on that
opcode (`CUserLocal::SendSkillUseRequest`). The magnet body is **not** a variant
of the mob-affecting-buff block — it is a structurally distinct arm that begins
diverging immediately after `skillLevel`.

All ten provisioned client versions were decompiled for this design (§2). The
payload has **two shapes**, not one, and the split is at the v48/v61 boundary —
not at the v72 boundary the PRD anticipated from the anti-repeat gate.

The WZ data was also read (§3): Monster Magnet carries **no `lt`/`rb`**, so the
existing `applyToMobs` rectangle path can never fire for it. It carries `range`,
`mobCount` and `prop` instead, and the client's target region is a **trapezoid**
derived from `range` — which atlas-channel does not currently decode off
atlas-data at all.

Four PRD requirements need adjustment as a result (§9). None of them are
blocking; all are resolved below.

---

## 1. What already exists (no work required)

| Piece | Where | State |
|---|---|---|
| Three magnet skill identities + per-version wire maps | `libs/atlas-constants/skill/constants.go:2950,2978,3007`; `version_gms_*_gen.go` | Wire id is `1121001`/`1221001`/`1321001` on **every** provisioned version. Not in task-187's divergence set. |
| Keydown prepare/keyup relay | `socket/handler/character_skill_prepare.go`, `character_buff_cancel.go`; `skill/identity.go:137-141` | Already covers all three identities. Regression check only. |
| `CATCH_MONSTER` codec + writer + template routes | `libs/atlas-packet/monster/clientbound/catch_monster.go`; `socket/writer/catch_monster.go`; all 10 non-v12 templates | Routed everywhere, **senderless** — left so by task-212 for exactly this task. |
| SKILL_USE effect `left` (direction) byte | `libs/atlas-packet/character/clientbound/effect_skill_use.go:99,121`; `character/effect_body.go:63,75` | The codec **already** gates a trailing `monsterMagnetLeft` bool on the three magnet ids. Both call sites hard-code `false`. |
| Per-cast self + foreign SKILL_USE broadcast | `socket/handler/character_skill_use.go:176,178` | Already fires once per cast for every skill. FR-6.1/FR-6.3 are already satisfied; only the `left` argument is missing. |
| MP cost | `skill/handler/common.go` `UseSkill` | `e.MPConsume()` (WZ `mpCon` 10→21) is charged generically before the per-skill dispatcher. The magnet handler must **not** charge it again. |
| `EnableActions` unlock | `character_skill_use.go` tail | Already emitted. Magnet warps nothing, so the unlock contract is unchanged. |

---

## 2. Wire layout — derived per version

Method: `CUserLocal::TryDoingMonsterMagnet` was located in every IDB and its
packet-build tail read instruction-by-instruction. Four IDBs had the function
unnamed; it was identified by structure and **renamed to
`CUserLocal__TryDoingMonsterMagnet` in the IDB** (gms_72, gms_79, gms_84,
gms_92) so the next reader does not repeat the search.

| Version | Function | Opcode (from that version's seed template) | Shape |
|---|---|---|---|
| gms_48 | `0x6AD842` | `0x46` (70) | **legacy** |
| gms_61 | `0x7B9684` | `0x53` (83) | modern |
| gms_72 | `0x876605` | `0x5A` | modern |
| gms_79 | `0x8C3117` | `0x59` | modern |
| gms_83 | `0x96C215` | `0x5B` | modern |
| gms_84 | `0x9ABDB7` | `0x5B` | modern |
| gms_87 | `0x9F086F` | `0x5E` | modern |
| gms_92 | `0x91F2A0` | `0x66` | modern |
| gms_95 | `0x940570` | `0x67` | modern |
| jms_185 | `0xA3C61C` | `0x56` | modern |

The opcode column matches the `CharacterUseSkillHandle` binding in each seed
template, and the `COutPacket` constructor argument in each function agrees
(v48 `push 46h` @`0x6ADABC`; v83 `COutPacket(0x5B)`; v61 `COutPacket(83)`).
**No new template route is needed** — the magnet is an arm of an
already-routed handler.

### 2.1 Modern shape (gms_61 … gms_95, jms_185)

```
updateTime   : uint32
skillId      : uint32
skillLevel   : byte
grabCount    : uint32            <-- FOUR bytes, not one
repeat grabCount times:
    objectId : uint32            <-- CMob object id (0 when the entry was released)
    grabbed  : byte              <-- Encode1(result == 3); a bool, not an enum
direction    : byte              <-- (CUserLocal.stance & 1); 1 = facing left
```

There is **no trailing `delay` short.** Encode-call addresses:
gms_61 `0x7B9684` body; gms_72 `0x876A2B/38/43/4E` + loop `0x876A86,0x876A95` +
tail `0x876AB2`; gms_79 `0x8C3540/4D/58/63` + loop `0x8C359B,0x8C35AA` + tail
`0x8C35C7`; gms_83 `0x96C215` body; gms_84 `0x9AC1F3/200/20B/216` + loop
`0x9AC24E,0x9AC25D` + tail `0x9AC27D`; gms_87 `0x9F0CAB/CB8/CC3/CCE` + loop
`0x9F0D06,0x9F0D15` + tail `0x9F0D35`; gms_92 `0x91FA54/60/71/7F` + loop
`0x91FABD,0x91FAE1` + tail `0x91FAFB`; gms_95 `0x940D25/31/42/50` + loop
`0x940D8D,0x940DB1` + tail `0x940DCB`; jms_185 `0xA3CC52/5C/67/72` + loop
`0xA3CCAA,0xA3CCC6` + tail `0xA3CCE3`.

Notes that matter for the decoder and the validator:

- **`grabbed` is boolean.** gms_83 `0x96C215`: `COutPacket::Encode1(v65, *v40 == 3)`.
  The `3` is a purely client-side sentinel written by the client's own prop roll
  (`CRand32::Random() % 100 < levelData.prop`). Since WZ `prop` is **100 at every
  level** (§3), the roll always succeeds; a `0` on the wire therefore means the
  mob was position-fixed or otherwise excluded, not "unlucky".
- **`objectId` can legitimately be `0`.** The encode loop reads the id out of the
  `ZRef<CMob>` slot, and slots whose `CanGoThrough`/`CanWalkThrough` probe failed
  were released earlier in the same function. `grabCount` counts the *candidate*
  array, not the survivors. The decoder must tolerate a zero id; the validator
  drops it.
- **`direction` is `stance & 1`**, the same facing bit the existing
  `applyToMobs` derives via `(c.Stance() & 1) == 1`.

### 2.2 Legacy shape (gms_48 only)

```
updateTime   : uint32            (0x6ADAD3)
skillId      : uint32            (0x6ADAE0)
skillLevel   : byte              (0x6ADAEB)
entryCount   : byte              (0x6ADB02)   <-- ONE byte
repeat entryCount times:
    objectId : uint32            (0x6ADB1B)   <-- NO per-entry result byte
delay        : uint16            (0x6ADB29)   <-- the shared action delay
```

There is **no direction byte** and **no per-entry grab result** on v48. Two
further v48-only facts, both verified in disassembly:

- **`entry[0]` is the CASTER, not a monster.** At `0x6AD977`–`0x6AD987` the
  function does `mov edi,[esi+654h]` (`esi` = `CUserLocal`) and
  `ZArray<ulong>::InsertBefore(-1)` *before* the mob loop begins. The loop then
  appends `[mob+654h]` per surviving mob (`0x6ADA89`–`0x6ADA99`) — the same
  object-id offset. So `entryCount == 1 + mobCount`, and index 0 must be
  discarded.
- The v48 client performs **no grab roll and no `AddDamageInfo`** — the whole
  per-mob hit-info block that gms_61+ runs is absent. Every surviving entry is
  an unconditional grab.

The legacy shape is byte-identical in *layout* to the generic mob-affecting-buff
block (`count:1, ids:4×N, delay:2`) — but it is **not** that block, because of
the leading caster entry. Routing v48 through `isMobAffectingBuff` would work
byte-wise and be wrong semantically.

### 2.3 Version gate

The split is v48 vs everything else. Per the file's established idiom the gate
is a region test plus `MajorAtLeast`, never a raw `>`:

```go
legacyMagnetLayout := t.IsRegion("GMS") && !t.MajorAtLeast(61)
```

JMS takes the modern branch (jms_185 `0xA3C61C` verified). This is a *narrower*
gate than the `isAntiRepeatBuffSkill` gate two branches above it, and the
comment must say so explicitly so a future reader does not "harmonise" them.

---

## 3. WZ data — derived, not assumed

Source: local extracted dump `tmp/<tenant>/GMS/83.1/Skill.wz/{112,122,132}.img.xml`.

All three skills are identical, 30 levels each:

| | level 1 | level 30 |
|---|---|---|
| `mpCon` | 10 | 21 |
| `prop` | 100 | 100 |
| `range` | 200 | 450 |
| `mobCount` | 3 | 7 |

**There is no `lt` and no `rb` node anywhere under any of the three skills.**

Consequences:

1. PRD FR-2.1's "membership in the skill's effect bounding box" cannot be
   evaluated with `calculateBoundingBox` — it would read `lt=rb=(0,0)`,
   `hasEffectBbox` would return false, and FR-2.4's fallback ("accept the client
   list subject to the cap only") would be the permanent live path. That is a
   validation no-op, which defeats FR-2's entire purpose.
2. `mobCount` **is** the correct cap and matches the client exactly: the client
   caps its own candidate array at `levelData+304` scaled by charge time
   (gms_83 `0x96C215`), and hard-rejects anything above 15.
3. `range` defines the real target region — see §4.2.

`range` is read and served by atlas-data
(`services/atlas-data/atlas.com/data/skill/reader.go:282`,
`effect/rest.go:78 "range"`) but **atlas-channel's `effect.RestModel` does not
decode it**. Adding the field is part of this task.

### 3.1 The client's target region is a trapezoid

`CUserLocal::TryDoingMonsterMagnet` selects candidates through
`sub_678AEC` → `CMobPool::CheckMobInTrapezoid` (gms_83 `0x679084`), called with
`(x0 = casterX, xStart = casterX ± 50, xEnd = casterX ± range, y = casterY - 28,
slope = 4)`; the sign follows `stance & 1`.

`CheckMobInTrapezoid` steps `x` from `xStart` to `xEnd` in 20px slices and, at
each slice, intersects the mob's **body rect** against
`[x, y − |x − x0|/4] … [x + 20, y + |x − x0|/4]`. So the region is a wedge that
opens as it gets further from the caster: half-height 12px at `|dx| = 50`,
112px at `|dx| = 450`.

---

## 4. Architecture

Five units, each independently testable.

```
 client ──USE_SKILL(magnet arm)──▶ SkillUsageInfo.Decode        [libs/atlas-packet]
                                        │ grabs[], direction
                                        ▼
                            character_skill_use.go              [atlas-channel]
                                        │  UseSkill(...)  (generic cost/cooldown)
                                        ▼
                       skill/handler/monstermagnet.Apply         [atlas-channel]
                            │              │              │
              validate (§4.2)│    CATCH_MONSTER│      2 × monster cmd
                            │      to others  │              │
                            ▼                 ▼              ▼
                        (drop/log)       field sessions   COMMAND_TOPIC_MONSTER
                                                                │
                                            ┌───────────────────┴───────────────┐
                                            ▼                                   ▼
                                    ClearAggro (§4.4)                  ForceControl (§4.5)
                                                                   └─▶ START_CONTROL event
```

### 4.1 Decode (`libs/atlas-packet/model/skill_usage_info.go`)

New immutable value type in the same package:

```go
type MagnetGrab struct {           // FR-1.6
    objectId uint32
    grabbed  bool
}
func (m MagnetGrab) ObjectId() uint32 { ... }
func (m MagnetGrab) Grabbed() bool    { ... }
```

New `SkillUsageInfo` fields `magnetGrabs []MagnetGrab`, `direction bool`, with
getters `MagnetGrabs()` / `Direction()` and builder setters
`SetMagnetGrabs` / `SetDirection` (FR-1.5).

The branch is inserted **immediately after `skillLevel` is read and before the
`isAntiRepeatBuffSkill` gate**, and `return`s — the magnet body shares no
suffix with the other arms:

```go
m.skillLevel = r.ReadByte()
if isMonsterMagnet(ctx, m.skillId) {
    m.decodeMagnet(r, legacyMagnetLayout(t))
    return
}
// ... existing anti-repeat / javelin / party / mob branches unchanged
```

`isMonsterMagnet` resolves through
`constants.For(region, major, minor).Skill.Resolve` and compares Identities
(FR-1.1), matching how `common.go` and `character_skill_prepare.go` already do
it. The magnet ids are **not** in task-187's divergence CSV, so the guard would
not force this — it is done for consistency and because this file's raw
`skill.Is` lists are the exact shape task-187 exists to retire.

Legacy branch drops index 0 (§2.2) and marks every remaining entry
`grabbed = true`; the `delay` short lands in the existing `m.delay`.
Modern branch reads the uint32 count and the per-entry bool, then `direction`.

**Backward-compatibility argument (NFR).** The branch is reachable only for the
three magnet Identities, it is the first branch, and none of the three appear in
`isMobAffectingBuff`, `isPartyBuff` or `isAntiRepeatBuffSkill` today (FR-1.3).
No other skill's decode changes by a single byte on any version.

#### Alternative considered — decode the magnet as a fourth "block" alongside the existing ones

Rejected. The existing branches are *additive suffixes* of one common prefix;
the magnet is a *replacement* body. Modelling it as another `if` in the same
falling-through chain invites a future edit that lets a magnet cast also run the
party or mob block, which would silently misalign the reader. An early `return`
makes the mutual exclusion structural rather than list-maintained.

### 4.2 Validation (`skill/handler/` + new `monstermagnet` subpackage)

The PRD (FR-2.6) asks for the `applyToMobs` checks to be factored into a shared
helper. Only part of that is right, because the geometry differs:

| Step | Shared with `applyToMobs`? |
|---|---|
| `mobCount` cap → reject entire cast | **yes** — extract `enforceMobCap` |
| client∩server id intersection + anomaly split | **yes** — `intersectMobIds` already exists and is reusable as-is |
| anomaly / summary log field vocabulary | **yes** — reuse `monster_buff_anomaly_*` names |
| rectangle from `lt`/`rb` | **no** — magnet has none (§3) |
| reflect skip, prop roll, status apply/cancel | **no** — magnet applies no status (FR-2.6) |

So the extraction is: pull the cap check and the anomaly-logging shape out of
`applyToMobs` into small package-level helpers in `skill/handler`, and give the
magnet its own geometry function.

**Geometry (new `magnetRegion` in `skill/handler/mob_select.go`).** The server
computes the **axis-aligned bounding box of the client's trapezoid**, not the
trapezoid itself:

```
s   = facingLeft ? -1 : +1
x1..x2 = sorted(casterX + s*50 - margin, casterX + s*range + margin)
yc     = casterY - 28
y1..y2 = yc - range/4 - margin, yc + range/4 + margin
```

Rationale for the AABB rather than the exact trapezoid: the client tests the
mob's **body rect**, and atlas-monsters only exposes the mob's anchor point via
`GetInMapRect`. Reproducing the trapezoid against a point would reject
legitimate grabs of tall mobs near the wedge edge. The AABB is a strict superset
of the client's region, which is the correct posture for an anti-cheat gate: it
still rejects a fabricated target on the other side of the map or beyond
`range`, and it never fights sub-pixel geometry. `margin` is a single named
constant with a comment explaining it stands in for the unmodelled body rect.

Exactly **one** `GetInMapRect` call per cast (NFR performance), limit = `mobCount`.

Order of operations, matching the PRD's stated policies:

1. `grabbed == false` entries and `objectId == 0` entries are dropped before
   anything else (FR-2.5, §2.1).
2. If the surviving count exceeds `e.MobCount()` → reject the **whole** cast,
   `warn` with `event=monster_magnet_anomaly_over_cap` (FR-2.2).
3. Load caster; on error → drop whole cast, `error` (FR-2.7).
4. Compute region, one rect query; on error → drop whole cast, `error` (FR-2.7).
5. Intersect; ids the server did not return are dropped **individually** and
   logged `warn event=monster_magnet_anomaly_out_of_rect` with claimed vs server
   ids and the computed rect (FR-2.3).
6. Per surviving id: broadcast + two commands.
7. One `debug` summary per cast.

If `range` is absent or zero for a tenant's data (defensive — it is present in
all read WZ), step 4 is skipped and the cast proceeds cap-only with a `debug`
log. This is FR-2.4's fallback, relocated from `lt`/`rb` to `range`.

### 4.3 Grab-effect broadcast

For each validated grab, announce `CatchMonsterWriter` with
`writer.CatchMonsterBody(objectId, result=1, success=1)`.

`result` is the value the client passes to `ShowCatchEffect`; on the magnet path
the local client computes it as `(grabResult == 3)` — verified in
`CMob::OnHit` gms_83 `0x668B83`: `cmp eax,111AE9h` (`0x668DB7`),
`cmp eax,12A189h` (`0x668DC3`), `cmp eax,142829h` (`0x668DCA`) all jump to
`0x668E14`, which does `setz al` on `(arg == 3)` and calls
`ShowCatchEffect` at `0x668E22`. So the wire `result` for a successful grab is
`1`. `success` is the gms_95-only second byte the codec already gates
(`catch_monster.go:64`); `1` matches. FR-3.4 is therefore "map the boolean
through", not "map an enum through".

**Recipients: other sessions in the field, not all sessions.** This is a
deliberate deviation from FR-3.1. The caster's own client already renders the
effect locally — `TryDoingMonsterMagnet` calls `CMob::AddDamageInfo` per grabbed
mob, which drives `CMob::OnHit` → `ShowCatchEffect` on that client only. Sending
`CATCH_MONSTER` to the caster too would play the animation twice on their
screen, which is precisely the double-render task-212 removed from the
catch-item path. Remote clients never run `AddDamageInfo` for the magnet, so
they *do* need the packet. Implementation: `_map.Processor.ForOtherSessionsInMap`.

`CATCH_MONSTER_WITH_ITEM` is never sent from this path (FR-3.3) — nothing in the
magnet handler references it.

### 4.4 Clear-aggro command

Contract addition, mirrored in **both** module copies
(`services/atlas-channel/.../kafka/message/monster/kafka.go` and
`services/atlas-monsters/.../kafka/consumer/monster/kafka.go`):

```
CommandTypeClearAggro = "CLEAR_AGGRO"
type ClearAggroCommandBody struct{}        // deliberately empty (FR-4.3)
```

An empty body is also the safe choice for this topic: every handler
unmarshals every message into its own body type, so a new field name whose type
disagrees with a sibling logs a spurious error per message
(see the `killCommandBody` / `catchCommandBody` comments). Keyed by
`producer.CreateKey(int(monsterId))` like every other monster command.

atlas-monsters side:

- `Registry.ClearDamageEntries(t, uniqueId) (ClearSummary, error)` — one
  `atomicUpdate` that sets `DamageEntries = nil` and, if `ControllerHasAggro`
  was true, flips it false and reports `AggroFlippedOff` plus the controller id.
  This mirrors `DecayDamageEntries` exactly (`registry.go:707`), which is what
  makes FR-4.4 hold: the wipe converges to the same state the decay sweep would
  have reached, so the sweep's next tick sees an empty list and does nothing,
  and the picker's `ControllerHasAggro` gate behaves identically.
- `Processor.ClearAggro(uniqueId) error` — emits `AGGRO_CHANGED` **only** when
  the flag actually flipped (idempotent, FR-4.5); monster-not-found →
  `Debugf` + return nil (FR-4.6).

#### Alternative considered — reuse `DecayDamageEntries` with a "force" flag

Rejected. Decay's contract is "age idle entries"; a boolean that turns it into
"delete everything" makes the sweep's hot path carry a branch it never takes and
makes both behaviours harder to test. Two small methods over one shared
`atomicUpdate` idiom is cheaper.

### 4.5 Force-control command

```
CommandTypeForceControl = "FORCE_CONTROL"
type ForceControlCommandBody struct {
    CharacterId uint32 `json:"characterId"`
}
```

`characterId: uint32` already appears with that exact name and type in
`damageCommandBody`, `killCommandBody` and `catchCommandBody`, so it introduces
no unmarshal collision on the shared topic.

atlas-monsters side — `Processor.ForceControl(uniqueId, characterId) error`:

1. `GetById`; not found → `Debugf` + nil (FR-5.5).
2. Already the controller → return nil without emitting (FR-5.4).
3. Character not present in the monster's field (`inFieldFn`) → `Debugf` + nil
   (FR-5.5).
4. Character is GM-hidden (`hiddenFn`) → `Debugf` + nil. Not in the PRD, but
   required for consistency: `RelinquishControlOnHide` actively strips control
   from hidden characters, so granting it here would produce a flap.
5. Otherwise take the existing control path.

**Setting the aggro flag (FR-5.2) without bypassing `StartControl` (FR-5.3).**
`Registry.ControlMonster` calls `Model.Control`, which sets only
`controlCharacterId`; after §4.4's wipe, `controllerHasAggro` is false, so a
plain `StartControl` would emit `START_CONTROL` with the flag clear and the
channel would write `StartControlMonsterBody(m, false)`.

The chosen shape keeps one code path:

- `Model.ControlWithAggro(characterId)` and
  `Registry.ControlMonsterWithAggro(t, uniqueId, characterId)` set both fields
  in the same `atomicUpdate`.
- `ProcessorImpl.startControl(uniqueId, controllerId, forceAggro bool)` holds
  the existing stop-then-start / `START_CONTROL` emit /
  `RepickReasonControlChange` sequence verbatim (`processor.go:387-418`).
- `StartControl` = `startControl(..., false)`; `ForceControl` =
  `startControl(..., true)` behind the guards above.

No controller state is written directly, and the existing repick semantics are
untouched. Note the consequence: with the flag set, `startControl`'s
`ControllerHasAggro` gate *will* run `RepickAndEmit(RepickReasonControlChange)`
for each grabbed mob. That is correct — the mobs were just aggroed onto the
caster — and is bounded by `mobCount` (≤7), unlike the map-entry storm the gate
was added to prevent.

**Ordering.** Both commands are keyed by `monsterId`, so `CLEAR_AGGRO` then
`FORCE_CONTROL` for the same monster land on the same partition in order. The
channel emits them in that order per monster. Emitting force-control first would
have the wipe immediately clear the flag it just set.

**Orthogonality (FR-4.3 / FR-5.6).** Neither body mentions skills, magnets, or
each other; either is usable alone by any future caller.

### 4.6 Skill-effect direction (FR-6)

The broadcast already happens for every cast. Only the `left` bool is missing,
and the codec already models it.

Following the existing `AnnounceBerserkEffect` precedent (`effects.go:44-70` —
same problem, same shape, for the Berserk flag), add:

```go
func AnnounceDirectedSkillUse(...)(skillId, characterLevel, skillLevel, left bool) ...
func AnnounceForeignDirectedSkillUse(...)(characterId, skillId, characterLevel, skillLevel, left bool) ...
```

`character_skill_use.go` calls the directed variants with `sui.Direction()`.
Existing `AnnounceSkillUse` / `AnnounceForeignSkillUse` keep their signatures and
delegate with `left = false`, so the four other call sites
(`heal`, `healdispel`, `resurrection`, the monster consumer) are untouched.
Because `effect_body.go` derives `isMonsterMagnet` from the skill id, passing
`left` for a non-magnet skill encodes nothing.

**v48 caveat.** `template_gms_48_1.json` binds **no `CharacterEffect` writer at
all** (its only effect writers are `FieldEffect`, `FieldEffectWeather`,
`MonsterSpecialEffectBySkill`). The self and foreign SKILL_USE broadcasts are
therefore already dropped on v48 for every skill — a pre-existing gap, not one
this task creates. FR-6 is inert on v48, and v48 sends no direction byte anyway
(§2.2). This is documented, not fixed here: adding the v48 `CharacterEffect`
route means deriving its opcode and the v48 `operations` mode table, which is a
separate, larger piece of work.

### 4.7 Handler registration

New package `skill/handler/monstermagnet/`, `init()` registering all three
Identities in the **`Handler`** registry (FR-7.1, FR-7.2), blank-imported from
`skill/handler/registrations/registrations.go`. Layout follows `dispel/`.

`AttackCastHandler` is wrong here for the reason `registry.go` documents: the
magnet arrives on the use-skill opcode, not an attack packet, and it deals no
damage. Registration in `Handler` also means `UseSkill` has already charged
`mpCon` before the handler runs — the handler must not charge it again.

---

## 5. Error handling

| Condition | Behaviour |
|---|---|
| Reader short / malformed magnet body | Decoder reads what the wire has; a truncated body yields zero-valued entries which the validator drops as `objectId == 0`. No panic path is introduced (the reader is the shared `request.Reader`). |
| Caster load fails | Whole cast dropped, `error`, `event=monster_magnet_caster_load_failed` |
| Rect query fails | Whole cast dropped, `error`, `event=monster_magnet_rect_query_failed` |
| Over-cap claim | Whole cast dropped, `warn`, `event=monster_magnet_anomaly_over_cap` |
| Target outside region | That target dropped, rest proceeds, `warn`, `event=monster_magnet_anomaly_out_of_rect` |
| `CATCH_MONSTER` announce fails for one mob | Logged, other mobs continue |
| Either command emit fails | Logged, other mobs continue; the handler still returns nil so `UseSkill` does not log a second error |
| Monster gone by the time a command is consumed | `Debugf` + drop, both handlers |
| Character absent from field / GM-hidden on force-control | `Debugf` + drop |

The handler always returns `nil`, matching `dispel`/`heal`: a partial failure
must never abort the caller's `EnableActions` unlock.

Anomaly log field names deliberately parallel the existing
`monster_buff_anomaly_*` vocabulary (caster id, skill id, level, claimed ids,
server ids, computed rect) so existing dashboards pick them up (NFR
observability), with the `magnet` discriminator in the event name.

---

## 6. Testing

**`libs/atlas-packet`** — table-driven decode tests over both shapes, plus the
FR-8 byte fixtures:

- One `packet-audit:verify`-marked byte fixture per version (10), each built
  from the read order derived above, with the evidence record pinned. These pin
  the **magnet arm** of the use-skill opcode; they do **not** promote the
  `SPECIAL_MOVE` matrix row, whose other fifteen fnames stay unverified (FR-8.3).
  The n-a / matrix consistency gates must be re-run and left green.
- Non-regression: a mob-affecting-buff cast, a party-buff cast, a Shadow Stars
  cast and a Resurrection cast decode byte-identically before and after.
- v48 specifics: leading caster entry discarded; `entryCount == 1` (caster only,
  zero mobs) yields an empty grab list; `delay` populated; `Direction()` false.
- Modern specifics: `grabCount == 0`; a mixed `grabbed` true/false set; a
  zero `objectId` entry; `direction` true and false.

**`atlas-channel`** — the handler's seams are the existing `var`-function
pattern (`loadCasterFunc`, `rectQueryFunc`), so tests run offline:

- over-cap → zero emits; out-of-region → partial emits + anomaly log;
  `grabbed == false` → no emit for that mob; caster-load / rect-query failure →
  zero emits; missing `range` → cap-only path.
- exactly one rect query per cast regardless of grab count.
- recipients: caster excluded from the `CATCH_MONSTER` fan-out.
- `Direction()` reaches `NewEffectSkillUse`'s `monsterMagnetLeft`.

**`atlas-monsters`**:

- `ClearDamageEntries`: full wipe; flag flips true→false and emits once; wiping
  an empty table emits nothing and returns nil; unknown monster returns the
  not-found sentinel and the handler drops it.
- Interaction: after a wipe, one `MonsterAggroDecayTask.Run()` tick is a no-op.
- `ForceControl`: happy path emits `STOP_CONTROL` (when previously controlled)
  then `START_CONTROL` with `controllerHasAggro = true`; same-controller is a
  no-op with zero emits; absent character, hidden character and missing monster
  each drop with no emit.

Test setup uses the project Builder pattern throughout — no `*_testhelpers.go`.

---

## 7. Files touched

```
libs/atlas-packet/model/skill_usage_info.go            magnet branch, MagnetGrab, getters, builder
libs/atlas-packet/model/skill_usage_info_*_test.go     decode tests + 10 byte fixtures
docs/packets/audits/...                                evidence records (10)

services/atlas-channel/atlas.com/channel/
  data/skill/effect/rest.go, model.go                  + Range (decode + getter)
  skill/handler/mob_select.go                          + magnetRegion, extracted cap/anomaly helpers
  skill/handler/common.go                              applyToMobs uses the extracted helpers
  skill/handler/monstermagnet/monstermagnet.go         NEW handler
  skill/handler/registrations/registrations.go         + blank import
  socket/handler/effects.go                            + directed announce variants
  socket/handler/character_skill_use.go                pass sui.Direction()
  monster/processor.go, producer.go                    + ClearAggro, ForceControl emitters
  monster/mock/processor.go                            + mocks
  kafka/message/monster/kafka.go                       + 2 command types/bodies  (MIRROR)

services/atlas-monsters/atlas.com/monsters/
  kafka/consumer/monster/kafka.go                      + 2 command types/bodies  (MIRROR)
  kafka/consumer/monster/consumer.go                   + 2 handlers
  monster/model.go                                     + ControlWithAggro
  monster/registry.go                                  + ClearDamageEntries, ControlMonsterWithAggro
  monster/processor.go                                 + ClearAggro, ForceControl, startControl split
```

No template edits, no new codec, no migrations, no new REST endpoints,
no `go.mod` changes anticipated (so no `docker buildx bake` unless one appears).

The two `kafka.go` copies are separate Go modules; a divergence fails no build
(the trade contract has a dedicated guard for exactly this class of bug). They
must be edited in the same commit, and the plan should call that out as a single
work item rather than two.

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| v48 `entry[0]` interpretation wrong → first mob silently dropped | Disassembly-verified at `0x6AD977`/`0x6ADA89` (same object-id offset `0x654` for caster and mob). The v48 byte fixture encodes the caster entry explicitly, so a wrong reading fails the test rather than the game. |
| AABB region too permissive | Accepted trade-off, argued in §4.2: the gate exists to reject fabricated targets, not to re-implement client geometry. A too-strict gate produces player-visible failures; a slightly loose one produces none. |
| Repick storm from force-control | Bounded by `mobCount` ≤ 7 per cast; the gate that exists to prevent storms (`ControllerHasAggro`) is deliberately satisfied here. |
| `range` missing for some tenant's ingested data | Cap-only fallback with a debug log (§4.2). |
| Contract mirror drift between the two `kafka.go` copies | Same-commit edit; called out in the plan. A guard script is out of scope for this task. |

---

## 9. PRD deltas

These supersede the corresponding PRD text; everything else stands.

1. **FR-1.2** — the payload is not one shape. gms_61+ and jms use a **uint32**
   count with a per-entry bool and a trailing **direction byte and no delay**;
   gms_48 uses a **byte** count, **no** per-entry result, a trailing **delay
   short and no direction**, and a **leading caster-id entry**. §2.
2. **FR-1.4** — the gate is `IsRegion("GMS") && !MajorAtLeast(61)`, i.e. v48
   only. Not the v72 boundary the anti-repeat gate uses.
3. **FR-2.1 / FR-2.4 / FR-2.6** — Monster Magnet has no `lt`/`rb`; the rect
   check is derived from WZ `range` (a new field on atlas-channel's effect
   model) as the AABB of the client's trapezoid. Only the cap check, the
   intersection and the anomaly-log shape are shared with `applyToMobs`. §3, §4.2.
4. **FR-3.1 / FR-3.4** — `CATCH_MONSTER` goes to **other** sessions in the field,
   not all, because the caster renders it locally (`CMob::OnHit` `0x668E22`).
   The grab result on the wire is a **boolean**, so `result = 1, success = 1`.
   §4.3.
5. **FR-6** — already implemented generically; only the `left` argument is new.
   FR-6 is **inert on gms_48**, which binds no `CharacterEffect` writer at all.
   §4.6.

## 10. Deferred, with reason

- **gms_48 `CharacterEffect` writer route.** Real gap, but it needs the v48
  clientbound opcode derived plus a v48 `operations` mode table, affects every
  skill rather than the magnet, and would drag an unrelated packet-audit pass
  into this branch. Recorded here so it is not rediscovered as a magnet bug.
- **`SPECIAL_MOVE` matrix promotion.** Out of scope per PRD non-goals; the row
  carries fifteen other fnames.
