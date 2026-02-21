# Storage

This service uses Redis for persistent storage. No SQL tables exist.

## Tables

### Expression Registry

Redis key-value store via `atlas.TTLRegistry`.

| Property | Value |
|----------|-------|
| Namespace | `expression` |
| Key | Character ID (`uint32` as string) |
| Value | expression.Model (JSON) |
| Default TTL | 5 seconds |

Scoped per tenant. Entries are automatically expired by the TTLRegistry.

### Tenant Tracking Set

Redis set for tracking tenants with active expression data.

| Property | Value |
|----------|-------|
| Key | `atlas:expression:_tenants` |
| Members | Serialized `tenant.Model` (JSON) |

Used by the RevertTask to scan all tenants for expired expressions.

## Relationships

No explicit relationships between stored keys. Expression entries are keyed by character ID within a tenant scope.

## Indexes

Not applicable. Redis provides direct key-based access.

## Migration Rules

No schema migrations. Redis keys are created on demand and expire automatically via TTL.
