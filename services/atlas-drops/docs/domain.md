# Drop Domain

## drop

### Responsibility

Manages the lifecycle of drops within game maps. A drop represents an item or meso that exists on the ground and can be picked up by characters. Equipment drops carry their stat attributes inline on the drop model itself.

### Core Models

#### Drop Model

Represents a dropped item or meso on a map. Equipment stats are stored directly on the model.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| id | uint32 | Unique drop identifier (auto-generated) |
| transactionId | uuid.UUID | Transaction identifier |
| field | field.Model | Field context (world, channel, map, instance) |
| itemId | uint32 | Item identifier (0 if meso drop) |
| quantity | uint32 | Item quantity |
| meso | uint32 | Meso amount (0 if item drop) |
| dropType | byte | Type of drop |
| x | int16 | X coordinate on the map |
| y | int16 | Y coordinate on the map |
| ownerId | uint32 | Owner character identifier |
| ownerPartyId | uint32 | Owner party identifier |
| dropTime | time.Time | Time when drop was created |
| dropperId | uint32 | Entity that dropped the item |
| dropperX | int16 | Dropper X coordinate |
| dropperY | int16 | Dropper Y coordinate |
| playerDrop | bool | Whether dropped by a player |
| status | string | Drop status (AVAILABLE, RESERVED) |
| petSlot | int8 | Pet slot for pet pickup (-1 if none) |
| strength | uint16 | Strength stat (equipment) |
| dexterity | uint16 | Dexterity stat (equipment) |
| intelligence | uint16 | Intelligence stat (equipment) |
| luck | uint16 | Luck stat (equipment) |
| hp | uint16 | HP stat (equipment) |
| mp | uint16 | MP stat (equipment) |
| weaponAttack | uint16 | Weapon attack stat (equipment) |
| magicAttack | uint16 | Magic attack stat (equipment) |
| weaponDefense | uint16 | Weapon defense stat (equipment) |
| magicDefense | uint16 | Magic defense stat (equipment) |
| accuracy | uint16 | Accuracy stat (equipment) |
| avoidability | uint16 | Avoidability stat (equipment) |
| hands | uint16 | Hands stat (equipment) |
| speed | uint16 | Speed stat (equipment) |
| jump | uint16 | Jump stat (equipment) |
| slots | uint16 | Upgrade slots (equipment) |

#### ModelBuilder

A fluent builder for constructing drop Models. Requires a valid tenant and field. Generates a transactionId and sets default petSlot of -1 on construction. Validation on Build() requires a non-nil tenant ID and a non-nil transactionId.

### Invariants

- Drop IDs are unique and auto-generated starting from 1000000001
- Drop IDs wrap around to 1000000001 after reaching 2000000000
- A drop must have a valid tenant and transactionId
- A drop is either an item drop (itemId > 0) or a meso drop (meso > 0)
- Only the character that reserved a drop can have the reservation cancelled for them; cancellation by a different character is a no-op
- Equipment drops carry all stats inline on the drop model (no separate equipment entity)
- Pet slot defaults to -1 (no pet) and is reset to -1 on reservation cancellation

### State Transitions

```
[Created] -> AVAILABLE
AVAILABLE -> RESERVED (via Reserve)
RESERVED -> AVAILABLE (via CancelReservation)
AVAILABLE -> [Removed] (via Gather or Expire)
RESERVED -> [Removed] (via Gather)
```

### Processors

#### Drop Processor

| Operation | Description |
|-----------|-------------|
| Spawn | Creates a new drop with inline equipment stats from the command |
| SpawnAndEmit | Creates a new drop and emits a CREATED event via Kafka |
| SpawnForCharacter | Creates a new drop originating from a character |
| SpawnForCharacterAndEmit | Creates a new character drop and emits a CREATED event via Kafka |
| Reserve | Reserves a drop for a character with optional pet slot; emits RESERVED on success or RESERVATION_FAILURE on failure |
| ReserveAndEmit | Reserves a drop and emits the event via Kafka |
| CancelReservation | Cancels a drop reservation; emits RESERVATION_FAILURE |
| CancelReservationAndEmit | Cancels a reservation and emits the event via Kafka |
| Gather | Removes a drop when picked up; emits PICKED_UP |
| GatherAndEmit | Gathers a drop and emits the event via Kafka |
| Expire | Removes a drop due to timeout; emits EXPIRED |
| ExpireAndEmit | Expires a drop and emits the event via Kafka |
| GetById | Retrieves a drop by ID from the registry |
| GetForMap | Retrieves all drops for a specific field (tenant + world + channel + map + instance) |
| ByIdProvider | Returns a model.Provider for a drop by ID |
| ForMapProvider | Returns a model.Provider for drops in a field |

#### AllProvider

A package-level provider that returns all drops across all tenants and maps from the registry.

#### Expiration Task

A background task that periodically scans all drops in the registry. Drops in AVAILABLE status that have exceeded the configured expiration duration (default 3 minutes) are expired and corresponding EXPIRED events are emitted. The expiration duration is read from the atlas-configurations service task configuration for `drop_expiration_task`.
