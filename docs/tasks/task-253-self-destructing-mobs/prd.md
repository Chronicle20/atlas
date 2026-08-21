# Self-Destructing Mobs — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

A small family of monsters in the MapleStory data set do not die the ordinary way. Instead of
running out of HP and playing the standard death animation, they **detonate** — either when
their HP drops below a data-defined threshold, or after a data-defined timer elapses. The WZ
data expresses this as a `selfDestruction` block on the mob's `info` node carrying three fields:
`action` (the death animation the client must play), `hp` (the HP threshold, `-1`/absent when the
mob is timer-driven), and `removeAfter` (the timer, `-1`/absent when the mob is HP-driven).

The most visible case is the Papulatus boss fight. Papulatus summons bomb mobs `8500003` and
`8500004`, which are supposed to explode — either because a player damaged them past the
5000-HP threshold, or because the client's controller reported that the bomb's first-attack
body rect touched the local user (`CMob::TryFirstSelfDestruction` → the serverbound
`MONSTER_BOMB` packet). The Boomer family (`5100002`) works the same way at a 1800-HP threshold,
as do several event and boss-summon mobs.

Atlas today parses `selfDestruction` in `atlas-data` and carries it through two REST DTO hops,
but **nothing reads it**. The damage path in `atlas-monsters` has no threshold check; the death
path in `atlas-channel` hardcodes a single destroy animation; the `MONSTER_BOMB` serverbound
handler is a decode-and-log stub. Net player-visible effect: Papulatus's bombs spawn and then
sit there, ignoring both damage thresholds and player contact, until killed normally. This task
makes `selfDestruction` a live mechanic end-to-end.

## 2. Goals

Primary goals:

- A monster whose `selfDestruction.hp > -1` is force-killed the moment its HP falls to or below
  that threshold, on any damage path that can reduce its HP.
- A monster whose `selfDestruction.hp < 0` is force-killed when its `selfDestruction.removeAfter`
  timer elapses after spawn.
- The client renders the correct death animation for a self-destruct — the WZ `action` byte
  reaches the client as the `CMobPool::OnMobLeaveField` dead-type, correctly encoded for every
  supported client version.
- The serverbound `MONSTER_BOMB` packet detonates the reported mob instead of being logged and
  dropped.
- All self-destruct paths award drops.

Non-goals:

- `MOB_TIME_BOMB_END` (`mob_time_bomb_end.go`), a separate serverbound stub for a different
  mechanic (Papulatus's timed field bomb). It stays a stub.
- Papulatus boss-fight scripting beyond the bomb mobs themselves — summon cadence, phase
  transitions, and the clock mechanic are out of scope.
- Monster Carnival mobs `9400547` / `9400550`. Their `selfDestruction` blocks are handled by the
  generic mechanic if they happen to spawn, but Monster Carnival itself is not being implemented
  here and is not a test target.
- The top-level `info/removeAfter` field (`information/rest.go:15`), which is a distinct
  "despawn after N" mechanic unrelated to `selfDestruction`. Only the
  `selfDestruction.removeAfter` timer is in scope.
- Any change to how ordinary (non-self-destructing) monsters die.

## 3. User Stories

- As a player fighting Papulatus, I want the bombs he summons to explode when I damage them past
  their threshold, so the fight plays out the way the content was designed.
- As a player fighting Papulatus, I want a bomb that drifts into me to explode on contact, so I
  am punished for standing in it rather than being able to ignore it.
- As a player killing a Boomer, I want it to play its explosion animation rather than the generic
  fade-out, so the mob reads as the bomb it is.
- As a player, I want to receive drops and EXP from a self-destructing mob I damaged, so killing
  it is worth the same as killing anything else.
- As a server operator, I want mobs with a `selfDestruction` timer to remove themselves on
  schedule, so summoned bombs do not accumulate in the field indefinitely.

## 4. Functional Requirements

### 4.1 Monster information plumbing

- **FR-1.1** The `atlas-monsters` `information.Model` domain type MUST carry the
  `selfDestruction` data. Today it exists only on the REST DTO
  (`services/atlas-monsters/atlas.com/monsters/monster/information/rest.go:38,53`) and has no
  accessor on `model.go`.
- **FR-1.2** The value MUST be exposed as an immutable value type with accessors for `Action`
  (byte), `RemoveAfter` (int32), and `Hp` (int32), reachable from the `Model` via a
  `SelfDestruction()` accessor, and set through the existing `ModelBuilder`.
- **FR-1.3** `Extract` in `information/rest.go` MUST map the DTO into the model. The builder,
  model, and Extract additions MUST follow the existing immutable-model + builder conventions in
  that package.
- **FR-1.4** A monster with no `selfDestruction` block in WZ MUST produce the reader's zero value
  — `action=0`, `removeAfter=-1`, `hp=-1` (see
  `services/atlas-data/atlas.com/data/monster/reader.go:206`) — and MUST be treated as
  "not self-destructing" by every consumer.
- **FR-1.5** The predicate for "this monster self-destructs on an HP threshold" is
  `hp > -1`. The predicate for "this monster self-destructs on a timer" is `hp < 0` **and** a
  `selfDestruction` block was present. Because `hp=-1` is also the absent-block default, the
  model MUST expose an explicit `Present()` (or equivalent) predicate so the two cases are
  distinguishable; a monster with no block MUST NOT be treated as timer-driven.

### 4.2 HP-threshold self-destruction

- **FR-2.1** In `atlas-monsters`, after damage lines have been applied and before the ordinary
  kill decision, the monster's post-damage HP MUST be compared against
  `selfDestruction.Hp` when `Present() && Hp > -1`.
- **FR-2.2** If post-damage HP `<=` the threshold, the monster MUST be killed with the
  self-destruct animation, and the ordinary "killed because HP reached zero" path MUST NOT also
  fire. Exactly one death is emitted.
- **FR-2.3** The check MUST live on the shared damage path (`damageCore`,
  `services/atlas-monsters/atlas.com/monsters/monster/processor.go:552`) so that every caller
  benefits: character attacks (`Damage`), Mortal Blow (`Kill`), and any future damage source.
- **FR-2.4** Damage-over-time ticks (poison/venom, `status_task.go`) MUST also trip the
  threshold. If those ticks do not currently route through `damageCore`, the design MUST either
  route them through it or apply the same check at their kill site — a Boomer poisoned below its
  threshold explodes.
- **FR-2.5** The threshold check MUST run after the full attack's lines are applied, matching the
  existing overkill-discard behavior — a multi-line attack that crosses the threshold on line 2
  produces one self-destruct, not one per line.
- **FR-2.6** A monster already below its threshold when the check runs (e.g. spawned at low HP)
  MUST self-destruct on the next damage event. There is no requirement to self-destruct at spawn
  time on the HP predicate alone.

### 4.3 Timer-driven self-destruction

- **FR-3.1** On spawn, a monster with `selfDestruction.Present() && Hp < 0` MUST be registered
  with a self-destruct timer of `selfDestruction.RemoveAfter`.
- **FR-3.2** When the timer elapses, the monster MUST be killed with the self-destruct animation,
  exactly as the HP-threshold path does.
- **FR-3.3** The timer MUST be cancelled if the monster dies, is destroyed, or the field is torn
  down first. A cancelled timer MUST NOT fire.
- **FR-3.4** The timer MUST be tenant-scoped and keyed by the monster's unique id, and MUST be
  modeled on the existing per-monster scheduler patterns in the service
  (`drop_timer_registry.go` / `drop_timer_task.go`, `status_task.go`, `recovery_task.go`) rather
  than a new bespoke mechanism.
- **FR-3.5** `RemoveAfter <= 0` MUST fire on the next scheduler tick rather than being treated as
  "no timer". (Mobs `9300166` and `9300329` carry `removeAfter=0`.)
- **FR-3.6** Server restart / registry teardown MUST NOT leak timers; `DestroyAll` / `Teardown`
  MUST clear the registry.

### 4.4 Death animation on the wire

- **FR-4.1** The `KILLED` and `DESTROYED` monster status events emitted by `atlas-monsters` MUST
  carry the death animation (dead-type) byte so `atlas-channel` can render it. Today
  `statusEventKilledBody` and `statusEventDestroyedBody`
  (`services/atlas-monsters/atlas.com/monsters/monster/kafka.go:94,132`) have no such field and
  the channel hardcodes `DestroyTypeFadeOut`.
- **FR-4.2** The field MUST default to the current behavior (fade-out, `1`) for every ordinary
  death, so no existing death changes on the wire.
- **FR-4.3** `atlas-channel`'s `handleStatusEventKilled` and `handleStatusEventDestroyed`
  (`kafka/consumer/monster/consumer.go:216,314`) MUST pass the event's animation through to
  `NewMonsterDestroy` instead of hardcoding fade-out.
- **FR-4.4** `libs/atlas-packet/monster/clientbound/destroy.go` MUST define constants for the
  dead-type values the WZ data actually uses. The data set uses `action` values **1, 3, 4, 5**
  (see §6.3); only `0`, `1`, and `4` have constants today.
- **FR-4.5** The `destroyType == 4` trailing `int32` MUST be version-gated. IDA confirms the
  branch exists on v95 (`CMobPool::OnMobLeaveField` @ `0x658b90`:
  `if (v4 == 4) v5 = CInPacket::Decode4(...)`) but **not** on v83 (@ `0x67961d`: `Decode4` mob id,
  `Decode1` dead type, no further reads). The current unconditional
  `if m.destroyType == DestroyTypeSwallow { w.WriteInt(...) }` in `destroy.go` would desync a v83
  client. The gate MUST use the `MajorAtLeast` idiom, never a raw comparison.
- **FR-4.6** The exact set of client versions that read the trailing field, and the client-side
  meaning of each dead-type value, MUST be derived from the IDB for every supported version — not
  assumed. `CMobPool::OnMobLeaveField` and the consumer of `CMob::m_nDeadType` are the anchors.
  Any version where a WZ `action` value is not a legal dead-type MUST be handled explicitly
  (documented fallback), not sent blind.
- **FR-4.7** No wire change is permitted for an already-verified packet × version cell beyond
  what FR-4.5 requires. If the coverage matrix marks `MonsterDestroy` verified for a version, the
  change MUST be re-verified for that cell.

### 4.5 `MONSTER_BOMB` serverbound handler

- **FR-5.1** `services/atlas-channel/atlas.com/channel/socket/handler/monster_bomb.go` MUST stop
  being decode-and-log. It MUST act on the decoded `mobId`.
- **FR-5.2** The handler MUST be **data-driven, not mob-id-driven**: any monster whose
  information carries a `selfDestruction` block is a legal detonation target. Cosmic's reference
  handler hardcodes `8500003`/`8500004`; Atlas does not.
- **FR-5.3** The handler MUST reject the request when the reporting character is dead, when the
  target monster is not in the character's field, or when the target has no `selfDestruction`
  block. A rejected request is logged at debug and produces no state change.
- **FR-5.4** On accept, the handler MUST cause the monster to be killed with its WZ
  `selfDestruction.Action` animation — via the existing channel → `atlas-monsters` command topic,
  not by writing a destroy packet directly from the channel. The channel is not the authority on
  monster life-cycle.
- **FR-5.5** The detonation MUST be idempotent: a second `MONSTER_BOMB` for a mob already dead or
  already removed is a no-op, not an error. Multiple clients in the field may each report the
  same first self-destruction.
- **FR-5.6** The command MUST be tenant/world/channel scoped like every other monster command.

### 4.6 Drops and rewards

- **FR-6.1** Every self-destruct path — HP threshold, timer expiry, and `MONSTER_BOMB` contact —
  MUST award drops. This is a deliberate divergence from Cosmic, which awards drops only on the
  HP-threshold path.
- **FR-6.2** On the HP-threshold path the killer is the character whose damage crossed the
  threshold, exactly as an ordinary kill. Drop ownership, party ownership, quest-drop filtering,
  and EXP distribution behave identically to a normal kill.
- **FR-6.3** On the timer and `MONSTER_BOMB` paths there may be no attacking character. The kill
  MUST attribute to the monster's damage leader when damage entries exist, and to "no killer"
  (`ActorId = 0`) when they do not.
- **FR-6.4** `atlas-monster-death` MUST tolerate `ActorId = 0`: `CreateDrops`
  (`services/atlas-monster-death/atlas.com/monster/monster/processor.go:41`) must not error, the
  drop must spawn unowned (no owner character, no owner party), and the quest-drop filter must
  exclude quest-specific drops rather than including them. EXP distribution over an empty
  `DamageEntries` list MUST be a no-op, not an error.
- **FR-6.5** A self-destruct kill MUST perform the same death bookkeeping as an ordinary kill:
  cooldown/attack-cooldown clearing, drop-timer unregistration, status-effect cancellation
  events, registry removal, and revive spawning.

## 5. API Surface

No new HTTP endpoints. Two REST payload shapes and two Kafka message shapes change.

### 5.1 REST — monster information (unchanged shape, newly consumed)

`GET /monsters/{monsterId}` from `atlas-data`, consumed by `atlas-monsters`, already returns:

```json
{
  "self_destruction": { "action": 3, "remove_after": -1, "hp": 5000 }
}
```

No wire change required — the field is already present and already parsed. Only the consumer
side (`information.Model`, `Extract`) grows.

### 5.2 Kafka — monster status events (`KILLED`, `DESTROYED`)

`statusEventKilledBody` and `statusEventDestroyedBody` in
`services/atlas-monsters/atlas.com/monsters/monster/kafka.go` gain a death-animation field.
Proposed:

```json
{
  "x": 100, "y": -200, "actorId": 4321, "boss": false,
  "damageEntries": [ { "characterId": 4321, "damage": 1800 } ],
  "deathType": 3
}
```

- Field is additive. An older producer omits it; the consumer MUST default the missing/zero value
  to the existing fade-out behavior so no ordinary death changes appearance (FR-4.2).
- The consumer is `atlas-channel`'s `handleStatusEventKilled` / `handleStatusEventDestroyed`.
  `atlas-monster-death` also consumes `KILLED` and MUST be unaffected by the new field.

### 5.3 Kafka — monster command (detonate)

`atlas-channel` needs a way to ask `atlas-monsters` to detonate a mob. Two options for the
design phase to choose between:

1. **Reuse `CommandTypeKill`** with an added "reason"/animation discriminator. `Kill` today
   (`processor.go:1751`) fail-closed *drops bosses* and always awards drops via `damageCore` —
   semantics tuned for Mortal Blow, not for a bomb.
2. **A new `CommandTypeSelfDestruct`** carrying `{ uniqueId, characterId }`, with the animation
   resolved server-side from the monster's own `selfDestruction.Action` rather than trusted from
   the client.

Option 2 is the recommended default: the client must never dictate the death animation, and the
boss guard on `Kill` is wrong for this mechanic (`9300266`/`9300267` and the Papulatus bombs are
boss-fight summons).

### 5.4 Packet — `MonsterDestroy` (clientbound, `CMobPool::OnMobLeaveField`)

Unchanged for existing values. Adds constants for dead-types `3` and `5`, and version-gates the
trailing `int32` on dead-type `4` (FR-4.5). Error cases: none on the wire — the packet has no
failure response.

### 5.5 Packet — `MONSTER_BOMB` (serverbound, `CMob::TryFirstSelfDestruction`)

Codec unchanged (`libs/atlas-packet/monster/serverbound/monster_bomb.go`, single `uint32 mobId`,
identical across all five versions). Only the handler's behavior changes.

## 6. Data Model

### 6.1 New value type — `information.SelfDestruction` (atlas-monsters)

| Field | Type | WZ source | Absent default |
|---|---|---|---|
| `action` | `byte` | `selfDestruction/action` | `0` |
| `removeAfter` | `int32` | `selfDestruction/removeAfter` | `-1` |
| `hp` | `int32` | `selfDestruction/hp` | `-1` |
| `present` | `bool` | block exists | `false` |

`present` is derived at Extract time, not carried on the REST DTO. Because the DTO's zero value
(`{0,-1,-1}`) is indistinguishable from "block absent with all defaults", the design MUST decide
whether to derive `present` from the DTO zero value or to add an explicit marker to the
`atlas-data` payload. Deriving it is sufficient: no real mob has an all-default block.

No database entities, no migrations, no `tenant_id` columns — this is all in-memory registry state
plus derived reference data. The self-destruct timer registry is in-memory and tenant-keyed like
its siblings (`GetDropTimerRegistry()`, `GetCooldownRegistry()`).

### 6.2 Kafka body field

`deathType byte` added to `statusEventKilledBody` and `statusEventDestroyedBody` (§5.2).

### 6.3 Reference data — every mob with `selfDestruction` (Cosmic `wz/Mob.wz`, 12 mobs)

| Mob id | action | hp | removeAfter | Trigger | Notes |
|---|---|---|---|---|---|
| 8500003 | 3 | 5000 | — | HP | Papulatus bomb (high) |
| 8500004 | 3 | 5000 | — | HP | Papulatus bomb (low) |
| 9300266 | 3 | 5000 | — | HP | |
| 9300267 | 3 | 5000 | — | HP | |
| 6300004 | 1 | 5000 | — | HP | |
| 8510100 | 1 | 5000 | — | HP | |
| 5100002 | 1 | 1800 | — | HP | Boomer |
| 9400566 | 1 | 99997 | 60 | HP (hp > -1 wins) | Threshold is effectively "any damage" |
| 9400547 | 5 | 350 | 90000 | HP | Monster Carnival — out of scope as content |
| 9400550 | 5 | 350 | 90000 | HP | Monster Carnival — out of scope as content |
| 9300166 | 4 | *(absent, -1)* | 0 | Timer | Fires immediately |
| 9300329 | 4 | *(absent, -1)* | 0 | Timer | Fires immediately |

Observations that shape the implementation:

- Distinct `action` values in the data set: **1, 3, 4, 5**. Only `1` (fade-out) and `4` (swallow,
  v95 only) have Atlas constants today; `3` and `5` have none.
- `action=4` collides with `DestroyTypeSwallow`. On v95 that path encodes a trailing
  `swallowCharacterId` that a self-destruct has no value for. This MUST be resolved during design
  (FR-4.6) — writing `4` with a zero trailing int, or discovering that the mob's dead-type
  consumer treats `4` differently than the swallow case, are both possible outcomes, but neither
  may be assumed.
- Only two mobs (`9300166`, `9300329`) are timer-driven, and both use `removeAfter=0`. The
  timer path is therefore low-traffic but must exist for correctness.
- `9400566`'s `hp=99997` sits above any plausible max HP, meaning it detonates on the first hit.
  It also carries `removeAfter=60`; because `hp > -1`, the HP predicate wins and the timer is not
  armed (Cosmic parity).

### 6.4 Unit of `removeAfter` — unresolved

Cosmic treats `selfDestruction.removeAfter` as **seconds**
(`MapleMap.java:1868`, `SECONDS.toMillis(selfDestruction.removeAfter())`). Under that reading
`9400547`'s `90000` is 25 hours, which is not a plausible design value, while `9400566`'s `60` is
plausible. Under a milliseconds reading, `90000` is 90 seconds (plausible) and `60` is 60ms
(not plausible). Neither reading is consistent across the data set. See §9.

## 7. Service Impact

### `atlas-data`
Likely **no change**. `selfDestruction` is already parsed
(`atlas.com/data/monster/reader.go:206`) and serialized (`monster/rest.go:43`). A change is only
needed if the design chooses an explicit "block present" marker over deriving it (§6.1).

### `atlas-monsters`
The bulk of the work.
- `monster/information/model.go`, `builder.go`, `rest.go` — carry `selfDestruction` onto the
  domain model (FR-1.1 – FR-1.3).
- `monster/processor.go` `damageCore` — HP-threshold check (FR-2.1 – FR-2.5).
- New self-destruct timer registry + task, modeled on `drop_timer_registry.go` /
  `drop_timer_task.go` (FR-3.1 – FR-3.6).
- `monster/kafka.go` — `deathType` on the killed/destroyed bodies (FR-4.1).
- `kafka/consumer/monster/consumer.go` — new detonate command handler (FR-5.4).
- `monster/producer.go` — emit the animation on the killed/destroyed providers.

### `atlas-channel`
- `socket/handler/monster_bomb.go` — replace the stub with validate-and-command (FR-5.1 – FR-5.6).
- `kafka/consumer/monster/consumer.go:216,314` — pass `deathType` through to `NewMonsterDestroy`
  instead of hardcoding `DestroyTypeFadeOut` (FR-4.3).
- The channel's monster mirror (`GetLiveMirror`) is the natural source for the "is this mob in my
  field" check in FR-5.3.

### `atlas-monster-death`
- Must tolerate `ActorId = 0` on the `KILLED` event (FR-6.4). May need no code change if the
  existing degradation is already correct — `filterByQuestState` already excludes quest drops on
  lookup failure (`processor.go:104-110`) and `party.GetByMemberId` failure already yields
  `ownerPartyId = 0` (`processor.go:61-64`). This MUST be verified, not assumed, and covered by a
  test.
- `information/rest.go:37` already mirrors the DTO; no shape change needed unless it starts
  reading the field.

### `libs/atlas-packet`
- `monster/clientbound/destroy.go` — new dead-type constants; version-gate the trailing `int32`
  on type 4 (FR-4.4 – FR-4.6); byte-fixture tests per affected version.
- Coverage matrix (`docs/packets/audits/status.json` / `STATUS.md`) re-verification for any
  `MonsterDestroy` cell whose bytes move.

### `atlas-ui`
No change.

## 8. Non-Functional Requirements

- **Performance.** The HP-threshold check runs on the hottest path in the server — every damage
  line on every monster. It MUST NOT introduce a per-hit remote lookup. `damageCore` already
  fetches `information.Model` once per damage event for the boss/revives decision
  (`processor.go:553-559`); the self-destruct data MUST ride that same lookup, and the
  information processor's existing cache (`information/cache.go`) MUST absorb the read.
- **Multi-tenancy.** The self-destruct timer registry MUST be tenant-keyed, following the
  existing registry pattern. A timer for tenant A must never fire against tenant B's monster.
- **Ordering / idempotence.** Detonation MUST be safe under concurrent triggers: two damage
  events crossing the threshold simultaneously, or a timer firing at the same instant as a
  contact report, MUST produce exactly one `KILLED` event. The existing registry's
  `ApplyDamage` compare-and-set is the model.
- **Observability.** Each detonation MUST log at debug with mob id, unique id, trigger
  (`threshold` / `timer` / `contact`), and the resolved action byte. A rejected `MONSTER_BOMB`
  MUST log its rejection reason.
- **Security.** `MONSTER_BOMB` is a client-controlled packet naming an arbitrary mob id. The
  handler MUST NOT trust it beyond "which mob": the animation comes from server-side WZ data, the
  field-membership and liveness checks are server-side, and a mob without a `selfDestruction`
  block MUST NOT be killable through this path. Otherwise the packet becomes a
  kill-anything-in-any-map primitive.
- **Backwards compatibility.** Rolling deploy must be safe in both orders: an old
  `atlas-channel` reading a `deathType` it does not know, and a new `atlas-channel` reading a
  `KILLED` event that omits it (FR-4.2).

## 9. Open Questions

1. **Unit of `selfDestruction.removeAfter`.** Cosmic reads it as seconds; the data set is
   internally inconsistent under either reading (§6.4). **Proposed resolution:** derive from the
   client during design — `CMob`'s remove-after handling / `CMobTemplate` load — rather than
   guessing. Because both in-scope timer mobs use `removeAfter=0`, the unit does not block
   implementation; it only affects the two out-of-scope Monster Carnival mobs and `9400566`
   (whose HP predicate wins anyway). If the client derivation is inconclusive, default to seconds
   for Cosmic parity and document it.
2. **`action=4` vs `DestroyTypeSwallow`.** Both `9300166` and `9300329` carry `action=4`, and on
   v95 dead-type 4 encodes a trailing `swallowCharacterId` (IDA-confirmed). Does the client's
   dead-type consumer branch on the swallow case only when a swallow character is set, or is
   dead-type 4 unconditionally the swallow animation? Must be resolved from the IDB (FR-4.6).
3. **Which client versions read the trailing field?** Confirmed present on v95 (`0x658b90`),
   confirmed absent on v83 (`0x67961d`). v48, v61, v72, v79, v84, v87, v92 and JMS v185 are
   unverified and MUST be derived.
4. **Are dead-types 3 and 5 legal on every supported version?** Older clients may not have the
   corresponding mob animation. Needs IDA derivation plus a documented fallback (FR-4.6).
5. **Do DoT ticks route through `damageCore`?** FR-2.4 requires the threshold to trip on poison.
   `status_task.go` computes poison damage independently; whether it shares the kill path is a
   design-phase finding.
6. **Command reuse vs new command type** for detonation (§5.3). Recommendation is a new
   `CommandTypeSelfDestruct`; the design phase decides.
7. **Do the Papulatus bombs currently spawn at all in Atlas, and via which path?** The idea note
   asserts "generic summon works". This should be confirmed on a live channel before the branch
   is called done, because a bomb that never spawns cannot be observed to explode.

## 10. Acceptance Criteria

Data plumbing:

- [ ] `information.Model` in `atlas-monsters` exposes `SelfDestruction()` with `Action`,
      `RemoveAfter`, `Hp`, and a presence predicate; set via the builder; mapped in `Extract`.
- [ ] A monster with no WZ `selfDestruction` block reports not-present and is never treated as
      self-destructing (unit test).

HP threshold:

- [ ] Damaging a mob with `selfDestruction.hp = 1800` (Boomer, `5100002`) from full HP to `1800`
      or below force-kills it (unit test on `damageCore`).
- [ ] The self-destruct kill emits exactly one `KILLED` event, not two (unit test).
- [ ] A multi-line attack crossing the threshold mid-attack produces exactly one death
      (unit test).
- [ ] A poison/DoT tick crossing the threshold detonates the mob (unit test).
- [ ] A mob with no threshold (`hp = -1`) dies normally at zero HP — no behavior change
      (regression test).

Timer:

- [ ] A mob with `hp < 0` and `removeAfter = 0` self-destructs shortly after spawn (unit test).
- [ ] The timer is cancelled when the mob dies or is destroyed first; a cancelled timer never
      fires (unit test).
- [ ] `DestroyAll` / teardown leaves no live self-destruct timers (unit test).

Wire:

- [ ] `KILLED` and `DESTROYED` carry `deathType`; an event without it renders as fade-out
      (unit test).
- [ ] `atlas-channel` writes the event's `deathType` into `MonsterDestroy` rather than a hardcoded
      `DestroyTypeFadeOut`.
- [ ] `destroy.go` has constants for every `action` value in the WZ data set (1, 3, 4, 5), each
      with an IDA citation for its client-side meaning.
- [ ] The trailing `int32` on dead-type 4 is version-gated via `MajorAtLeast`, with byte-fixture
      tests proving a v83 encode is 5 bytes and a v95 encode is 9 bytes.
- [ ] Every `MonsterDestroy` coverage-matrix cell whose bytes changed is re-verified and the
      evidence re-pinned; `packet-audit` checks exit 0.

`MONSTER_BOMB`:

- [ ] The handler no longer contains a `// behavior: deferred` comment.
- [ ] Reporting a mob that has a `selfDestruction` block, is alive, and is in the reporter's
      field detonates it with that mob's WZ action.
- [ ] Reporting a mob with no `selfDestruction` block is rejected and logged (unit test) — the
      packet cannot kill an arbitrary monster.
- [ ] Reporting a mob not in the reporter's field is rejected (unit test).
- [ ] Reporting from a dead character is rejected (unit test).
- [ ] A duplicate report for an already-dead mob is a silent no-op (unit test).

Rewards:

- [ ] An HP-threshold self-destruct awards drops and EXP to the damaging character exactly as a
      normal kill (unit test).
- [ ] A timer or contact detonation with no damage entries emits `KILLED` with `ActorId = 0`,
      `atlas-monster-death` creates unowned drops without error, quest-specific drops are
      excluded, and EXP distribution is a no-op (unit test in `atlas-monster-death`).
- [ ] A self-destruct kill clears cooldowns, unregisters the drop timer, emits status-cancel
      events, and spawns revives — same as an ordinary kill (unit test).

Gates:

- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` and `plan-adherence-reviewer` run before the PR.
- [ ] Live observation on a channel: a Papulatus bomb (`8500003`) damaged past 5000 HP plays the
      self-destruct animation and drops, and a bomb touched by a player detonates via
      `MONSTER_BOMB`. If the bombs do not spawn (§9.7), that is reported explicitly rather than
      silently skipped.
