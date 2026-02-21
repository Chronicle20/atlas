# Storage

## Tables

This service uses no relational database. All state is stored in Redis.

## Redis Registries

### transport.RouteRegistry (transport/route_registry.go)

Stores scheduled route models per tenant using `atlas.TenantRegistry`.

| Key Pattern | Type | Description |
|-------------|------|-------------|
| transport-route:{tenantKey}:{routeId} | JSON | Route model |

**Value:** `transport.Model` (JSON serialized)

**Backed by:** `atlas.TenantRegistry[uuid.UUID, Model]`

### instance.RouteRegistry (instance/route_registry.go)

Stores instance route models per tenant using `atlas.TenantRegistry`.

| Key Pattern | Type | Description |
|-------------|------|-------------|
| instance-route:{tenantKey}:{routeId} | JSON | Instance route model |

**Value:** `instance.RouteModel` (JSON serialized)

**Backed by:** `atlas.TenantRegistry[uuid.UUID, RouteModel]`

### instance.InstanceRegistry (instance/instance_registry.go)

Stores active transport instances with multiple Redis structures.

| Key Pattern | Type | Description |
|-------------|------|-------------|
| transport:instances | SET | Set of all active instance IDs |
| transport:instance:{instanceId} | STRING | Instance metadata (JSON) |
| transport:instance:{instanceId}:chars | HASH | Character entries keyed by character ID |
| transport:route:{tenantId}:{routeId} | SET | Set of instance IDs for a route |

**Metadata Value:** `TransportInstance` (JSON serialized, excludes characters)

**Character Entry Value:** `CharacterEntry` (JSON serialized)

### instance.CharacterRegistry (instance/character_registry.go)

Maps character IDs to their active transport instance.

| Key Pattern | Type | Description |
|-------------|------|-------------|
| transport:characters | HASH | Character ID to instance ID mapping |

**Field:** Character ID (string)

**Value:** Instance ID (UUID string)

### channel.Registry (channel/registry.go)

Stores active channel models per tenant.

| Key Pattern | Type | Description |
|-------------|------|-------------|
| transport:channels:{tenantKey} | SET | Set of channel members (worldId:channelId) |

**Member Format:** `{worldId}:{channelId}`

## Relationships

Instance transport instances reference routes by route ID and tenants by tenant ID within the Redis structures. The `transport:route:{tenantId}:{routeId}` set provides a secondary index from route to instances.

## Indexes

- `transport:instances`: Global index of all active instance IDs
- `transport:route:{tenantId}:{routeId}`: Per-route index of instance IDs
- `transport:characters`: Global index of character-to-instance mapping

## Migration Rules

Not applicable. All state is ephemeral and reconstructed on startup from the configuration service. Redis keys have no expiration set; instances are explicitly released when completed, cancelled, or during graceful shutdown.
