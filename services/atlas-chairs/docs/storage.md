# Storage

This service uses Redis for persistent storage. No SQL tables exist.

## Tables

### Chair Registry

Redis key-value store via `atlas.TenantRegistry`.

| Property | Value |
|----------|-------|
| Namespace | `chair` |
| Key | Character ID (`uint32` as string) |
| Value | JSON `{"id": uint32, "chairType": string}` |

Scoped per tenant.

### Character Registry

Redis sets for tracking character locations.

| Property | Value |
|----------|-------|
| Namespace | `chair-char` |
| Key Pattern | `atlas:chair-char:{tenantKey}:{worldId}:{channelId}:{mapId}:{instanceId}` |
| Members | Character IDs (`uint32` as strings) |

Scoped per tenant and field (world, channel, map, instance).

## Relationships

No explicit relationships between stored keys. Chair assignments are keyed by character ID. Character locations are keyed by field (world, channel, map, instance).

## Indexes

Not applicable. Redis provides direct key-based access.

## Migration Rules

No schema migrations. Redis keys are created on demand.
