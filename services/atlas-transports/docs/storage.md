# Storage

## Tables

This service uses no persistent storage. All state is held in-memory using thread-safe singleton registries.

## In-Memory Registries

### transport.RouteRegistry (transport/route_registry.go)

Stores scheduled route models keyed by tenant and route ID.

| Key | Type | Description |
|-----|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| routeId | uuid.UUID | Route identifier |

**Value:** `transport.Model`

**Concurrency:** `sync.RWMutex`, singleton via `sync.Once`

### instance.RouteRegistry (instance/route_registry.go)

Stores instance route models keyed by tenant and route ID.

| Key | Type | Description |
|-----|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| routeId | uuid.UUID | Route identifier |

**Value:** `instance.RouteModel`

**Concurrency:** `sync.RWMutex`, singleton via `sync.Once`

### instance.InstanceRegistry (instance/instance_registry.go)

Stores active transport instances. Indexed by instance ID and by route key (tenant + route).

| Index | Key | Type |
|-------|-----|------|
| Primary | instanceId | uuid.UUID |
| Secondary | RouteKey{TenantId, RouteId} | composite |

**Value:** `*instance.TransportInstance`

**Concurrency:** `sync.RWMutex`, singleton via `sync.Once`

### instance.CharacterRegistry (instance/character_registry.go)

Maps character IDs to their active transport instance.

| Key | Type | Description |
|-----|------|-------------|
| characterId | uint32 | Character identifier |

**Value:** `uuid.UUID` (instance ID)

**Concurrency:** `sync.RWMutex`, singleton via `sync.Once`

### channel.Registry (channel/registry.go)

Stores active channel models keyed by tenant ID.

| Key | Type | Description |
|-----|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |

**Value:** `[]channel.Model`

**Concurrency:** `sync.RWMutex`, singleton via `sync.Once`

## Relationships

No persistent relationships. Instance transport instances reference routes by route ID and tenants by tenant ID within the in-memory registries.

## Indexes

Not applicable. In-memory maps provide O(1) lookup by key.

## Migration Rules

Not applicable. All state is ephemeral and reconstructed on startup from the configuration service.
