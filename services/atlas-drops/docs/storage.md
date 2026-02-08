# Storage

## Tables

This service uses in-memory storage only. No database tables exist.

## In-Memory Registry

Drops are stored in a thread-safe in-memory registry (singleton) with the following structures:

| Structure | Type | Description |
|-----------|------|-------------|
| dropMap | map[uint32]Model | Maps drop ID to drop model |
| dropReservations | map[uint32]uint32 | Maps drop ID to reserving character ID |
| dropLocks | map[uint32]*sync.Mutex | Per-drop locks for concurrent access |
| mapLocks | map[mapKey]*sync.Mutex | Per-map locks for concurrent access |
| dropsInMap | map[mapKey][]uint32 | Maps map key to list of drop IDs in that map |

### Map Key

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| field | field.Model | Field context (world, channel, map, instance) |

## Relationships

- Each drop belongs to exactly one field (identified by tenant + world + channel + map + instance)
- Equipment drops carry all stats inline on the drop model; there is no separate equipment storage
- Drop reservations are tracked separately from the drop model itself, mapping drop ID to the reserving character ID

## Indexes

In-memory indexes:

- Drop by ID (primary lookup via dropMap)
- Drops by map key (tenant + field via dropsInMap)

## Migration Rules

Not applicable. This service uses ephemeral in-memory storage that is cleared on service restart. On shutdown, all drops are expired and corresponding EXPIRED events are emitted before the process exits.
