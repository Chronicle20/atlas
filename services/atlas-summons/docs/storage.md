# Summon Storage

This service uses Redis for all state storage. There is no SQL or relational database.

## Tables

Not applicable — no relational schema. State is represented as Redis keys, described below.

### Summon Instances

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:summon:{tenantId}:{id}` | String (JSON) | Summon instance data |

The JSON value (`storedSummon`) contains every `Model` field plus the tenant
identity (`tenantId`, `tenantRegion`, `tenantMajorVersion`,
`tenantMinorVersion`) and the field components (`worldId`, `channelId`,
`mapId`, `instance`). Time fields (`spawnTime`, `expiresAt`, `nextHealAt`,
`nextBuffAt`) serialize as Unix milliseconds, with 0 representing the zero
time.

### Field Index

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:summon-map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Set of summon id values present in a field |

### Owner Index

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:summon-owner:{tenantId}:{characterId}` | Set | Set of summon id values owned by a character |

### ID Allocation

Summon IDs are NOT minted by this service. They come from the shared
`atlas-object-id` allocator (`libs/atlas-object-id/allocator.go`), which is
also used by atlas-monsters, atlas-reactors, and atlas-drops.

Allocator-managed keys (per tenant, NOT per service):

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:oid:{tenantId}:next` | String (counter) | Sequential ID counter |
| `atlas:oid:{tenantId}:free` | List | LIFO recycle pool, consulted once the counter approaches exhaustion |

A single tenant-scoped namespace is shared across reactors, monsters, drops,
and summons because the v83 client keys map objects by oid alone — colliding
IDs across entity types crash the client.

### Leader-Election Lock

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:lock:summons-sweep` | String (lock) | Leader-election lock gating the expiry and Beholder aura sweep tasks to a single pod |

## Relationships

Summon instances are indexed by field via the `atlas:summon-map` Set keys and
by owner via the `atlas:summon-owner` Set keys. Both Sets contain string
representations of summon id values that correspond to `atlas:summon` keys.

## Indexes

| Index Key Pattern | Points To |
|--------------------|-----------|
| `atlas:summon-map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | `atlas:summon:{tenantId}:{id}` |
| `atlas:summon-owner:{tenantId}:{characterId}` | `atlas:summon:{tenantId}:{id}` |

A stale index entry (pointing to a summon id no longer present under
`atlas:summon`) is skipped on lookup rather than surfaced as an error.

## Migration Rules

All state is ephemeral. Summons are despawned on expiry, on the owner's
logout/channel-change/map-change, or explicitly, and removed from Redis at
that point. State is not preserved across service restarts beyond what
remains in Redis.
