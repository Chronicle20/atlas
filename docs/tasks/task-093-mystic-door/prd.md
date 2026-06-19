# Mystic Door (Priest) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-13
Baseline reference: Cosmic (`~/source/Cosmic`, v83) — `server/maps/Door.java`, `server/maps/DoorObject.java`, `net/server/channel/handlers/DoorHandler.java`, `net/server/world/Party.java`, `client/Character.java`, `tools/PacketCreator.java`.
---

## 1. Overview

Mystic Door (`skill id 2311002`) is the Priest (Magician 3rd job) utility skill that
creates a two-way town portal. On cast it consumes MP and a Magic Rock, then deploys
a **paired door object**: an *area door* at the caster's position in the current field
and a *town door* at a fixed slot in that field's return town. The door persists for a
level-scaled duration and is visible to — and usable by — the caster and every member
of the caster's party who is on the same channel. Walking into either door warps the
character between the field and the town.

This feature is the first persistent, party-shared, two-map field object in Atlas. It
is modeled structurally on the in-flight **`atlas-summons`** service (task-088): a
dedicated owner-bound object service backed by a per-tenant Redis registry, a shared
object-id allocator, leader-elected expiry sweeps, and Kafka command/event topics,
with `atlas-channel` acting as the thin per-version packet edge. The core door engine
is **version-agnostic**; all version variance is confined to the packet encoders in
`libs/atlas-packet` and the per-version opcode entries in tenant socket templates.

The feature must ship for **all supported tenant versions**: `gms_v83`, `gms_v84`,
`gms_v87`, `gms_v92`, `gms_v95`, and `jms_v185`. Packet handling is required in both
directions (clientbound spawn/remove/party-portal; serverbound enter-door).

## 2. Goals

Primary goals:
- A Priest can cast Mystic Door; on success an area door spawns at their position and a
  paired town door spawns at a non-overlapping slot in the field's return town.
- The door is visible to the caster and all same-channel party members in either map,
  and walking into either door warps the user between field and town.
- The door expires after a duration derived from the skill level and is cleaned up on
  caster disconnect, channel change, and the caster leaving the source field.
- MP and the Magic Rock item are consumed on cast (via the existing skill-cast cost path).
- Town-side door slots are deterministically assigned per party so doors never overlap.
- All of the above works on every supported tenant version with no per-version logic in
  the core engine.

Non-goals:
- Any other Priest / Magician skill (Holy Symbol, Bless, Resurrection, Doom, Dispel,
  Shining Ray, Summon Dragon, etc.). **Mystic Door only.**
- Super GM / admin door tooling, persistence of doors across server restart, or doors
  surviving a channel migration of the caster.
- atlas-ui changes.
- Cross-channel door visibility (doors are per-channel by design).

## 3. User Stories

- As a **Priest**, I want to cast Mystic Door so that my party and I can quickly return
  to and from town from a hunting field.
- As a **party member**, I want to see and use the doors my Priest party-mates create so
  that I can travel without my own town-scroll.
- As a **party member who just joined a party**, I want to start seeing that party's
  existing doors so that I can use them.
- As a **player who left a party**, I do **not** want to see that party's doors anymore.
- As a **Priest**, I want recasting Mystic Door to replace my previous door so I never
  hold two doors at once.
- As a **player**, I want a door I no longer have access to (expired, owner left) to
  disappear cleanly without crashing my client.

## 4. Functional Requirements

### 4.1 Cast & eligibility
- **FR-1.1** The `atlas-channel` skill-cast handler (`character_skill_use.go`) routes
  skill id `2311002` (`skill.PriestMysticDoorId`) to a door **spawn** command rather
  than treating it as a plain buff.
- **FR-1.2** Cast is rejected (no door, client re-enabled) when any of the following
  hold; each rejection must leave the client in a clean state (equivalent to Cosmic's
  `enableActions` / `blockedMessage`):
  - The caster's current map has field limit `FieldLimitNoMysticDoor` (`0x02`,
    `libs/atlas-constants/map/field_limit.go`) set.
  - The caster's current map is itself a town (`Town == true`) or has no valid return
    map (`ReturnMapId`/`ForcedReturnMapId` resolves to none).
  - The caster cannot deploy at the requested position (Cosmic `canDeployDoor`).
  - All town door slots for the caster's party are unavailable (see §4.4).
- **FR-1.3** MP and the skill's item cost (Magic Rock) are consumed on successful cast.
  Cost values are **data-derived** from the skill effect for the cast level (atlas-data),
  never hardcoded. Item consumption flows through the existing
  `REQUEST_ITEM_CONSUME` → `ConsumeBare` path already present in `atlas-consumables`.
  (See Open Question OQ-1 on whether the generic skill-cast path already emits the
  Magic Rock consume request.)
- **FR-1.4** Casting replaces the caster's existing door: any prior door owned by the
  caster is removed (with the same cleanup as expiry) before the new one is deployed.

### 4.2 Door object & placement
- **FR-2.1** A successful cast produces two linked door objects sharing a pair id:
  - **Area door** — placed in the caster's current field at the cast position; its
    linked target is the town and its linked portal id is the town door portal's id.
  - **Town door** — placed in the return town at the chosen town door portal's position;
    its linked target is the field and it is flagged "in town" (Cosmic encodes this as
    `linkedPortalId == -1`).
- **FR-2.2** Each door object occupies a field-level object id drawn from the **shared**
  object-id pool (`libs/atlas-object-id`, MinId 1,000,000) so door oids never collide
  with monsters / drops / reactors / summons on the same map.
- **FR-2.3** The return town is the field's `ReturnMapId` (falling back to
  `ForcedReturnMapId` semantics as MapleStory does). Town and field map metadata
  (return map, town flag, door portals) are sourced from `atlas-data`.

### 4.3 Visibility & broadcast
- **FR-3.1** In the **field**, the area door is shown (clientbound `spawnDoor`) only to
  the caster and same-channel party members present on that map.
- **FR-3.2** In the **town**, the town door is shown to the caster and same-channel party
  members present in town.
- **FR-3.3** The clientbound `spawnPortal` (town↔target minimap portal) and the party
  minimap door indicator (`partyPortal` — the long-standing "Write doors for party" TODO
  in `docs/TODO.md`, wired through the party-operation door fields already reserved in
  `libs/atlas-packet/party/clientbound/created.go`) are sent to party members so the
  door appears on their minimap.
- **FR-3.4** A player entering a map (field or town) that contains a door they are
  eligible to see receives the spawn packets for it; a player leaving stops tracking it.

### 4.4 Town slot assignment (non-overlap)
- **FR-4.1** The town-side door portal is chosen by the caster's **party door slot**: the
  caster's 0-based index in the party's members-sorted-by-history list (Cosmic
  `Party.getPartyDoor`). A solo caster uses slot `0`.
- **FR-4.2** The town door portal is the town map portal with id `0x80 + slot` (Cosmic
  `MapleMap.getDoorPortal` → `portals.get(0x80 + doorid)`). Because party size is capped
  at 6, slots `0..5` map to portals `0x80..0x85`, guaranteeing no two doors in the same
  party overlap. Slot/overlap constraints are scoped **per party**; doors owned by
  different parties may legitimately occupy the same portal id.
- **FR-4.3** When party membership changes (join/leave/leader change), the slots of all
  affected members' doors are recomputed and the town doors re-broadcast (Cosmic
  `updatePartyTownDoors` / `Door.updateDoorPortal`).

### 4.5 Entering / warping
- **FR-5.1** The serverbound enter-door packet (Cosmic `DoorHandler`: `int ownerId`,
  `byte direction` where `1` = town→target and `0` = target→town) is decoded by a new
  `atlas-channel` inbound handler.
- **FR-5.2** On enter, the server validates that a door owned by `ownerId` exists on the
  character's current map and that the character is the owner or a current party member;
  otherwise it sends the blocked message and re-enables actions.
- **FR-5.3** A valid enter warps the character to the linked map at the linked portal /
  position and plays the portal sound (Cosmic `playPortalSound`). The warp itself uses
  the **existing atlas-channel map-change path** (to be confirmed against Cosmic in
  design — see OQ-2); the door service validates the request and the channel performs
  the standard map transition.
- **FR-5.4** Enter is rejected while the character is mid-map-change or banned.

### 4.6 Lifecycle & cleanup
- **FR-6.1** A door expires automatically after its level-derived duration; on expiry the
  area and town doors are removed and `removeDoor` / cleared `partyPortal` packets are
  broadcast to all eligible viewers in both maps.
- **FR-6.2** A door is removed when its owner disconnects, changes channel, or leaves the
  source field (the field the area door is in).
- **FR-6.3** Door removal applies a short deploy-effect grace delay before the remove
  broadcast (Cosmic uses ~3000 ms from deploy) to avoid the client crash that occurs when
  a door is destroyed immediately after spawn (Cosmic comment: "doors crashing players
  when instantly cancelling buff").
- **FR-6.4 (party membership — chosen behavior, Cosmic parity).** This resolves PRD
  question #5:
  - A character **joining** a party begins seeing that party's existing doors; their own
    door (if any) is re-slotted into the party's slot ordering.
  - A character **leaving** a party immediately stops seeing that party's doors (removed
    from their client), and their own door is re-slotted to solo slot `0`.
  - Existing party members' town doors are re-slotted and re-broadcast on any membership
    change.
- **FR-6.5** Doors are **per-channel**: party members on a different channel than the
  caster neither see nor can use the door.
- **FR-6.6** Doors are ephemeral (Redis-backed, no relational persistence); they are not
  restored across a service/server restart.

### 4.7 Version coverage
- **FR-7.1** All clientbound (`spawnPortal`, `spawnDoor`, `removeDoor`, `partyPortal`,
  `playPortalSound`) and serverbound (enter-door) packets are implemented for every
  supported version: `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`.
- **FR-7.2** Per-version opcodes are added to the tenant socket templates (handlers +
  writers) for each supported version, and to any live tenant config that needs them
  (existing tenants do not auto-receive new handler/writer opcodes — see Known Bug
  Patterns: "New opcodes missing from live tenant config").
- **FR-7.3** Each new socket handler entry must declare a validator
  (`LoggedInValidator` for the enter-door handler) or it will be silently dropped at
  registration (see Known Bug Patterns: "Socket handler with missing/empty validator").
- **FR-7.4** Packet byte structures are verified per version against the IDA/WZ source of
  truth, not assumed from v83 (versions diverge above ~0x3D opcodes and in structure).

## 5. API Surface

### 5.1 New service `atlas-doors` (JSON:API + Kafka)
REST (read/debug, mirroring `atlas-summons`):
- `GET /maps/{mapId}/instances/{instanceId}/doors` — list doors in a field/instance.
- `GET /doors/{doorId}` — fetch a single door (owner, field, town, slot, positions,
  pair id, expiry).

Kafka command topic `COMMAND_TOPIC_DOOR`:
- `SPAWN` — `{ownerCharacterId, field, x, y, skillId, skillLevel}` (emitted by channel on cast).
- `ENTER` — `{characterId, ownerId, direction}` (emitted by channel on enter-door) — or
  resolved channel-side per OQ-2.
- `REMOVE` — `{ownerCharacterId}` (explicit removal; recast/leave-field).

Kafka event topic `EVENT_TOPIC_DOOR_STATUS` (consumed by `atlas-channel` to broadcast):
- `CREATED` — door deployed (area + town placement, slot, positions, pair id).
- `REMOVED` — door torn down (reason: expiry / disconnect / channel-change / left-field /
  recast).
- `SLOT_CHANGED` — town slot reassigned on party membership change (re-broadcast).

### 5.2 `atlas-channel` packet edge
Serverbound (new inbound handler): enter-door — `int ownerId`, `byte direction`.
Clientbound (new writers, per-version): `spawnPortal`, `spawnDoor`, `removeDoor`,
`partyPortal` (party-operation door fields), `playPortalSound`. Exact opcode names and
byte layouts to be filled from per-version source during design/plan.

### 5.3 Cross-service reads
- `atlas-parties` — caster's party membership and **history-sorted** member ordering
  (for slot assignment, FR-4.1) and the same-channel member set (for broadcast targeting).
- `atlas-data` — map metadata (`ReturnMapId`, `Town`, `ForcedReturnMapId`) and town door
  portals (`PortalTypeDoor = 6`; portal lookup by id `0x80 + slot`), and skill effect
  data (duration by level, MP cost, item cost).

## 6. Data Model

`atlas-doors` is Redis-backed, per-tenant, ephemeral (no relational migration), mirroring
`atlas-monsters`/`atlas-summons`.

| Field | Type | Notes |
|---|---|---|
| `doorId` (area object id) | `uint32` | From `libs/atlas-object-id` (shared pool). |
| `townDoorId` (town object id) | `uint32` | Paired object id (the linked door). |
| `pairId` | `uint32` | Links area↔town door. |
| `ownerCharacterId` | `uint32` | Caster. |
| `skillId` / `skillLevel` | `uint32` / `byte` | Drives duration; identifies Mystic Door. |
| `field` (world/channel/map/instance) | `field.Model` | The source field (area door map). |
| `townMapId` | `_map.Id` | Resolved return town. |
| `slot` | `byte` | Party door slot `0..5` → town portal `0x80+slot`. |
| `areaX` / `areaY` | `int16` | Area door position (cast position). |
| `townX` / `townY` | `int16` | Town door position (from town door portal). |
| `townPortalId` | `uint32` | `0x80 + slot`. |
| `deployTime` / `expiresAt` | `time` | Duration-driven expiry + deploy grace. |

Indices: per-field index (broadcast & map-enter spawn), per-owner index (recast/cleanup),
per-town index (slot allocation & town broadcast).

Multi-tenancy: all registry keys and Kafka payloads are tenant-scoped per the standard
`tenant.MustFromContext` / header-propagation pattern; Redis keys go through
`libs/atlas-redis` (rediskeyguard invariant).

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-doors** (new) | Owns door lifecycle: registry, id allocator, expiry sweep (leader-elected), slot allocation, command/event Kafka, REST. Version-agnostic engine. |
| **atlas-channel** | Route `PriestMysticDoorId` in `character_skill_use.go` to a SPAWN command; new inbound enter-door handler; new consumer of `EVENT_TOPIC_DOOR_STATUS` → broadcast spawn/remove/party-portal packets to in-range + party sessions; wire `partyPortal` into the party-operation door fields. |
| **libs/atlas-packet** | New `door/` clientbound encoders + serverbound decoder; party door-field encoding in `party/clientbound`. Per-version structures. |
| **libs/atlas-constants** | Reuse `skill.PriestMysticDoorId`, `map.FieldLimitNoMysticDoor`; add any door object-type / topic env constants as needed. |
| **atlas-consumables** | Magic Rock consumption already handled via `ConsumeBare`; confirm cast triggers it (OQ-1). |
| **atlas-parties** | Provide history-sorted member ordering + same-channel member set (confirm existing read suffices; may need a small read addition). |
| **atlas-data** | Map return/town/door-portal data already exposed; confirm town door portals (`0x80+slot`) are present in the data for all relevant towns. |
| **Tenant socket templates / live config** | Per-version handler + writer opcode rows for door packets (all 6 versions), with validators (FR-7.2/7.3). |
| **deploy / docker-bake / services.json** | Register `atlas-doors` (services.json + hardcoded `go_services` in docker-bake.hcl + go.work + Dockerfile COPY lines + k8s manifests). |

## 8. Non-Functional Requirements

- **Multi-tenancy:** every door, registry key, and event is tenant-scoped; Redis access
  via `libs/atlas-redis` only (rediskeyguard clean).
- **Concurrency:** registry guarded by RWMutex (`atlasredis.Registry`); slot allocation
  must be race-safe under concurrent casts in the same party/town.
- **Resilience:** door spawn must not depend on a config-load-once-then-Fatalf pattern;
  follow the config-status projection adoption (do not crash on tenant provisioned after
  pod start).
- **Cleanup correctness:** orphaned doors (owner gone, channel down) must be swept; the
  leader-elected expiry task is the backstop.
- **Client safety:** honor the deploy-effect grace delay (FR-6.3) so rapid cast→cancel
  cannot crash clients.
- **Observability:** log cast rejections (reason), spawn, slot assignment, and removal
  reason at appropriate levels; no silent drops.
- **Version verification:** packet bytes verified per version against IDA/WZ source, not
  ported blindly from v83 (Known Bug Patterns: opcode-table shift, MajorVersion off-by-one).

## 9. Open Questions

- **OQ-1 — Magic Rock consume trigger.** Does the generic skill-cast path already emit the
  `REQUEST_ITEM_CONSUME` for a skill's `itemCon` (the Magic Rock), or must the door
  spawn flow explicitly request it? The `atlas-consumables` `ConsumeBare` fallback names
  Mystic Door, implying *something* already requests it — confirm the emitter during
  design and avoid double-consumption.
- **OQ-2 — Warp ownership.** Confirm against Cosmic (`DoorObject.warp` → `changeMap`)
  whether the field↔town warp is performed entirely by the existing atlas-channel
  map-change path (channel validates door + issues standard map change) or whether
  atlas-doors should emit a warp command. User's expectation: handled "naturally through
  existing atlas-channel means."
- **OQ-3 — Town door portal data availability.** Verify that the `0x80+slot` door portals
  exist in `atlas-data` map data for the towns players actually return to across all
  supported versions; if a town lacks them, define the fallback (Cosmic `getDoorPortal`
  falls back to a default portal).
- **OQ-4 — Instance fields.** Define behavior when casting in an instanced field (most
  instances forbid Mystic Door via field limit; confirm `FieldLimitNoMysticDoor` covers
  them and that the return-map resolution is sane for instances).
- **OQ-5 — Per-version packet structure deltas.** Enumerate the exact opcode + byte layout
  for spawnDoor/removeDoor/spawnPortal/partyPortal/enter-door per version during design
  (v84 opcode-table shift, jms differences).

## 10. Acceptance Criteria

- [ ] A Priest casting `2311002` in a non-town field spawns an area door at their position
      and a town door at portal `0x80+slot` in the field's return town.
- [ ] Casting deducts MP and consumes one Magic Rock (values data-derived), with no
      double-consume.
- [ ] Recasting removes the caster's prior door before deploying the new one.
- [ ] Cast is rejected (clean client state) on a town map, a `NoMysticDoor` field-limit
      map, a map with no return map, or when no slot is available.
- [ ] Same-channel party members see both doors (field + town) and the party minimap
      door indicator; non-party / cross-channel players do not.
- [ ] Walking into either door warps owner and party members between field and town and
      plays the portal sound; ineligible players get the blocked message.
- [ ] The door expires after the level-derived duration and is removed cleanly from both
      maps; rapid cast→remove does not crash clients (grace delay honored).
- [ ] The door is removed on owner disconnect, channel change, and leaving the source field.
- [ ] Joining a party reveals that party's doors; leaving a party hides them and re-slots
      the leaver's own door to solo; remaining members are re-slotted/re-broadcast.
- [ ] All five clientbound packets and the serverbound enter-door packet function on
      `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, and `jms_v185`, with handler
      validators present and live tenant configs patched.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...`, `docker buildx bake
      atlas-doors` (and any other touched service), and `tools/redis-key-guard.sh` are all
      clean.
- [ ] `atlas-doors` is registered in `services.json`, `docker-bake.hcl` `go_services`,
      `go.work`, the shared `Dockerfile` COPY lines, and k8s manifests.
