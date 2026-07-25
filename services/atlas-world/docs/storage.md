# Storage

This service uses Redis for all persistent state. There are no database tables.

## Redis Registries

### Channel Registry

Stores active channel server registrations per tenant using `atlas-redis.TenantRegistry`.

| Key Prefix | Value Type | Description |
|------------|-----------|-------------|
| channel | channel.Model | Active channel server entries |

Composite key format: `{worldId}:{channelId}`

Tenant tracking set: `channel:tenants`

### Rate Registry

Stores per-world rate multipliers per tenant using `atlas-redis.TenantRegistry`.

| Key Prefix | Value Type | Description |
|------------|-----------|-------------|
| rate | rate.Model | Per-world rate multipliers |

Key format: `{worldId}`

### Broadcast Queue Registry

Stores per (world, family) broadcast queue state per tenant using `atlas-redis.TenantRegistry`, mutated under compare-and-swap (WATCH/MULTI/EXEC).

| Key Prefix | Value Type | Description |
|------------|-----------|-------------|
| world-broadcast | broadcast.QueueModel | Active and pending Maple TV / avatar-megaphone broadcast entries |

Composite key format: `{worldId}:{family}`

Tenant tracking set: `world-broadcast:tenants`

## Relationships

- Channel entries are keyed by world and channel ID within a tenant namespace
- Rate entries are keyed by world ID within a tenant namespace
- Broadcast queue entries are keyed by world ID and family within a tenant namespace
- Tenant set tracks all tenants that have registered channels
- Broadcast tenant set tracks all tenants that have created a broadcast queue

## Indexes

No additional indexes beyond the key-based lookups provided by Redis.

## Migration Rules

No schema migrations. Data is ephemeral and rebuilt from channel status events and configuration on startup.
