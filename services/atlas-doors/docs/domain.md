# Door Domain

## Responsibility

Manages Mystic Door instances: spawning the paired area/town door, computing the caster's party door slot and town-portal position, reslotting town portals when party membership changes, and removing doors on recast, expiry, logout, channel change, and field departure.

## Core Models

### Model

Represents an active Mystic Door instance. Immutable; constructed via `ModelBuilder` (`NewBuilder`, `Clone`).

| Field | Type | Description |
|-------|------|-------------|
| areaDoorId | uint32 | Object id of the area-side door (also the registry primary key and `PairId`) |
| townDoorId | uint32 | Object id of the town-side door |
| ownerCharacterId | character.Id | Caster who owns the door |
| partyId | uint32 | Owner's party id (0 when solo) |
| skillId | skill.Id | Skill that cast the door |
| skillLevel | byte | Level of the casting skill |
| fld | field.Model | Area field (world, channel, source map, instance) |
| townMapId | _map.Id | Resolved return-town map |
| slot | byte | 0-based party door slot |
| townPortalId | uint32 | Wire town-portal id (0x80 + slot) |
| areaX | point.X | Area-door X coordinate |
| areaY | point.Y | Area-door Y coordinate |
| townX | point.X | Town-door X coordinate |
| townY | point.Y | Town-door Y coordinate |
| deployTime | time.Time | When the door was deployed |
| expiresAt | time.Time | When the door expires (zero when no expiry) |

`PairId()` returns `areaDoorId`. `Reslot(slot, townPortalId, townX, townY)` returns a clone with the slot, town-portal id, and town position replaced (area side unchanged).

### TownPortal

A town map's door-type portal position (atlas-data portal `Type==6`), in load order.

| Field | Type | Description |
|-------|------|-------------|
| X | point.X | Portal X coordinate |
| Y | point.Y | Portal Y coordinate |

### spawnPlan

The resolver's verdict for a single spawn (internal).

| Field | Type | Description |
|-------|------|-------------|
| townMapId | _map.Id | Resolved return-town map |
| slot | byte | Caster's party door slot |
| townPortalId | uint32 | Wire town-portal id (0x80 + slot) |
| townX | point.X | Town-door X coordinate |
| townY | point.Y | Town-door Y coordinate |
| durationMs | int32 | Door lifetime in milliseconds (0 means no expiry) |

## Invariants

- `PairId` equals `areaDoorId`.
- The area object id is allocated before the town object id; on town-allocation failure the area id is released and nothing is persisted or emitted.
- On owner-door persistence failure both allocated object ids are released.
- A spawn first removes any existing door owned by the caster (recast) before deploying the replacement.
- `ComputeSlot` returns the caster's 0-based party index; solo casters (partyId 0) or non-members resolve to slot 0. Party indices at or beyond the maximum party size (6) clamp to slot 5.
- `ResolveTownPortal` encodes the wire portal id as `0x80 + slot`. When the town map exposes more door portals than the slot index, the slot's portal position is used; otherwise the supplied fallback position (default 0,0) is used. It always succeeds.
- `ResolveTownMap` picks `forcedReturnMapId` when it is a real map (not the `EmptyMapId` sentinel `999999999` and not 0); otherwise `returnMapId`.
- `HasValidReturn` is false only when the resolved town map equals the `EmptyMapId` sentinel.
- A door's `expiresAt` is zero when the skill effect duration is `<= 0`; otherwise it is `deployTime + durationMs`.
- The expiry sweep removes a door only when `expiresAt` is non-zero, has passed, and `now - deployTime` is at least the deploy grace window (3 seconds).
- `ReslotParty` reslots the town side only; it never re-sends the area door. Current members reslot to their computed party slot; leavers reslot to solo (slot 0).
- `Reslot` is a no-op when the new slot equals the current slot.

## State Transitions

### Door Lifecycle

1. **Spawn**: Any existing owner door is removed (recast). The caster's party id and a spawn plan (town map, slot, town portal, position, duration) are resolved. Area and town object ids are allocated. The door is persisted and a `CREATED` status event is emitted.
2. **Reslot**: On a party membership change, an affected door's town portal id and town position are recomputed to the member's current party slot and the door is updated; a `SLOT_CHANGED` status event is emitted. Area side is unaffected.
3. **Remove**: The door is deleted from the registry, both object ids are released, and a `REMOVED` status event carrying the removal reason is emitted.

### Spawn Resolution

The resolver fetches the source map and re-checks the cast is permitted: the source map must not be a town, must not have the no-Mystic-Door field limit set, and must have a valid return destination. It resolves the town map and its door-type portals, computes the caster's party slot, resolves the town-portal id/position, and reads the skill effect duration.

### Removal Reasons

| Reason | Trigger |
|--------|---------|
| RECAST | The owner casts a new door (or a `REMOVE` command with no reason) |
| EXPIRY | The expiry sweep removes a door past its lifetime |
| LOGOUT | The owner logs out |
| CHANNEL_CHANGED | The owner changes channel |
| LEFT_FIELD | The owner moves to a field that is neither the door's source field nor its town map |
| PARTY_LEFT | (defined) party-departure removal |

## Processors

### Processor

Interface defining door processing operations. Constructed via `NewProcessor(l, ctx)`, which wires the Kafka emitter, the REST-backed resolver, and the object-id allocator.

**Queries:**
- `GetById`: Retrieves a door by its area door id.
- `GetInField`: Retrieves all doors in a field.
- `GetByOwner`: Retrieves all doors owned by a character.

**Commands:**
- `Spawn`: Removes any existing owner door (recast), resolves the spawn plan, allocates area/town object ids, persists the door, and emits `CREATED`.
- `RemoveByOwner`: Removes all of an owner's doors with the given reason, releasing object ids and emitting `REMOVED` per door.
- `RemoveByOwnerIfLeftField`: Removes an owner's door only when the new field is neither the door's source field nor its town map (walking into the spanned town is a warp, not abandonment).
- `Reslot`: Recomputes a door's town slot, town portal id, and town position; emits `SLOT_CHANGED`. No-op when the slot is unchanged.

### resolver

Internal seam that computes a `spawnPlan` from external data. The production implementation (`restResolver`) reads map metadata and skill effect duration from atlas-data and party membership from atlas-parties.

- `ResolveSpawn`: Produces the spawn plan, performing the town / field-limit / valid-return re-checks.
- `PartyIdFor`: Returns the caster's party id, or 0 when not in a party.

### ReslotParty

Recomputes each affected member's door town slot after a party membership change and reslots the doors via `Processor.Reslot`. Current members reslot to their computed party slot; leavers reslot to solo (slot 0).

### IdAllocator

Wraps the shared field-scoped object-id allocator for doors. `Allocate` surfaces errors so a spawn can fail cleanly and release any already-allocated id; there is no MinId fallback. `Release` returns an object id to the tenant's free pool and discards release errors.

### ExpiryTask

Periodic task (1-second interval) that iterates all doors across all tenants and removes those whose `expiresAt` is non-zero, has passed, and is outside the deploy grace window (3 seconds), with reason `EXPIRY`. When leader election is enabled, only the leader pod runs the sweep.
