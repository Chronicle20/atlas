# Storage

## Tables

This service uses in-memory storage only. No database tables exist.

## In-Memory Registry

Drops are stored in a thread-safe in-memory registry with the following structures:

| Structure | Type | Description |
|-----------|------|-------------|
| dropMap | map[uint32]Model | Maps drop ID to drop model |
| dropReservations | map[uint32]uint32 | Maps drop ID to reserving character ID |
| dropsInMap | map[mapKey][]uint32 | Maps map key to list of drop IDs in that map |

### Map Key

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| mapId | map.Id | Map identifier |

## Relationships

- Each drop belongs to exactly one map (identified by tenant, world, channel, map)
- A drop may reference an equipment ID stored in atlas-equipables service

## Indexes

In-memory indexes:

- Drop by ID (primary lookup)
- Drops by map key (tenant + world + channel + map)

## Migration Rules

Not applicable. This service uses ephemeral in-memory storage that is cleared on service restart. On shutdown, all drops are expired and corresponding events are emitted.
