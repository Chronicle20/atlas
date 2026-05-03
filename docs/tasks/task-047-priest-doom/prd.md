# Priest Doom (Skill 2311005) — Product Requirements Document

Version: v2 (revised after wrong-channel discovery)
Status: Draft
Created: 2026-05-03 (v1), revised 2026-05-03 (v2)
Predecessor: see `postmortem.md` for the v1 → v2 discovery story.

---

## 1. Overview

The Priest job skill **Doom** (id `2311005`, master level 30) is a
non-damaging area magic skill that polymorphs affected monsters into snails
for the skill's duration. In v83 the snail visual and the elemental-resistance
normalization are entirely client-side — they trigger off the
`MORPH`/`DOOM` mask bit on the server-broadcast `MonsterStatSet` packet.
The server's responsibilities are limited to: routing the cast,
selecting the affected monsters, applying the `DOOM` monster-status entry
to each, broadcasting the status packet, charging the skill's
`itemConsume` cost (Magic Rock), and removing the status when its duration
expires. Reference behaviour comes from the Cosmic source tree:

- `client/SkillFactory.java:250` — Doom is registered as a Priest skill.
- `server/StatEffect.java:823-825` — Doom maps to `MonsterStatus.DOOM` in
  the effect builder.
- `server/StatEffect.java:1531` — Doom is in `isMonsterBuff`.
- `server/StatEffect.java:1180-1204` — `applyMonsterBuff` does
  server-authoritative bounding-box selection from the caster's position
  + the skill's `lt`/`rb` rectangle, applies the status to up to
  `mobCount` monsters in that rectangle.
- `net/server/channel/handlers/SpecialMoveHandler.java:138` — the buff
  (SPECIAL_MOVE) packet handler is what invokes `effect.applyTo`, which
  dispatches into `applyMonsterBuff` for monster-buff skills.

The cast does **not** flow through the magic-attack opcode. The v83 client
sends Doom over the SPECIAL_MOVE opcode only; the packet body carries
cast position, not a list of affected monster ids. (See `postmortem.md`
for the empirical confirmation.)

The atlas wiring that already exists end-to-end:

- atlas-data effect mapping for skill 2311005 (`reader.go:351-352`),
  populating `MonsterStatus["DOOM"] = 1` and a non-zero duration.
- atlas-monsters command consumer for `APPLY_STATUS`
  (`kafka/consumer/monster/consumer.go:91-119`), and
  `ApplyStatusEffect` with the elemental / boss immunity gates.
- atlas-channel `monster.Processor.ApplyStatus` that emits the Kafka
  command and the `MonsterStatSet` broadcast wiring with the DOOM
  mask bit.
- atlas-channel `CharacterUseSkillHandle` (the SPECIAL_MOVE handler)
  that dispatches into `handler.UseSkill` and applies HP/MP/cooldown
  costs and party buffs.

What does **not** exist today: any code path that turns a Doom cast
(arriving via the buff packet, with no client-supplied mob list) into
`ApplyStatus` calls per affected mob. `handler.UseSkill`'s `applyToMobs`
short-circuits because `info.AffectedMobIds()` is empty. There is also no
generic `itemConsume` charge anywhere in the skill cast pipeline.

This PRD scopes the work to add server-authoritative monster selection
for Doom and to plumb generic `itemConsume` charging into the buff-path
cost block — both of which mirror canonical Cosmic v83 behavior.

## 2. Goals

Primary goals (unchanged from v1):

- A Priest with skill `2311005` learned can cast Doom and cause every
  legal target in the LT/RB area to receive the `DOOM` monster status
  for the skill's duration. Targets are selected server-side from the
  caster's current position and facing.
- The v83 client renders affected mobs as snails for the duration and
  resumes the original sprite at expiry without the player having to
  rejoin the map. (Verified by wire-level evidence: `MonsterStatSet`
  with the DOOM bit, followed by `MonsterStatReset` with the same bit
  at expiry.)
- Doom's status-apply path is exempt from the existing poison/freeze
  elemental-immunity gate so an element-resistant mob does not silently
  reject the DOOM status. (Already implemented in atlas-monsters at
  `processor.go:1107-1111`. Stays.)
- Bosses do not receive the DOOM status. The existing
  `isBossAllowedStatus` allow-list rejects DOOM. (Already pinned by an
  atlas-monsters test. Stays.)
- Magic-reflect targets do not receive Doom. The new buff-path handler
  consults the reflect mirror per affected mob and skips reflect mobs.
  Doom does no damage, so no reflect damage is emitted.
- A grep-friendly debug log line is emitted at status-apply time
  identifying the caster, the affected monster id, the skill, the
  level, and the duration. (Already implemented at
  `services/atlas-channel/atlas.com/channel/monster/processor.go:73`.
  Stays.)
- The Magic Rock (`itemConsume = 4006000`, `itemConsumeAmount = 1` per
  the v83 effect data) is consumed once per cast. Generic across every
  current and future `itemConsume` skill (Mystic Door, summons, mists,
  …).

Non-goals (mostly unchanged from v1):

- No other Priest skills are in scope.
- No server-side polymorph entity swap. Polymorph-to-snail is a
  v83 client-side effect of the DOOM mask bit; the server does not
  change the spawned monster id.
- No server-side elemental damage recomputation while a mob is
  Doomed.
- No XP award for the cast itself.
- No new Kafka topic or event type.
- No work on the Solution test framework (task-042); per-package unit
  tests in atlas-channel mirror the `skill/handler/heal/` style.
- No work in `libs/atlas-packet`. The buff packet for Doom carries
  cast position + skill id/level + trailing byte; no list-of-mobs
  bytes are exchanged. atlas-channel does not need cast-position
  decoding because the bounding box uses caster position (Cosmic
  parity, `StatEffect.java:1181`).

## 3. User Stories

(Unchanged from v1.)

- As a Priest player, I want to cast Doom on a group of regular mobs so
  they visibly turn into snails and become harmless for the duration.
- As a Priest player, I want Doom to land on element-resistant mobs
  (fire imps, ice mobs) so the skill is useful for its intended
  counter-niche.
- As a Priest player, I want Doom to *not* affect bosses so I do not
  waste MP and a Magic Rock on an immune target.
- As a Priest player, I want each Doom cast to consume one Magic Rock
  (per the skill's `itemConsume` data), and the cast to fail
  client-side if I have none.
- As a server operator diagnosing a stuck mob report, I want a single
  log line per Doom apply that names the caster, the target, and the
  duration so I can reconstruct the timeline quickly.

## 4. Functional Requirements

### 4.1 Cast intake (corrected from v1)

- The buff (SPECIAL_MOVE) packet handler
  (`services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go`)
  routes a packet whose `SkillId() == 2311005` through `handler.UseSkill`
  (the existing common buff path).
- A new per-skill handler is registered for `PriestDoomId` in
  `services/atlas-channel/atlas.com/channel/skill/handler/doom/`. The
  registration uses the same `channelhandler.Register(skill2.PriestDoomId, Apply)`
  pattern as `skill/handler/heal/heal.go:40`.
- HP / MP cost: charged by the existing `UseSkill` cost block
  (`skill/handler/common.go:22-27`). For skill 2311005 level 30, the
  effect data carries `MPConsume = 88, HPConsume = 0`.
- Item cost: a new `itemCon` consume call is added to the same `UseSkill`
  cost block, after the MP charge. It is generic — runs for any
  `e.ItemConsume() > 0`. For Doom, this charges one Magic Rock
  (`item.Id(4006000)`) from the caster's ETC compartment.
- Cooldown: handled by the existing `UseSkill` cooldown block
  (`common.go:28-30`). For Doom, `e.Cooldown()` is 0 in the v83 data,
  so no cooldown is recorded.

### 4.2 Target resolution (corrected from v1)

- The new Doom handler computes the target rectangle from the caster's
  current position + facing direction + the skill effect's `LT()`/`RB()`
  rectangle, mirroring Cosmic's `calculateBoundingBox`
  (`StatEffect.java:1206-1218`):
  - facing right: `[caster.x - rb.x, caster.y + lt.y, caster.x - lt.x, caster.y + rb.y]`
  - facing left:  `[caster.x + lt.x, caster.y + lt.y, caster.x + rb.x, caster.y + rb.y]`
- The handler queries atlas-monsters for monsters in that rectangle on
  the caster's field, capped to `e.MobCount()` (per the v83 data, 6
  for Doom). Result type: `[]monster.Model` (the channel-side view).
- For each returned mob:
  - Magic-reflect probe: if the mob has an active magic-reflect window
    via `monster.GetStatusMirror().GetReflect(t, mob.UniqueId(), "MAGICAL")`,
    skip the apply for that mob. Doom does no damage, so no reflect
    damage is emitted.
  - Otherwise: emit `mp.ApplyStatus(field, mob.UniqueId(), characterId,
    PriestDoomId, skillLevel, {"DOOM": 1}, e.Duration())`.
- Cosmic's `applyMonsterBuff` also runs `makeChanceResult()` (a `prop`
  RNG) per mob. The atlas-data effect data for Doom carries
  `prop = 0.52` per the v83 data. Mirror this: skip a mob with
  probability `(1 - prop)`. (Implementation: a `rand.Float64() <= prop`
  guard around the apply.)

### 4.3 Status apply (unchanged from v1)

- atlas-monsters' `ApplyStatusEffect` (`processor.go:1060-1098`) accepts
  the player-sourced DOOM effect and:
  - Bypasses the elemental-immunity gate via the explicit DOOM
    short-circuit in `isElementallyImmune` (`processor.go:1107-1111`,
    Cosmic parity at `StatEffect.java:1531`).
  - Rejects the apply if the target is a boss (`isBossAllowedStatus`
    does not include DOOM, so the default-reject branch returns
    `boss immunity`).
- The status is added to the monster's effect list (refresh semantics
  documented in `processor_test.go:TestApplyStatusEffect_Doom_ReapplyReplacesExisting`).
- A `STATUS_APPLIED` Kafka event is emitted; atlas-channel's broadcast
  pipeline turns it into a `MonsterStatSet` packet with the DOOM mask
  bit.

### 4.4 Status broadcast (unchanged from v1)

- The `STATUS_APPLIED` event flows through atlas-channel's existing
  monster status consumer; no new code is required.
- The `MonsterStatSet` packet carries the DOOM bit
  (`libs/atlas-packet/model/monster.go:108` —
  `TemporaryStatTypeDoom`). v83 clients render the snail and normalize
  the mob's elemental table.

### 4.5 Status expiry (unchanged from v1)

- The existing status-expiry timer in atlas-monsters fires
  `STATUS_EXPIRED`, which atlas-channel turns into `MonsterStatReset`
  with the DOOM bit. v83 clients restore the original sprite.

### 4.6 Cast logging

- Per-apply log line at
  `services/atlas-channel/atlas.com/channel/monster/processor.go:73`:
  `Doom: caster=[%d] monster=[%d] skill=[%d] level=[%d] duration=[%d]ms.`
  (Already implemented; fires on the corrected path because the new
  handler calls `mp.ApplyStatus`.)
- The new Doom handler also emits one summary line per cast at the end:
  `Doom: caster=[%d] level=[%d] mobsInRect=[%d] applied=[%d] reflectSkipped=[%d] propSkipped=[%d].`
  for diagnosis at the cast level (mirrors the heal handler's summary
  log at `skill/handler/heal/heal.go:166`).

### 4.7 Tests

Reuse what stays from v1:

- atlas-data: `TestReader_PriestDoom_MapsDoomStatus`. (Stays.)
- atlas-monsters: `TestApplyStatusEffect_Doom_BypassesElementalImmunity`,
  `TestApplyStatusEffect_Doom_RejectedOnBoss`,
  `TestApplyStatusEffect_Doom_ReapplyReplacesExisting`. (Stay.)

Add for the new handler:

- atlas-channel `skill/handler/doom/doom_test.go`:
  - `TestDoom_Apply_AppliesToMobsInRect` — given a caster + 3 mobs (2
    in rect, 1 outside), the handler emits exactly 2 `ApplyStatus`
    calls, with the right monster ids, skill id, level, and duration.
  - `TestDoom_Apply_RespectsMobCount` — given 8 mobs in rect and
    `e.MobCount() = 6`, exactly 6 `ApplyStatus` calls are emitted.
  - `TestDoom_Apply_SkipsMagicReflectMobs` — a mob with a magic-reflect
    window is excluded from the apply; no reflect damage is emitted.
  - `TestDoom_Apply_RespectsProp` — with `prop = 0.0`, no apply
    fires; with `prop = 1.0`, every in-rect mob receives the apply.
    (Implementation note: tests inject a deterministic RNG via a
    package-private hook to avoid flakiness.)
  - `TestDoom_Apply_LeftFacingRectMirror` — caster facing left,
    target only in the left-mirrored rectangle, applies.
- atlas-channel `skill/handler/common_test.go` (new or extended):
  - `TestUseSkill_ItemConsume_BurnsItem` — given an effect with
    `ItemConsume() > 0`, the cost block emits exactly one
    `RequestItemConsume` call with the slot of the matching asset.
  - `TestUseSkill_ItemConsume_LogsWarningOnMissingItem` — given a
    caster with no Magic Rock and an effect requiring it, the cost
    block logs the warning and proceeds with the cast.

## 5. Non-Functional Requirements

(Same as v1 — preserved here for completeness.)

- Backward compat: no Kafka schema or REST contract changes for existing
  endpoints. The new "monsters in rect" endpoint is additive.
- Multi-tenancy: every new query and command propagates the tenant via
  the existing context plumbing.
- Observability: Debugf log lines on the cast and apply paths; INFO/WARN
  for failure / item-missing edge cases.
- Performance: the rect query runs once per Doom cast. atlas-monsters
  already keeps an in-memory monster registry per tenant; the rect
  filter is an O(N) walk over field monsters, which is acceptable
  (typical map populations are well under 100 mobs).

## 6. Constraints

- Keep the bounding-box semantics aligned with Cosmic
  (`StatEffect.java:1206-1218`). Atlas's coordinate system uses the
  same axis convention (y increases downward, x increases right).
  Server-side facing-direction handling matters: the new handler must
  ask atlas-character (or read from the channel-side character model)
  for the caster's `Stance() & 1` parity to derive `isFacingLeft`.
- Generic `itemConsume` plumbing: must not double-charge for skills
  that have a per-skill handler (the handler is invoked by `UseSkill`
  *after* the cost block, so the cost block is the single source of
  truth for `itemConsume`).
- Do not require a server-side inventory/cap pre-check before the cast.
  The v83 client gates the cast UI on item availability; if the player
  somehow casts without the item, the warning is logged and the cast
  proceeds (matching today's HP/MP semantics, where cost is debited
  even if the resulting value would go negative — the existing
  `cp.ChangeHP` / `ChangeMP` paths handle the clamp).

## 7. Service / package impact

| Service / package | Action | Notes |
|---|---|---|
| `services/atlas-data/atlas.com/data/skill/reader.go` | None | Existing Doom mapping at line 351-352 stays. |
| `services/atlas-data/atlas.com/data/skill/reader_test.go` | None | Existing v1 reader test stays. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | None | DOOM short-circuit in `isElementallyImmune` and `testInformationLookup` extension stay. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` | None | Three Doom apply tests stay. |
| `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` | None | `SetBoss`/`SetResistances` stay. |
| `services/atlas-monsters` REST/processor | **Add** | New "monsters in field within rectangle" query. Endpoint, handler, processor method, REST model. |
| `services/atlas-channel/atlas.com/channel/monster/` | **Add** | Thin client wrapper for the new rect query. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | **Modify** | Add generic `itemConsume` charge after the MP block. |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/` | **Add (new package)** | New per-skill Doom handler. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | **Revert** | Remove the Doom-gated reflect probe (e05a1983a) and the `itemCon` consume in the cost gate (4a3312d6d / 9f1b14a00). Keep the helper extraction (03c84901c) and the imports it needs. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go` | **Revert** | Remove the four `TestProcessDamageInfoEntry_Doom_*` tests, the supporting helpers, and any imports left orphaned. |
| `services/atlas-channel/atlas.com/channel/monster/processor.go` | None | `Doom: caster=…` Debugf stays. |
| `libs/atlas-packet` | None | No decoder changes. |
| `libs/atlas-constants` | None | All constants exist (`PriestDoomId`, `StatusDoom`, `TemporaryStatTypeDoom`). |

## 8. Open Questions

1. **`prop` (cast probability) handling**: Cosmic applies a per-mob
   `makeChanceResult()` gate inside `applyMonsterBuff`. The v83 Doom
   data has `prop = 0.52`. **Decision needed**: do we honor `prop`
   per-mob (Cosmic parity) or always apply when in rect? The PRD
   currently mirrors Cosmic. Tests will need RNG injection to be
   deterministic.
2. **Field instance**: the rect query needs the caster's field
   (world + channel + map + instance). The new handler has the field
   in scope from `UseSkill`'s signature. The atlas-monsters processor
   already takes a `field.Model` for related queries.
3. **Facing derivation**: `Stance() & 1` is the OdinMS / Cosmic
   convention for "facing left" (odd stance). Confirm by reading
   atlas-channel's character model accessor for stance.

## 9. Risks

- **v83 packet structure variance**: if a future Priest skill is added
  on the buff path with a different packet shape, the new Doom handler
  is independent and unaffected. The `libs/atlas-packet`
  `SkillUsageInfo` decoder is unchanged, so existing buff-path skills
  are untouched.
- **Per-mob reflect probe cost**: each Doom cast triggers up to
  `mobCount` reflect lookups in the in-memory mirror. The existing
  mirror is per-tenant in-memory and O(stack-depth) per query (per the
  comment at `monster/status_mirror.go:63`). Negligible for
  `mobCount = 6`.
- **prop RNG flakiness**: the `prop = 0.52` gate means a Doom cast can
  apply to between 0 and 6 mobs even with 6 in rect. Tests inject a
  deterministic RNG. Production accepts the variance — Cosmic parity.
- **Duration unit**: atlas-data effect `Duration()` is in milliseconds
  (the reader's `time * 1000` multiplier when `Duration() == -1`). For
  Doom, the v83 data carries `time = 60` seconds → `Duration() = 60000`
  ms. atlas-channel's `mp.ApplyStatus` and atlas-monsters'
  consumer both treat `duration` as milliseconds (`time.Duration(c.Body.Duration) * time.Millisecond`).
  Consistent across the chain.

## 10. Acceptance Criteria

In-game (manual verification):

1. A Priest casts Doom on an empty area. No mobs receive the status; no
   Magic Rock is consumed. (Cosmic parity: `applyMonsterBuff` finds an
   empty list and returns silently.)
2. A Priest casts Doom on a single regular mob in range. The mob renders
   as a snail in the v83 client for the skill duration and resumes its
   original sprite at expiry. One Magic Rock is consumed.
3. A Priest casts Doom on a group of 3+ regular mobs in range, all on
   one side of the player. All in-rect mobs that pass the `prop`
   chance render as snails. One Magic Rock is consumed (per cast, not
   per mob).
4. A Priest casts Doom on a group containing a fire-immune mob. The
   fire-immune mob still receives DOOM (the elemental-immunity
   short-circuit is engaged).
5. A Priest casts Doom on a boss. The boss does not receive DOOM. The
   atlas-monsters log shows `Monster [..] is a boss. Status rejected.`.
   One Magic Rock is consumed (the cast itself succeeded; only the
   per-target status apply was rejected).
6. A Priest casts Doom on a mob standing on a magic-reflect window.
   That mob is excluded from the apply. No reflect damage is emitted
   (Doom does no damage). atlas-channel logs include the
   `Doom: monster [..] has MAGICAL reflect; status apply skipped.`
   line emitted by the new handler.
7. A Priest with zero Magic Rock attempts to cast Doom (e.g., via a
   third-party client that bypasses the UI gate). The cast proceeds
   per Cosmic / existing HP/MP semantics; the warning
   `Character [..] cast skill [..] requiring item [..] but no such item
   found in inventory; cast permitted (defense-in-depth gate only).`
   appears in the atlas-channel logs.
8. The grep `Doom: caster=\[` returns one line per Doom apply (i.e.,
   per affected mob).

Code (automated):

- All atlas-data, atlas-monsters, and atlas-channel test suites pass.
- The new `skill/handler/doom/doom_test.go` covers the apply, mob-count
  cap, reflect-skip, prop-gate, and left-facing branches.
- The new `skill/handler/common_test.go` extension covers the
  `itemConsume` charge and the missing-item warning.
