# Monster Storage

This service uses Redis for all state storage. There is no SQL or relational database.

## Keys

### Monster Instances

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:monster:{tenantId}:{uniqueId}` | String (JSON) | Monster instance data |
| `atlas:monster-map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Set of monster uniqueId values in a field |

The monster JSON structure contains all model fields, damage entries as an array, and status effects with timing fields serialized as milliseconds.

Updates to monster instances use optimistic locking via `WATCH`/`TxPipelined` with up to 10 retries. Damage application uses a Lua script for atomic HP deduction and damage entry append.

### ID Allocation

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:monster-ids:{tenantId}:next` | String (counter) | Sequential ID counter (range 1000000000-2000000000) |
| `atlas:monster-ids:{tenantId}:free` | List | LIFO pool of recycled IDs via LPUSH/LPOP |

Allocation prefers recycled IDs from the free list. Sequential allocation uses a Lua script for atomic check-and-init with wrapping at MaxMonsterId.

### Skill Cooldowns

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:monster-cooldown:{tenantId}:{monsterId}:{skillId}` | String (TTL) | Skill cooldown marker; expires after cooldown duration |

Cooldown checks use `EXISTS`. Clearing all cooldowns for a monster uses `SCAN` + `DEL` on the pattern `atlas:monster-cooldown:{tenantId}:{monsterId}:*`.

### Drop Timers

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:drop-timer:{tenantId}:{uniqueId}` | String (JSON) | Friendly monster drop timer state |

The drop timer JSON contains monsterId, field, dropPeriod, weaponAttack, maxHp, lastDropAt, and lastHitAt (timing as milliseconds). Updates use optimistic locking via `WATCH`/`TxPipelined`.

## Relationships

Monster instances are indexed by field via the `atlas:monster-map` Set keys. The Set contains string representations of uniqueId values that correspond to `atlas:monster` keys.

Drop timer entries reference monster uniqueId values that correspond to `atlas:monster` keys.

## Indexes

| Index Key Pattern | Points To |
|-------------------|-----------|
| `atlas:monster-map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | `atlas:monster:{tenantId}:{uniqueId}` |

## Migration Rules

All state is ephemeral. On service teardown, all monsters are destroyed and removed from Redis. State is not preserved across restarts.
