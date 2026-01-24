# Drop Domain

## Responsibility

Manages the lifecycle of drops within game maps. A drop represents an item, equipment, or meso that exists on the ground and can be picked up by characters.

## Core Models

### Drop Model

Represents a dropped item, equipment, or meso on a map.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Unique drop identifier |
| tenant | tenant.Model | Tenant context |
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| mapId | map.Id | Map identifier |
| itemId | uint32 | Item identifier (0 if meso drop) |
| equipmentId | uint32 | Equipment identifier (0 if not equipment) |
| quantity | uint32 | Item quantity |
| meso | uint32 | Meso amount (0 if item drop) |
| dropType | byte | Type of drop |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| ownerId | uint32 | Owner character identifier |
| ownerPartyId | uint32 | Owner party identifier |
| dropTime | time.Time | Time when drop was created |
| dropperId | uint32 | Entity that dropped the item |
| dropperX | int16 | Dropper X coordinate |
| dropperY | int16 | Dropper Y coordinate |
| playerDrop | bool | Whether dropped by a player |
| status | string | Drop status (AVAILABLE, RESERVED) |
| petSlot | int8 | Pet slot for pet pickup (-1 if none) |

### Equipment Model

Represents equipment item attributes fetched from atlas-equipables service.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Equipment identifier |
| itemId | uint32 | Item identifier |
| strength | uint16 | Strength stat |
| dexterity | uint16 | Dexterity stat |
| intelligence | uint16 | Intelligence stat |
| luck | uint16 | Luck stat |
| hp | uint16 | HP stat |
| mp | uint16 | MP stat |
| weaponAttack | uint16 | Weapon attack stat |
| magicAttack | uint16 | Magic attack stat |
| weaponDefense | uint16 | Weapon defense stat |
| magicDefense | uint16 | Magic defense stat |
| accuracy | uint16 | Accuracy stat |
| avoidability | uint16 | Avoidability stat |
| hands | uint16 | Hands stat |
| speed | uint16 | Speed stat |
| jump | uint16 | Jump stat |
| slots | uint16 | Upgrade slots |

## Invariants

- Drop IDs are unique and auto-generated starting from 1000000001
- Drop IDs wrap around after reaching 2000000000
- A drop must have a valid tenant, world, channel, and map
- A drop is either an item drop (itemId > 0) or a meso drop (meso > 0)
- A reserved drop can only be gathered by the character that reserved it
- Equipment drops require a corresponding equipment record in atlas-equipables

## State Transitions

```
[Created] -> AVAILABLE
AVAILABLE -> RESERVED (via Reserve)
RESERVED -> AVAILABLE (via CancelReservation)
AVAILABLE -> [Removed] (via Gather or Expire)
RESERVED -> [Removed] (via Gather or Expire)
```

## Processors

### Drop Processor

| Operation | Description |
|-----------|-------------|
| Spawn | Creates a new drop. For equipment items, creates equipment via atlas-equipables |
| SpawnForCharacter | Creates a new drop from a character without equipment generation |
| Reserve | Reserves a drop for a character with optional pet slot |
| CancelReservation | Cancels a drop reservation |
| Gather | Removes a drop when picked up |
| Expire | Removes a drop due to timeout |
| GetById | Retrieves a drop by ID |
| GetForMap | Retrieves all drops for a specific map |

### Equipment Processor

| Operation | Description |
|-----------|-------------|
| Create | Creates equipment via atlas-equipables REST API |
| Delete | Deletes equipment via atlas-equipables REST API |
| GetById | Retrieves equipment by ID |

### Expiration Task

A background task that periodically checks for expired drops and removes them. Drops expire after a configurable duration (default 3 minutes) if they remain in AVAILABLE status.
