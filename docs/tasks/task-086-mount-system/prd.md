# Mount / Monster-Rider System — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-12
---

## 1. Overview

MapleStory's "mount" system (also called "monster rider" or "tamed monster") lets a
character ride a creature that grants increased movement speed and jump height, with a
distinct visual rendered both for the rider and for every other player who can see them on
the map. There are two families of mount:

1. **Tamed-monster mounts** — the Explorer/Cygnus flow. A character who owns the *Monster
   Rider* skill (beginner-band skill `jobType × 10000000 + 1004`), a **saddle** equip
   (item class `191`, e.g. `1912000`, in equip slot `-19`) and a **taming-mob** equip
   (item class `190`, e.g. Hog `1902000`, in equip slot `-18`) can cast the skill to mount
   the creature. These mounts carry persistent per-character progression (level, exp,
   tiredness) and are fed with "revitalizer" USE items.

2. **Skill-only mounts** — mounts that require no equipped saddle/taming-mob, granted purely
   by owning a skill: Yeti (`1932003`/`1932004`), Witch Broomstick (`1932005`), Balrog
   (`1932010`), and Space Ship (`1932000 + skillLevel`). These render identically but have
   no equip prerequisite and (per real-server behavior) no tiredness progression.

Mounting applies a `MONSTER_RIDING` character temporary stat (buff). The Atlas buff and
packet infrastructure already models this stat — `libs/atlas-constants/character/temporary_stat.go:119`
defines `TemporaryStatTypeMonsterRiding`, and `libs/atlas-packet/model/character_temporary_stat.go`
already places it in the trailing "base temporary stat" block of `GW_CharacterTemporaryStat`
for both the self-buff (`Encode`) and observer-buff (`EncodeForeign`) paths. The catch is
that the value is currently hard-coded to zero (`character_temporary_stat.go:720-721`,
`// TODO look up actual buff values if riding mount`), so the client receives the buff flag
but no vehicle to render. This task closes that gap and builds the surrounding mechanics.

Dismount occurs when the buff is cancelled (the player re-presses the skill / cancels the
buff) or on job change. Taking damage does **not** dismount (verified against the reference
server — there is no damage-driven dismount in the tamed-monster path; only the Corsair
Battleship, which is out of scope here, dismounts on HP depletion).

This feature also delivers the skill-acquisition questline (the "Riding Mimiana" line) so
that a fresh character can earn the Monster Rider skill and starter mount through gameplay
rather than a GM grant.

## 2. Goals

Primary goals:
- A character meeting the prerequisites can cast the Monster Rider skill and become mounted,
  with the correct creature rendered for themselves and for all observers on the same map.
- Cancelling the buff (or changing job) dismounts the character, removing the visual for
  self and observers.
- Tamed-monster mounts carry persistent per-character state — level, exp, tiredness — that
  survives logout/login and channel changes.
- Tiredness increases on a fixed cadence (every 60 seconds) while mounted, is clamped at 99,
  and notifies the player when the mount becomes too tired to ride.
- Feeding a revitalizer USE item reduces tiredness and converts the healed amount into mount
  exp, leveling the mount up to a cap.
- Skill-only mounts (Yeti, Broomstick, Balrog, Space Ship) render correctly from skill
  ownership alone, with no equip prerequisite and no tiredness progression.
- A player can complete the Riding Mimiana questline to be awarded the Monster Rider skill
  and a starter saddle + taming-mob.
- The wire encoding matches the v83 client exactly (verified via IDA): the Monster Riding
  base temporary stat carries `nOption = vehicle/taming-mob item id` and
  `rOption = skill id`.

Non-goals:
- The Corsair **Battleship** mount (`5221006` / item `1932000` Battleship): different
  lifecycle (HP-gated, dismounts on HP 0, has a cooldown on depletion). Deferred to a
  follow-up.
- Player-NPC / cash-shop mount acquisition flows beyond the questline and normal item
  acquisition.
- Mount-specific combat interactions beyond the standard speed/jump buff (the client
  renders movement; no new server-side combat rules).
- Any UI work in atlas-ui.

## 3. User Stories

- As a player who has earned the Monster Rider skill and equipped a saddle and a Hog, I want
  to press the skill so that I ride the Hog and move/jump faster.
- As a player near a mounted character, I want to see them riding their creature so the world
  feels consistent.
- As a mounted player, I want to press the skill again (cancel the buff) to dismount.
- As a player whose mount has grown tired, I want to feed it a revitalizer to restore it and
  have it gain experience and levels over time.
- As a player who changes job, I want to be dismounted automatically (the client expects
  this).
- As a player who logs out while owning a mount, I want my mount's level, exp, and tiredness
  to be exactly as I left them when I log back in.
- As a new player, I want to complete the Riding Mimiana questline to be awarded the Monster
  Rider skill and a starter mount.
- As a player with a skill-only mount (e.g. a Yeti), I want to ride it without needing an
  equipped saddle or taming-mob.

## 4. Functional Requirements

### 4.1 Prerequisite validation (tamed-monster mounts)

When a character casts a skill whose id satisfies `skillId % 10000000 == 1004`:
- FR-1.1: The character MUST own the skill at level ≥ 1 (existing skill-ownership check).
- FR-1.2: The character MUST have a taming-mob item equipped in slot `-18`
  (`libs/atlas-constants/inventory/slot` already reserves `-18` = `tamingMob`).
- FR-1.3: The character SHOULD have a saddle equipped in slot `-19`
  (`-19` = `saddle`). Match reference-server strictness: the saddle gates equipping the
  taming-mob; if both equips are required by the client, enforce both. (Open question 9.1.)
- FR-1.4: If prerequisites are not met, the cast MUST be a no-op (no buff applied, client
  re-enabled for actions) rather than an error to the player.

### 4.2 Skill-only mounts

- FR-2.1: For mount skills that require no equip (Yeti `1017`/`1018`, Witch Broomstick
  `1019`, Balrog `1031`, Space Ship `1013` in the beginner band, plus their Noblesse/Legend
  equivalents), casting applies the `MONSTER_RIDING` buff with a fixed vehicle id derived
  from the skill (Yeti1 → `1932003`, Yeti2 → `1932004`, Broomstick → `1932005`, Balrog →
  `1932010`, Space Ship → `1932000 + skillLevel`).
- FR-2.2: Skill-only mounts have **no** tiredness/exp/level progression and do not register
  with the tiredness ticker.
- FR-2.3: The skill→vehicle-id mapping MUST be resolvable from skill data. This requires
  extending the skill effect reader (`atlas-data/atlas.com/data/skill/reader.go`, currently
  a TODO at the documented line) so the mount vehicle id is available to the cast handler.

### 4.3 Mount activation (buff application)

- FR-3.1: On a successful cast, the system applies a `MONSTER_RIDING` character temporary
  stat to the character via atlas-buffs.
- FR-3.2: The buff MUST carry the **vehicle item id** (taming-mob item id for tamed mounts,
  or the fixed skill-mount vehicle id) and the **skill id**. Given the buff stat model
  (`buff/stat/model.go`) carries `statType` + a single `amount int32`, the vehicle id rides
  as the stat `amount` and the skill id rides as the buff's `sourceId`.
- FR-3.3: The self-buff packet (`CharacterBuffGive`) and the observer-buff packet
  (`CharacterBuffGiveForeign`) MUST both encode the Monster Riding base temporary stat with
  `nOption = vehicle item id` and `rOption = skill id`, replacing the zeroed placeholder at
  `libs/atlas-packet/model/character_temporary_stat.go:720-721`.
- FR-3.4: The buff applied to observers MUST be broadcast to all characters who can see the
  rider on the same map (existing buff broadcast path).
- FR-3.5: A character MUST NOT stack two mounts; casting while already mounted is a no-op or
  a re-apply (match client expectation — open question 9.2).

### 4.4 Dismount

- FR-4.1: Cancelling the `MONSTER_RIDING` buff (player re-presses the skill / sends the
  cancel-buff request) MUST dismount the character: the buff is removed for self and
  observers, and the tiredness ticker stops for that character.
- FR-4.2: A job change MUST cancel any active `MONSTER_RIDING` buff (and therefore dismount).
- FR-4.3: Taking damage MUST NOT dismount.
- FR-4.4: Logging out or changing channel while mounted MUST persist mount state (4.5) and
  clear the active-mount/ticker registration for that character on the departed channel.

### 4.5 Persistent mount state (tamed-monster mounts)

- FR-5.1: Each character has at most one persistent mount record holding `level`, `exp`,
  `tiredness`, and the timestamp of the last tiredness tick.
- FR-5.2: Mount state MUST be scoped by `tenant_id` + `character_id`.
- FR-5.3: Mount state MUST persist across logout/login and channel changes.
- FR-5.4: New characters (or characters who have never mounted) default to
  `level = 1, exp = 0, tiredness = 0`.
- FR-5.5: On mount activation, the persisted state is loaded and reflected to the client via
  the taming-mob-info packet (4.7).

### 4.6 Tiredness ("hunger") system

- FR-6.1: While a tamed-monster mount is active, tiredness increments by 1 every 60 seconds.
- FR-6.2: On each increment, the new tiredness is broadcast to the rider's map via the
  taming-mob-info packet (4.7).
- FR-6.3: Tiredness is clamped to a maximum of 99. When the clamp is hit, the player receives
  a notice: "Your mount grew tired! Treat it some revitalizer before riding it again!"
- FR-6.4: The ticker operates per active mount and MUST stop when the mount is dismounted or
  the character leaves the channel. (Atlas pattern: a registry of active-mount characters
  storing per-character world/channel context + a periodic task, mirroring the existing
  `atlas-buffs/tasks` expiration-task pattern.)

### 4.7 Taming-mob info packet (`SET_TAMING_MOB_INFO`)

- FR-7.1: A new outbound writer MUST encode: `int characterId, int level, int exp,
  int tiredness, byte levelUp`.
- FR-7.2: It is sent (broadcast to the rider's map) on: mount activation, each tiredness
  increment, and on exp/level change from feeding.
- FR-7.3: The writer's opcode is supplied per-tenant via the existing opcode configuration
  (no hard-coded opcode bytes).

### 4.8 Feeding (revitalizer items)

- FR-8.1: Using a revitalizer USE item while a mount exists heals up to 30 tiredness
  (capped at the current tiredness).
- FR-8.2: The healed fraction converts to mount exp:
  `exp += ceil((healed/30) × (2 × level + 6))`.
- FR-8.3: When `exp ≥ expNeededForLevel(level)` and `level < 31`, the mount levels up.
- FR-8.4: The revitalizer item is consumed (one per use).
- FR-8.5: The resulting level/exp/tiredness change is broadcast via the taming-mob-info
  packet (4.7) with the `levelUp` byte set appropriately.
- FR-8.6: The mount exp-to-level table MUST be defined (reference-server parity; values
  sourced from local data, not memory — see Open Question 9.4).

### 4.9 Skill-acquisition questline

- FR-9.1: The Riding Mimiana questline MUST be authored/converted so that completing it
  awards the Monster Rider skill (`1004` in the appropriate band) plus a starter saddle and
  taming-mob item, using the existing quest reward path (atlas-quest → atlas-skills skill
  grant; item grant via inventory).
- FR-9.2: Questline NPC conversation and quest definitions follow the project's
  `convert-quest` / `convert-npc` JSON conventions.

## 5. API Surface

This feature is primarily Kafka- and packet-driven; REST additions are limited to exposing
persistent mount state if needed for tooling/UI parity.

- **Inbound packet (existing route):** Monster Rider skill cast arrives through the existing
  special-move / skill-use handler in atlas-channel
  (`socket/handler/character_skill_use.go`). A mount branch is added for
  `skillId % 10000000 == 1004` and the skill-only mount skill ids.
- **Inbound packet (existing route):** revitalizer "use mount food" request — confirm whether
  the v83 client sends a dedicated mount-food opcode or routes through the standard
  use-item path (Open Question 9.3); add/extend the handler accordingly.
- **Outbound packet (new):** `SET_TAMING_MOB_INFO` writer in `libs/atlas-packet` +
  registration in atlas-channel's writer set, opcode per tenant config.
- **Outbound packet (modified):** `CharacterBuffGive` / `CharacterBuffGiveForeign` —
  Monster Riding base stat now carries real `nOption`/`rOption`.
- **Kafka:** mount activation/cancel reuse the existing `character_buff_status_event` topic
  (`Apply`/`Cancel` in atlas-buffs). Persistent mount state changes (tiredness tick, feed,
  level-up) emit a mount status/update event consumed by atlas-channel to drive the
  taming-mob-info packet. Exact topic/event shape defined in design.
- **REST (optional):** `GET` of a character's mount state for tooling, following JSON:API
  conventions, if the design finds it necessary for the channel↔character split.

Error cases: failed prerequisite checks are silent no-ops (FR-1.4); invalid/again-mount
casts are no-ops (FR-3.5); feeding with no mount is a no-op.

## 6. Data Model

New persistent entity (in the service that owns mount state — see Service Impact / Open
Question 9.5), modeled on the existing `saved_location` sub-entity pattern
(`atlas-character/.../saved_location/entity.go`):

`character_mounts`
- `id` (uuid, pk)
- `tenant_id` (uuid, not null) — multi-tenant scoping
- `character_id` (uint32, not null)
- `level` (int, not null, default 1)
- `exp` (int, not null, default 0)
- `tiredness` (int, not null, default 0)
- `last_tiredness_tick_at` (timestamp, nullable) — supports tick accounting across restarts
- Unique index on `(tenant_id, character_id)`.

Notes:
- The equipped saddle and taming-mob items themselves live in the existing equip
  compartment (slots `-19` / `-18`) in atlas-inventory — no new item storage.
- No change to the core `characters` entity (mount state is a separate sub-entity to avoid
  bloating the hot character row).
- Skill ownership (the Monster Rider skill) lives in atlas-skills as today.

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | **Fix** `getBaseTemporaryStats` Monster Riding base stat to carry `nOption = vehicle id`, `rOption = skill id` (`character_temporary_stat.go:720-721`). **Add** `SET_TAMING_MOB_INFO` writer. |
| `libs/atlas-constants` | Already has slot `-18`/`-19`, item classifications `190`/`191`, `TemporaryStatTypeMonsterRiding`. **Add** a revitalizer item classification if one is not already covered, and any mount-skill id constants needed. |
| `atlas-channel` | **Add** mount branch in the skill-use handler (prereq validation, resolve vehicle id, call atlas-buffs `Apply`). **Add** revitalizer/use-mount-food handling. **Register** the new taming-mob-info writer and emit it on activate/tick/feed. **Consume** mount status events. |
| `atlas-buffs` | Reuse `Apply`/`Cancel`; ensure the `MONSTER_RIDING` stat carries the vehicle id as `amount`. Job-change cancel already routes through `CancelAll`/`CancelByStatTypes`. |
| `atlas-character` | **Add** the `character_mounts` sub-entity + processor + producer (load on mount, persist on change), mirroring `saved_location`. Ensure job-change emits the buff-cancel that dismounts. |
| `atlas-data` | **Extend** `skill/reader.go` to expose mount vehicle ids for skill-only mounts (TODO at the documented line). |
| `atlas-consumables` | **Add** revitalizer effect handling: heal tiredness, grant mount exp/level, consume item, emit mount update. |
| `atlas-skills` | No code change expected; existing skill-grant path used by the questline reward. |
| `atlas-quest` | **Add/convert** the Riding Mimiana questline (quest definitions + NPC conversation) granting skill `1004` + starter items. |

The tiredness ticker lives wherever the active-mount registry lives (channel-side is the
natural home since it has world/channel context and drives the broadcast); the exact owner
is a design decision (Open Question 9.5).

## 8. Non-Functional Requirements

- **Multi-tenancy:** all persistent state scoped by `tenant_id`; all processors derive tenant
  from context (`tenant.MustFromContext`). Per-tenant opcode config for the new writer.
- **Wire correctness:** the Monster Riding base-stat encoding must match the v83 client
  byte-for-byte; verify with a byte-level encode test (the self and foreign buff paths both
  append the base-stat block). Vehicle-id / skill-id placement confirmed via IDA
  (`CAvatar::SetRidingVehicle` stores the vehicle id as the avatar look).
- **Performance:** the tiredness ticker scales with the number of actively-mounted characters
  per channel, not all characters; the existing task pattern polls a registry. A 60s cadence
  is cheap. Avoid per-character goroutines/timers in favor of one task iterating the registry.
- **Observability:** log mount activate/dismount/feed/level-up at debug with character id;
  surface no PII.
- **Redis discipline:** any cross-channel active-mount/registry state that uses Redis must
  route through `libs/atlas-redis` (repo invariant, `tools/redis-key-guard.sh`).
- **Resilience:** mount state persistence must be transactional with respect to the
  consumed event (feed/tick) so a crash doesn't double-apply exp or lose tiredness.
- **Verification:** every changed Go module passes `go test -race ./...`, `go vet ./...`,
  `go build ./...`; `docker buildx bake atlas-<svc>` for each service whose `go.mod` is
  touched; `tools/redis-key-guard.sh` clean.

## 9. Open Questions

- 9.1: Does the v83 client strictly require **both** a saddle (slot -19) and a taming-mob
  (slot -18) to render a tamed mount, or only the taming-mob? Confirm from client behavior /
  IDA before finalizing FR-1.3.
- 9.2: When a mounted character re-casts the mount skill, does the client expect a clean
  dismount (cancel), or a re-apply? Confirm to settle FR-3.5 / FR-4.1.
- 9.3: Does the v83 client send a dedicated "use mount food" opcode, or does revitalizer
  consumption route through the standard use-item path? Determines the FR-8 inbound handler.
- 9.4: Source the mount exp-to-level table values (reference-server `getMountExpNeededForLevel`)
  from local data/WZ rather than memory; confirm cap of level 31.
- 9.5: Where should persistent mount state + the active-mount registry/ticker live —
  atlas-character (owns persistence) with channel driving the buff/broadcast, or a thin
  channel-side registry calling atlas-character for persistence? Resolve in design.
- 9.6: Confirm the full set of mount **skill ids** across Explorer/Noblesse/Legend bands and
  their vehicle-id mappings against local skill data.
- 9.7: Confirm the `SET_TAMING_MOB_INFO` opcode is present in live tenant configs (seed
  templates apply only at tenant creation; existing tenants need a config patch — known
  pitfall for new opcodes).

## 10. Acceptance Criteria

- [ ] A character with the Monster Rider skill, an equipped saddle (-19) and taming-mob (-18)
      can cast the skill and become mounted; the correct creature renders for the rider.
- [ ] Other players on the same map see the rider on the correct creature.
- [ ] The self-buff and foreign-buff packets encode the Monster Riding base stat with the
      real vehicle item id and skill id (byte-level test passes; `getBaseTemporaryStats`
      TODO removed).
- [ ] Cancelling the buff dismounts the character for self and observers and stops the
      tiredness ticker.
- [ ] Changing job while mounted dismounts the character.
- [ ] Taking damage while mounted does NOT dismount.
- [ ] Mount level/exp/tiredness persist across logout/login and channel change.
- [ ] While mounted, tiredness increments every 60s, broadcasts via `SET_TAMING_MOB_INFO`,
      clamps at 99, and the player is notified at the clamp.
- [ ] Feeding a revitalizer heals up to 30 tiredness, grants exp per the formula, levels up
      to the cap, consumes one item, and broadcasts the update with the correct `levelUp` flag.
- [ ] Skill-only mounts (Yeti, Broomstick, Balrog, Space Ship) render from skill ownership
      alone with no equip prerequisite and no tiredness progression.
- [ ] Completing the Riding Mimiana questline awards the Monster Rider skill plus a starter
      saddle and taming-mob.
- [ ] The new `SET_TAMING_MOB_INFO` writer is registered and its opcode resolves from tenant
      config.
- [ ] All changed Go modules pass `go test -race`, `go vet`, `go build`; affected services
      pass `docker buildx bake`; `tools/redis-key-guard.sh` is clean.
- [ ] Battleship (Corsair) is explicitly NOT touched.
