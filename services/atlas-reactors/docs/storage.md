# Reactor Storage

## Tables

This service uses Redis for volatile storage. No relational database is used.

### Key Patterns

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `reactors:next_id` | String (uint32) | Atomic ID counter for reactor instance IDs |
| `reactors:all` | Set | Global set of all reactor instance IDs |
| `reactor:{id}` | String (JSON) | JSON-serialized reactor Model |
| `reactors:map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Set of reactor IDs for a specific tenant and field |
| `reactor:cd:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}:{classification}:{x}:{y}` | String | Cooldown marker with TTL based on reactor delay |

### reactor:{id} Value Structure

JSON-serialized `Model` containing:

| Field          | JSON Type | Description                    |
|----------------|-----------|--------------------------------|
| tenant         | object    | Tenant model                   |
| id             | number    | Reactor instance ID            |
| worldId        | number    | World identifier               |
| channelId      | number    | Channel identifier             |
| mapId          | number    | Map identifier                 |
| instance       | string    | UUID instance identifier       |
| classification | number    | Reactor classification ID      |
| name           | string    | Reactor name                   |
| data           | object    | Reactor game data              |
| state          | number    | Current reactor state          |
| eventState     | number    | Event state                    |
| delay          | number    | Respawn delay in milliseconds  |
| direction      | number    | Facing direction               |
| x              | number    | X coordinate                   |
| y              | number    | Y coordinate                   |
| updateTime     | string    | Last update timestamp          |

## Relationships

- `reactors:all` contains the IDs of all `reactor:{id}` entries
- `reactors:map:{...}` contains the IDs of reactors within a specific tenant/field combination
- Each reactor ID appears in both `reactors:all` and one `reactors:map:{...}` set

## Indexes

- `reactors:all` serves as a global index for iterating all reactors (used during teardown)
- `reactors:map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` serves as a secondary index for field-scoped queries

## Migration Rules

N/A. Redis keys are volatile and do not persist across service restarts.

### ID Generation

Reactor IDs are generated atomically via a Lua script that increments `reactors:next_id`. IDs range from 1000000001 to 2000000000 and wrap around when the maximum is exceeded.

### Cooldown Expiration

Cooldown keys use Redis TTL set to the reactor's delay value in milliseconds. Expiration is handled automatically by Redis.
