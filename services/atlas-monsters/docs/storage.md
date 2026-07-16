# Monster Storage

This service uses Redis for all state storage. There is no SQL or relational database.

## Keys

### Monster Instances

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:monster:{tenantId}:{uniqueId}` | String (JSON) | Monster instance data |
| `atlas:monster-map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Set of monster uniqueId values in a field |

The monster JSON structure contains all model fields (including `controllerHasAggro`, `nextEligibleRepickAtMs`, and `lastDamageTakenMs`), damage entries as an array (each carrying a per-character `lastHitMs`), and status effects with timing fields serialized as milliseconds (including the reflect fields `reflectKind`, `reflectPercent`, `reflectLtX/Y`, `reflectRbX/Y`, `reflectMaxDamage`, omitted when zero/empty).

Updates to monster instances use the shared atlas-redis `Registry.Update` optimistic-lock helper (`WATCH`/`GET`/`TxPipelined SET`), retried up to 1000 times on contention. Damage application, HP/MP recovery, and aggro decay are applied as pure Go closures under this same optimistic-lock update (no Lua scripts).

### ID Allocation

Monster IDs are NOT minted by this service. They come from the shared `atlas-object-id` allocator (`libs/atlas-object-id/allocator.go`), which is also used by atlas-reactors and atlas-drops.

Allocator-managed keys (per tenant, NOT per service):

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:oid:{tenantId}:next` | String (counter) | Sequential ID counter; range `1000000` (`MinId`) to `2147483647` (`MaxId`, the v83 wire-format positive int32 ceiling) |
| `atlas:oid:{tenantId}:free` | List | LIFO recycle pool; only consulted once the counter passes `RecycleThreshold = MaxId - 100M` |

A single tenant-scoped namespace is shared across reactors, monsters, and drops because the v83 client keys map objects by oid alone — colliding IDs across entity types crash the client. Per-tenant rather than per-field: each service stores its entities under `<entity>:{tenantId}:{id}` with no field component in the key, so per-field allocation would collide in storage when the same id was minted in two different fields. See the package-level comment in `libs/atlas-object-id/allocator.go` for the full rationale.

Below `RecycleThreshold` the script always INCRs the counter and `Release` is a no-op (the free list stays empty); only once the counter approaches exhaustion does the LIFO pool kick in as a safety valve.

### Skill Cooldowns

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:monster-cooldown:{tenantId}:{monsterId}:{skillId}` | String (TTL) | Skill cooldown marker; expires after cooldown duration |

Cooldown checks use `EXISTS`. Clearing all cooldowns for a monster uses `SCAN` + `DEL` on the pattern `atlas:monster-cooldown:{tenantId}:{monsterId}:*`.

### Basic-Attack Cooldowns

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:monster-attack-cooldown:{tenantId}:{monsterId}:{attackPos}` | String (TTL) | Basic-attack-position cooldown marker; expires after the attack's cooldown duration |

Cooldown checks use `EXISTS`. Clearing all attack cooldowns for a monster uses `SCAN` + `DEL` on the pattern `atlas:monster-attack-cooldown:{tenantId}:{monsterId}:*`.

### Puppets

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:monster-puppet:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}:{ownerCharacterId}` | String (JSON) | Puppet payload: ownerCharacterId, x, y |
| `atlas:monster-puppet-field:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Set of owner character IDs with a registered puppet in the field |

### Drop Timers

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:drop-timer:{tenantId}:{uniqueId}` | String (JSON) | Friendly monster drop timer state |

The drop timer JSON contains monsterId, field, dropPeriod, weaponAttack, maxHp, lastDropAt, and lastHitAt (timing as milliseconds). Updates use the shared atlas-redis `Registry.Update` optimistic-lock helper.

## Relationships

Monster instances are indexed by field via the `atlas:monster-map` Set keys. The Set contains string representations of uniqueId values that correspond to `atlas:monster` keys.

Drop timer entries reference monster uniqueId values that correspond to `atlas:monster` keys.

Puppet entries are indexed by field via the `atlas:monster-puppet-field` Set keys, containing owner character IDs that correspond to `atlas:monster-puppet` keys for the same field.

## Indexes

| Index Key Pattern | Points To |
|-------------------|-----------|
| `atlas:monster-map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | `atlas:monster:{tenantId}:{uniqueId}` |
| `atlas:monster-puppet-field:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | `atlas:monster-puppet:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}:{ownerCharacterId}` |

## Migration Rules

All state is ephemeral. On service teardown, all monsters are destroyed and removed from Redis. State is not preserved across restarts.
