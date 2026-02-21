# Storage

This service uses Redis for all persistent storage. There are no relational database tables.

## Data Structures

### chalkboard (TenantRegistry)

Stores chalkboard messages per character, scoped by tenant.

| Key Component | Type | Description |
|---------------|------|-------------|
| tenant | tenant.Model | Tenant scope |
| characterId | uint32 | Character identifier |

| Value | Type | Description |
|-------|------|-------------|
| message | string | Chalkboard message content |

Managed by `atlas-redis.TenantRegistry` with namespace `chalkboard`.

### chalk-char (Redis Sets)

Stores character-to-field membership using Redis sets, scoped by tenant and field.

Key format: `atlas:chalk-char:{tenantKey}:{worldId}:{channelId}:{mapId}:{instance}`

| Set Member | Type | Description |
|------------|------|-------------|
| characterId | uint32 (string-encoded) | Character identifier |

## Relationships

None. Chalkboard messages and character locations are independent data structures joined at query time by the chalkboard REST handler.

## Indexes

No additional indexes. Redis key structure provides direct lookup by tenant+character (chalkboard) and tenant+field (character location).

## Migration Rules

No migrations. Redis data structures are created on first write.
