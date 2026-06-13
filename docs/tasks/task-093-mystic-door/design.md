# Mystic Door (Priest) — Design

Status: Draft for review
Task: task-093-mystic-door
PRD: `docs/tasks/task-093-mystic-door/prd.md`
Baseline reference: Cosmic (`~/source/Cosmic`, v83)
Primary structural template: `atlas-summons` (task-088, in `.worktrees/task-088-player-summons`)

---

## 1. Summary

Mystic Door (skill `2311002`) introduces the first **persistent, party-shared, two-map
field object** in Atlas. We build a new version-agnostic engine service, **`atlas-doors`**,
modeled directly on `atlas-summons`: a per-tenant Redis registry, the shared object-id
allocator, a leader-elected expiry sweep, and Kafka command/event topics. `atlas-channel`
remains the thin per-version packet edge — it routes the cast, decodes the enter-door
packet, performs the warp through the existing portal path, and broadcasts spawn/remove/
party-minimap packets to eligible viewers. All version variance lives in a new
`libs/atlas-packet/door` package plus per-version tenant socket-template opcodes.

The design resolves all five PRD open questions (§9) and confirms the new-service approach
over two rejected alternatives (§3).

### Key resolutions up front

- **OQ-1 (Magic Rock / MP consume):** Already handled. `skill/handler/common.go:UseSkill`
  consumes HP/MP and emits `REQUEST_ITEM_CONSUME` for the skill's `itemCon` generically
  (lines 73–95) **before** any per-skill dispatch, and only applies a *character buff* when
  `Duration>0 && len(StatUps)>0`. Mystic Door's effect has a duration but **no statups**, so
  no phantom buff is applied. We hook the **existing per-skill `Lookup` dispatcher**
  (`common.go:121`) — the same seam Heal/Dispel/Cure/MPEater use — to emit the door SPAWN
  command. No new cost logic, no double-consume.
- **OQ-2 (warp ownership):** Channel-side. The enter-door handler validates against
  `atlas-doors` (synchronous REST read) and then warps via the **existing**
  `portal.Processor.Warp(field, characterId, targetMapId)` path. `atlas-doors` never issues
  a warp; it owns door state and validation only.
- **OQ-3 (town door portals):** Resolved with a documented fallback (§6.3). Flagged as the
  top data-verification risk for the plan phase.
- **OQ-4 (instance fields):** `FieldLimitNoMysticDoor` gate + town resolution covers it (§6.1).
- **OQ-5 (per-version packet bytes):** Enumerated as a plan-phase verification matrix (§9);
  byte layouts are NOT assumed from v83.

---

## 2. Architecture overview

```
                         ┌──────────────────────────────────────────────┐
                         │ atlas-channel  (per-version packet edge)      │
 cast 2311002 ─────────► │  skill/handler: door Lookup handler           │
                         │    └─ emits COMMAND_TOPIC_DOOR / SPAWN         │
 enter-door packet ────► │  socket/handler/door_enter: validate+warp     │
                         │    └─ REST GET door; portal.Warp(...)         │
                         │  kafka/consumer/door: EVENT_TOPIC_DOOR_STATUS  │
                         │    └─ broadcast spawnDoor/removeDoor/portal/   │
                         │       partyPortal to eligible sessions         │
                         └───────────────┬───────────────▲──────────────┘
                          COMMAND_TOPIC_DOOR     EVENT_TOPIC_DOOR_STATUS
                                         │               │
                         ┌───────────────▼───────────────┴──────────────┐
                         │ atlas-doors  (version-agnostic engine)        │
                         │  door/: Model+Builder, Registry (Redis),      │
                         │         IdAllocator, Processor                │
                         │  expiry_task.go: leader-elected sweep         │
                         │  slot allocation (per party, per town)        │
                         │  kafka/consumer/door:  SPAWN/REMOVE            │
                         │  kafka/consumer/character: LOGOUT/CHANNEL/MAP  │
                         │  kafka/consumer/party:  membership → re-slot   │
                         │  rest/: GET doors in field, GET door by id     │
                         └───────────────┬───────────────────────────────┘
                                         │ REST reads
                  ┌──────────────────────┼───────────────────────┐
                  ▼                      ▼                        ▼
            atlas-data            atlas-parties            (atlas-object-id /
       map return/town/portals   history-sorted members    atlas-redis libs)
       skill effect (duration)   same-channel set
```

**Division of responsibility**

| Concern | Owner |
|---|---|
| Door lifecycle, state, expiry, slot allocation | `atlas-doors` (engine) |
| Party-membership-driven re-slotting | `atlas-doors` (consumes party events) |
| Which sessions can *see* a door, and packet encoding | `atlas-channel` (edge) |
| Per-version packet bytes/opcodes | `libs/atlas-packet/door` + tenant templates |
| Warp execution | `atlas-channel` existing `portal.Warp` |
| Cost consumption (MP + Magic Rock) | existing `UseSkill` cost path (unchanged) |

The engine is **version-agnostic**: it never sees an opcode or a packet byte. The edge is
**stateless about doors**: it reads door state from `atlas-doors` (REST) or reacts to door
events (Kafka), and owns only the packet translation + session targeting.

---

## 3. Approaches considered

**(A) New standalone `atlas-doors` service — CHOSEN.** Mirrors `atlas-summons` one-for-one:
dedicated registry, id allocator, leader-elected expiry, command/event topics, REST. Doors
are an owner-bound, expiring, oid-occupying field object — structurally identical to summons.

- *Pros:* Reuses a proven, in-flight template (task-088); clean service boundary; the
  expiry/leader/slot logic lives in one testable place; honors the PRD mandate.
- *Cons:* One more service to register/deploy; cross-service reads for party + map data.

**(B) Fold door logic into `atlas-channel`.** No new service; the channel holds doors in a
local/Redis registry and broadcasts directly.

- *Rejected:* Doors are per-channel but their *state* (slot ownership, expiry, party
  re-slotting) is shared and must survive a single channel pod. Channel is the version edge;
  putting version-agnostic lifecycle + leader-elected sweeps there breaks the established
  edge/engine split and would duplicate the summons machinery. Also re-creates the
  config-load-once-Fatalf crash class the engine pattern avoids.

**(C) Extend `atlas-summons` into a generic "field objects" service.** Doors and summons
share an owner, an oid, an expiry, and a field.

- *Rejected (for now):* Premature generalization. Summons and doors diverge sharply: doors
  are paired across two maps, party-slotted, and warp-capable; summons attack/heal/move.
  A shared abstraction would be mostly conditionals. YAGNI — keep them separate; a future
  refactor can extract a common `libs/atlas-fieldobject` if a third object type appears.

---

## 4. New service: `atlas-doors`

Go module name: **`atlas-doors`** (short form, per the project convention —
`go.mod` module is `atlas-doors`, path `services/atlas-doors/atlas.com/doors`).

### 4.1 Package layout (mirrors atlas-summons)

```
services/atlas-doors/atlas.com/doors/
  main.go                       bootstrap, leader election, consumer + route init
  leaderconfig.go               DOOR_LEADER_* env parsing
  logger/                       logrus + ECS hook
  tasks/                        generic Task interface + Register (goroutine loop)
  door/
    model.go                    immutable Model (paired area+town door)
    builder.go                  fluent Builder
    registry.go                 Redis registry + indices (field / owner / town)
    id_allocator.go             wraps libs/atlas-object-id (shared pool)
    processor.go                Processor interface + Impl (Spawn/Remove/Get/Reslot)
    slot.go                     party slot → town portal computation
    expiry_task.go              leader-elected expiry sweep
    producer.go                 EVENT_TOPIC_DOOR_STATUS providers
    rest.go                     GET /doors/{doorId}
    resource.go                 RestModel + Transform (JSON:API)
    kafka.go                    StatusEvent envelope + body types + env const
  world/
    resource.go                 GET /worlds/.../maps/{mapId}/instances/{instanceId}/doors
  kafka/
    consumer/door/              COMMAND_TOPIC_DOOR: SPAWN, REMOVE
    consumer/character/         EVENT_TOPIC_CHARACTER_STATUS: LOGOUT/CHANNEL/MAP cleanup
    consumer/party/             EVENT_TOPIC_PARTY_STATUS: membership → re-slot
    producer/                   producer wrapper (spans + tenants)
  data/
    map/                        REST client → atlas-data map metadata + portals
    skill/                      REST client → atlas-data skill effect (duration by level)
  party/                        REST client → atlas-parties (history-sorted members)
```

### 4.2 Domain model (immutable + Builder)

A single `door.Model` represents the **pair** (area door + town door share one record with
one `pairId`); this avoids cross-record consistency problems on expiry/removal.

| Field | Type | Notes |
|---|---|---|
| `areaDoorId` | `uint32` | Object id of the field-side door (from shared pool). |
| `townDoorId` | `uint32` | Object id of the town-side door (second allocation). |
| `pairId` | `uint32` | Stable pair identifier (use `areaDoorId`). |
| `ownerCharacterId` | `uint32` | Caster. |
| `partyId` | `uint32` | `0` = solo. Drives slot ordering + visibility set. |
| `skillId` / `skillLevel` | `uint32` / `byte` | Identifies Mystic Door; drives duration. |
| `field` | `field.Model` | Source field (area door map: world/channel/map/instance). |
| `townMapId` | `_map.Id` | Resolved return town. |
| `slot` | `byte` | Party door slot `0..5`. |
| `townPortalId` | `uint32` | Resolved town door portal id (see §6.3). |
| `areaX` / `areaY` | `int16` | Cast position. |
| `townX` / `townY` | `int16` | Town door portal position. |
| `deployTime` | `time.Time` | Spawn instant (grace-delay anchor, FR-6.3). |
| `expiresAt` | `time.Time` | `deployTime + duration(level)`. |

`Reslot(slot, townPortalId, townX, townY)` returns a new Model (immutable copy) used by the
party-change path. No setters on Model.

### 4.3 Registry & indices (Redis via `libs/atlas-redis`)

Mirror summons exactly. Keys are tenant-scoped through `libs/atlas-redis` (rediskeyguard
clean — no raw go-redis keyed calls). A `storedDoor` JSON struct carries all Model fields
(times as unix-milli).

| Key | Purpose | Lookup |
|---|---|---|
| `door:{tenant}:{areaDoorId}` | Primary record. | `GetById`. |
| `door-field:{tenant}:{world}:{channel}:{map}:{instance}` (set) | Field index. | `GetInField` — area-door spawn on map-enter + field broadcast. |
| `door-owner:{tenant}:{characterId}` (set) | Owner index. | `GetByOwner` — recast replace, disconnect cleanup. |
| `door-town:{tenant}:{world}:{channel}:{townMap}:{partyId}` (set) | Town+party slot index. | Slot allocation (which slots taken) + town broadcast. |

Note the town index is keyed by **`partyId`** as well as town map, because slot/overlap
constraints are **per party** (FR-4.2): two different parties may legitimately occupy the
same town portal id. Solo casters use a synthetic party scope keyed by `ownerCharacterId`
(or a reserved `partyId=0` namespace disambiguated by owner) so two solo casters' slot-0
doors at the same town don't collide in the index.

### 4.4 Object-id allocation

Wrap `libs/atlas-object-id.Allocator` (shared per-tenant pool, `MinId=1_000_000`), exactly
like `summon/id_allocator.go`. **Two allocations per door** (area + town). On removal,
`Release()` both. Propagate `Allocate` errors and **fail the spawn** rather than falling back
to `MinId` (the silent-fallback collision bug noted in TODO.md) — a failed allocation emits
no CREATED event and the cast is a clean no-op.

### 4.5 Leader-elected expiry sweep

Copy `summon/expiry_task.go` + `leaderconfig.go`: `libs/atlas-lock` Redis leader election
(`DOOR_LEADER_ELECTION_ENABLED`/`_TTL`/`_REFRESH`/`_BACKOFF`), `tasks.Register` goroutine
loop ticking every second. Each tick enumerates `GetRegistry().GetAll()` grouped by tenant
and removes any door where `now.After(expiresAt)`, emitting `REMOVED{reason: expiry}`. This
is also the **orphan backstop** (owner gone without a clean event, channel down).

---

## 5. Kafka contracts

### 5.1 Command topic `COMMAND_TOPIC_DOOR` (consumed by atlas-doors)

`Command[E]` envelope (tenant/span via headers, body typed by `Type`):

- **`SPAWN`** — `{ownerCharacterId, field, x, y, skillId, skillLevel}`. Emitted by the
  channel door Lookup handler on cast. atlas-doors resolves town/slot/portal, allocates oids,
  persists, emits `CREATED`.
- **`REMOVE`** — `{ownerCharacterId, reason}`. Explicit removal (recast replace; left-field;
  manual). atlas-doors removes the owner's door, emits `REMOVED`.

> Recast (FR-1.4) is handled inside `SPAWN`: the processor first removes any existing
> owner door (same cleanup as expiry) before deploying the new one — a single atomic command,
> no separate REMOVE round-trip.

### 5.2 Event topic `EVENT_TOPIC_DOOR_STATUS` (consumed by atlas-channel)

`StatusEvent[E]` envelope keyed by map id (CreateKey on the area field map):

- **`CREATED`** — full placement: `{pairId, ownerCharacterId, partyId, field, townMapId,
  slot, areaX, areaY, townX, townY, areaDoorId, townDoorId, townPortalId, expiresAt}`.
  Channel broadcasts `spawnDoor` (field) + `spawnDoor`/`spawnPortal` (town) + `partyPortal`
  to eligible viewers in both maps.
- **`REMOVED`** — `{pairId, ownerCharacterId, partyId, field, townMapId, areaDoorId,
  townDoorId, slot, reason}`. Channel broadcasts `removeDoor` to both maps + clears
  `partyPortal`.
- **`SLOT_CHANGED`** — `{ownerCharacterId, partyId, townMapId, oldSlot, newSlot,
  townPortalId, townX, townY, ...}`. Emitted when a party membership change re-slots a
  town door (FR-4.3/FR-6.4). Channel re-broadcasts the town door at its new portal + updates
  `partyPortal`.

### 5.3 Consumed events (atlas-doors side)

- **`EVENT_TOPIC_CHARACTER_STATUS`** — `LOGOUT`, `CHANNEL_CHANGED`, `MAP_CHANGED`. Each
  triggers `RemoveAllForOwner(reason)`:
  - LOGOUT / CHANNEL_CHANGED → always remove the owner's door (FR-6.2, FR-6.5).
  - MAP_CHANGED → remove **only if** the owner left the area door's **source field**
    (FR-6.2). If the owner walks into their own town door (moving to the town the door
    already spans), that is a *warp*, not an abandonment — the door persists. The MAP_CHANGED
    handler compares the new field against the door's source field and town map to decide.
- **`EVENT_TOPIC_PARTY_STATUS`** — membership change (join/leave/leader/disband). Triggers
  the re-slot routine (§7).

---

## 6. Door placement (engine logic)

### 6.1 Cast eligibility (validated channel-side before SPAWN; re-checked engine-side)

The channel door Lookup handler performs the cheap, locally-available rejections so a
rejected cast never round-trips to atlas-doors and leaves the client cleanly re-enabled
(the existing `enableActions` after `UseSkill` already fires — see §8):

- Map has `FieldLimitNoMysticDoor` (`0x02`) set → reject (`map.FieldLimit & 0x02 != 0`).
- Map is itself a `Town`, or resolves to no valid return map → reject.

Engine-side, `Spawn` re-resolves town + slot and rejects if **no town slot is available**
(all of `0x80..0x85` taken for the party) by emitting no CREATED (the channel observes the
absence; see §8 note on rejection signalling). Cosmic's `canDeployDoor` position check is
applied with the data we have; a richer footprint check is out of scope (YAGNI) unless WZ
foothold data is already exposed by atlas-data.

### 6.2 Return town resolution

`townMapId = ReturnMapId` of the source map, with `ForcedReturnMapId` taking precedence when
set to a real map (MapleStory semantics: a forced-return overrides the normal return). Read
from atlas-data map metadata. Instances (OQ-4): most instanced fields carry
`FieldLimitNoMysticDoor`, so the cast is rejected before town resolution; for any instance
that does *not*, the resolved return map is the non-instanced town (warp out uses
`uuid.Nil`, consistent with the transports convention).

### 6.3 Town door slot → portal (OQ-3, the key data resolution)

Cosmic computes the town door portal as `portals.get(0x80 + slot)`. Atlas-data's portal
reader assigns **door-type portals (`PortalTypeDoor = 6`) sequential ids starting at 1 per
map load** — it does *not* expose a `0x80+slot` portal id. So we cannot look up `0x80+slot`
directly. Resolution:

1. Fetch the town map's portals from atlas-data and filter to `Type == PortalTypeDoor (6)`,
   in load order → an ordered list of door anchor positions.
2. The caster's **party door slot** (`0..5`, §7) indexes into that list:
   `townPortal = doorPortals[slot]`. `townX/townY = portal.X/Y`; `townPortalId` is the
   wire portal id the client expects for `0x80+slot` (the client addresses door portals as
   `0x80+slot` regardless of atlas-data's internal id — we encode `0x80+slot` on the wire,
   §8).
3. **Fallback (OQ-3 risk):** if the town has *fewer than 6* door-type portals (or none),
   fall back to a default door position near the town spawn portal (Cosmic `getDoorPortal`
   default-portal behavior). The fallback still encodes `0x80+slot` on the wire so the client
   places the minimap indicator correctly; overlapping fallback positions are acceptable
   degradation for a misconfigured town.

> **Plan-phase verification (top risk):** confirm that the towns players actually return to
> across all six versions expose ≥6 `PortalTypeDoor` portals in atlas-data. If not, the
> fallback path is exercised in normal play and may need richer default placement. This is a
> data question, answerable with a throwaway `GET /api/data/maps/{id}/portals` against a
> live tenant (see Observability memory).

The wire `townPortalId = 0x80 + slot` is what the client consumes; the atlas-data internal
portal id is used only to obtain the **position**.

---

## 7. Party slot allocation & membership changes

### 7.1 Slot rule (FR-4.1/4.2)

The town slot is the caster's **0-based index in the party's history-sorted member list**
(Cosmic `Party.getPartyDoor`). Solo caster → slot `0`. Party size ≤ 6 ⇒ slots `0..5` ⇒
town portals `0x80..0x85`, no intra-party overlap by construction. atlas-doors reads the
history-sorted member ordering from **atlas-parties** (the same ordering Cosmic uses; confirm
atlas-parties already returns members in stable history order — a small read addition if not).

### 7.2 Re-slot on membership change (FR-4.3 / FR-6.4 — Cosmic parity)

atlas-doors consumes `EVENT_TOPIC_PARTY_STATUS`. On any membership change for a party that
has live doors:

1. Recompute each affected member's slot from the new history-sorted ordering.
2. For each owner whose slot changed, `Reslot` the door (new `slot`, `townPortalId`,
   `townX/Y` from §6.3) and emit `SLOT_CHANGED` → channel re-broadcasts the moved town door
   + `partyPortal`.
3. **Joiner:** begins seeing the party's existing doors (channel resends spawns to the
   joiner's session on the next party-state projection / on map presence). Their own door (if
   any) is re-slotted into the party ordering.
4. **Leaver:** immediately stops seeing the party's doors — channel sends `removeDoor` to the
   *leaver only* for every other party door (visibility revocation, not destruction). The
   leaver's own door is re-slotted to **solo slot 0** and re-broadcast.
5. **Disband:** every member reverts to solo slot 0; doors persist (owner-bound), visibility
   collapses to owner-only.

> Visibility revocation for a leaver (sending `removeDoor` to one session without destroying
> the door) is a **channel** concern driven by the `SLOT_CHANGED`/party event; the engine
> only owns slot state. The channel computes the per-session visibility delta from the party
> membership it already reads.

Slot allocation must be **race-safe** under concurrent casts in the same party/town: the
town+party Redis set index (§4.3) is the allocation source of truth; `Spawn` claims a slot by
deriving it from party order and verifying the index, under the registry's serialization.

---

## 8. atlas-channel packet edge

### 8.1 Cast routing (the door Lookup handler)

Register a per-skill handler for `skill.PriestMysticDoorId` in the existing `Lookup`
dispatcher invoked at `skill/handler/common.go:121`. By the time it runs, `UseSkill` has
already consumed MP + Magic Rock (lines 73–95) and skipped the generic buff (no statups).
The handler emits `COMMAND_TOPIC_DOOR / SPAWN{ownerCharacterId, field, casterX, casterY,
skillId, skillLevel}`. Position comes from the caster's character model (`c.X(), c.Y()`),
same source `applyToParty` uses.

> This keeps `character_skill_use.go` untouched and uses the *intended* extension seam.
> Cost/cooldown/animation announce all continue through the standard path; `enableActions`
> at the end of `character_skill_use.go` re-enables the client regardless of door outcome,
> giving the clean rejection state FR-1.2 requires.

**Rejection signalling.** Eligibility rejections that are knowable channel-side (field limit,
town map, no return map) are checked *in the handler before emitting SPAWN* and simply emit
nothing (client already re-enabled). Engine-side rejections (no slot) produce no CREATED
event; the channel doesn't block on a response, so the cast is a silent no-op for the client
(MP/rock already consumed — matching MapleStory, where a failed door still burns the cast).
A future `REJECTED` event could surface a blocked message, but is out of scope (YAGNI).

### 8.2 New inbound handler: enter-door

New `socket/handler/door_enter.go` decoding Cosmic's `DoorHandler` shape: `int ownerId`,
`byte direction` (`1` = town→target, `0` = target→town). Registered with the
**`LoggedInValidator`** (FR-7.3 — a validator-less handler is silently dropped). Flow:

1. REST `GET /worlds/.../maps/{currentMap}/.../doors` (or `GET /doors` by owner) from
   atlas-doors; find the door owned by `ownerId` present on the character's current map.
2. Validate requester is the owner **or** a current same-channel party member (channel reads
   party membership it already has). Reject (blocked message + `enableActions`) otherwise, or
   if mid-map-change/banned (FR-5.4).
3. Warp via the **existing** `portal.NewProcessor(l, ctx).Warp(field, characterId,
   linkedMapId)` to the linked map; resolve the destination portal/position from the door
   record (town side warps to area door position; area side warps to town door position).
   Play the portal sound (`playPortalSound`, FR-5.3).

No new warp command — OQ-2 resolved channel-side.

### 8.3 New consumer: door status → broadcast

New `kafka/consumer/door/consumer.go` (mirrors the monster/summon status consumers:
`SetHeaderParsers(Span, Tenant)`, `LastOffset`). Handlers for `CREATED`/`REMOVED`/
`SLOT_CHANGED` resolve **eligible viewers** = caster ∪ same-channel party members present in
the relevant map, then `session.Announce` the packets:

- **Field:** `spawnDoor` (area door) to eligible viewers in the source field.
- **Town:** `spawnDoor` + `spawnPortal` (minimap portal) to eligible viewers in the town.
- **Party minimap:** `partyPortal` to party members (door map + minimap x/y).
- **REMOVED:** `removeDoor` to both maps + cleared `partyPortal`.

Eligible-viewer resolution intersects `_map` session enumeration
(`ForSessionsInMap`/`ForOtherSessionsInMap`) with party membership (party REST read) — the
channel already does both. Visibility is **party-scoped**, not whole-map, so the broadcast
filters map sessions by party membership (caster always included).

### 8.4 Map-enter spawn (FR-3.4)

A player entering a field or town that contains a door they're eligible to see must receive
its spawn packets. Hook the existing map-enter/load path in atlas-channel: on entering a
field/town, query atlas-doors `GET .../doors` for that map, filter to doors whose owner is
self or a same-channel party member, and `Announce` the spawns to the entering session only.
This reuses the same eligibility filter as §8.3.

### 8.5 partyPortal — the long-standing TODO

`libs/atlas-packet/party/clientbound/created.go` already reserves the door fields (currently
hard-zeroed: door map x/y as int, minimap x/y as short). The `partyPortal` minimap indicator
("Write doors for party" TODO in docs/TODO.md) is wired by populating those fields — and the
dedicated party-operation door-update packet — from live door state when a door is
created/removed/re-slotted. The channel sends it to party members so the door shows on their
minimap. Exact party-operation opcode/bytes per version are part of the §9 verification
matrix.

---

## 9. Version coverage & packet library

New `libs/atlas-packet/door/` with:

- `clientbound/`: `spawnDoor`, `removeDoor`, `spawnPortal`, `playPortalSound` encoders, plus
  the `partyPortal` party-operation door fields in `party/clientbound`.
- `serverbound/`: enter-door decoder (`int ownerId`, `byte direction`).

Each encoder/decoder reads the tenant from context to branch on version where structures
diverge (the established pattern — e.g. `character/clientbound/spawn.go`). **Byte layouts are
NOT ported blindly from v83** (opcode-table shift ≥0x3D, `MajorVersion()` off-by-one for v84,
jms divergence — all in Known Bug Patterns).

**Per-version verification matrix (plan-phase work, OQ-5).** For each of `gms_v83`,
`gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185` × each packet
(`spawnDoor`, `removeDoor`, `spawnPortal`, `playPortalSound`, `partyPortal`, enter-door):

1. Resolve the opcode (handler + writer) from the per-version IDA export / WZ source of truth.
2. Confirm the byte structure against the decompile (not assumed).
3. Add the handler + writer rows to the tenant socket **templates** for that version, each
   handler with a validator (`LoggedInValidator` for enter-door).
4. Patch **live tenant config** for existing tenants (new handler/writer opcodes are not
   auto-applied to already-provisioned tenants; channel must be restarted — projection does
   not hot-reload handlers/writers — Known Bug Patterns).
5. Add per-version byte-level tests in `libs/atlas-packet/door`.

Versions lacking a usable IDB for a specific opcode are a **stop-and-escalate** (unresolved
fname rule), parked like the v92 mount-food handler — not guessed.

---

## 10. Service registration & deployment

atlas-doors adds **no new shared lib** (it reuses atlas-redis/object-id/lock/kafka/etc.), so
the root `Dockerfile` needs **no edit** — it builds via the parameterized
`COPY services/${SERVICE}/ services/${SERVICE}/`. Registration checklist (the
docker-bake/services.json hand-sync gotcha applies):

- [ ] `.github/config/services.json` — add the `atlas-doors` go-service entry (name, path
      `services/atlas-doors`, module_path `services/atlas-doors/atlas.com/doors`, docker_image).
- [ ] `docker-bake.hcl` — add `"atlas-doors"` to the hardcoded `go_services` list (HCL can't
      read JSON — both must be edited).
- [ ] `go.work` — add `./services/atlas-doors/atlas.com/doors`.
- [ ] `deploy/k8s/base/atlas-doors.yaml` — Deployment + Service mirroring atlas-summons
      (env: Redis, Kafka topics, DATA/PARTY service URLs via BASE_SERVICE_URL fallback —
      do **not** hard-code `*_SERVICE_URL` from the kustomize base, per Known Bug Patterns;
      readiness probe path `/api/readyz`, not `/readyz`).
- [ ] Kafka topic env constants for `COMMAND_TOPIC_DOOR` / `EVENT_TOPIC_DOOR_STATUS` in the
      deployment + any compose/local config.
- [ ] Config-status projection adoption (do not load-once-then-Fatalf on a tenant provisioned
      after pod start) — copy the summons header-driven tenant pattern.

---

## 11. Testing strategy

- **Engine unit tests (atlas-doors):** Model/Builder immutability; slot computation
  (solo=0, party indices, 6-member saturation, no-slot rejection); town resolution
  (return vs forced-return); recast replace; expiry sweep; re-slot on join/leave/disband
  (FR-6.4 table). Use the project Builder pattern for fixtures — no `*_testhelpers.go`.
- **Registry tests:** field/owner/town index membership across add/remove; per-party slot
  isolation (two parties, same town, same portal id — both allowed); solo non-collision.
- **Channel handler tests:** door Lookup handler emits SPAWN with caster position; enter-door
  decode + ownership/party validation + warp call (seam-injected like `loadCasterFunc`);
  eligibility rejection leaves client re-enabled.
- **Packet byte tests:** per-version encode/decode golden bytes for all six packets across
  all six versions (the §9 matrix), verified against IDA/WZ — the gate for "version done."
- **Cross-cutting:** rediskeyguard clean (GOWORK=off); `go test -race`, `go vet`,
  `go build`, `docker buildx bake atlas-doors` (+ atlas-channel) per CLAUDE.md.

---

## 12. Risks & open verification items

| Risk | Mitigation / plan-phase action |
|---|---|
| **OQ-3:** towns may not expose ≥6 `PortalTypeDoor` portals in atlas-data. | Verify via live `GET /api/data/maps/{id}/portals` per version; document fallback placement (§6.3). Highest-priority pre-implementation check. |
| **OQ-5:** per-version door opcodes/bytes unknown until IDA-verified. | §9 matrix; escalate unresolved fnames, don't guess. |
| atlas-parties may not return **history-sorted** members. | Confirm during plan; small read addition if needed (§7.1). |
| Client crash on rapid cast→cancel (FR-6.3). | Honor ~3000 ms deploy-effect grace before remove broadcast; expiry/remove paths check `now - deployTime ≥ grace` or defer. |
| Recast race / concurrent same-party casts. | Town+party Redis index as slot source of truth under registry serialization (§7.2). |
| `MajorVersion()`/opcode-table off-by-one for v84/v87. | Use `>=87` not `>83`; treat v84≡v83 for structure but verify opcode rows (Known Bug Patterns). |
| Live tenants don't auto-get new opcodes. | Patch live config + restart channel as an explicit deploy step (§9.4). |

---

## 13. Out of scope (reaffirmed)

Other Priest/Magician skills; GM door tooling; door persistence across restart; cross-channel
visibility; atlas-ui changes; a `REJECTED`-event blocked-message UX; generalized field-object
library. All per PRD §2 non-goals.
