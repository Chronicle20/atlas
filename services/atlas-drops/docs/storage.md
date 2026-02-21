# Storage

## Tables

This service uses Redis for storage. No relational database tables exist.

## Redis Key Structures

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `drops:next_id` | String (integer) | Counter for unique drop ID generation |
| `drop:<id>` | String (JSON) | Serialized drop entry for a single drop |
| `drops:all` | Set | Set of all drop ID strings |
| `drops:map:<tenantId>:<worldId>:<channelId>:<mapId>:<instanceId>` | Set | Set of drop ID strings for a specific map instance |

### Drop Entry Format

Each `drop:<id>` key stores a JSON-serialized `dropEntry`:

| Field | Type | Description |
|-------|------|-------------|
| drop | Model (JSON) | Full drop model including tenant, field, item/meso data, equipment stats |
| reservedBy | uint32 | Character ID that reserved the drop (0 if not reserved) |

## Relationships

- Each drop belongs to exactly one field (identified by tenant + world + channel + map + instance)
- A drop's membership in the global set (`drops:all`) and map set (`drops:map:...`) is maintained alongside the drop entry
- Equipment drops carry all stats inline on the drop model; there is no separate equipment storage
- Drop reservations are tracked within the drop entry itself, pairing the drop model with the reserving character ID

## Indexes

- Drop by ID: direct key lookup via `drop:<id>`
- Drops by map: set membership via `drops:map:<tenantId>:<worldId>:<channelId>:<mapId>:<instanceId>`
- All drops: set membership via `drops:all`

## ID Generation

Drop IDs are generated using a Redis INCR operation on `drops:next_id` with a Lua script that wraps the counter back to 1000000001 when it exceeds 2000000000. The counter is initialized to 999999999 (one below the minimum) on first use via SETNX.

## Migration Rules

Not applicable. Drops are ephemeral and do not require schema migrations. On service shutdown, all drops are expired and corresponding EXPIRED events are emitted before the process exits.
