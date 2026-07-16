# Storage

This service uses Redis for persistent storage. No SQL tables exist.

## Tables

### Expression Registry

Redis key-value store via `atlas.TTLRegistry`.

| Property | Value |
|----------|-------|
| Namespace | `expression` |
| Key | `atlas:expression:<tenantId>:<region>:<majorVersion>.<minorVersion>:<characterId>` (characterId as decimal string) |
| Value | expression.Model (JSON) |
| Default TTL | 5 seconds (application-managed; entries carry no native Redis key TTL) |

Scoped per tenant. Expiration is tracked via the Expiry Set below and enforced when entries are popped.

### Expiry Set

Redis sorted set (ZSET) tracking expiration timestamps for Expression Registry entries.

| Property | Value |
|----------|-------|
| Key | `atlas:expression:<tenantId>:<region>:<majorVersion>.<minorVersion>:_expiry` |
| Member | Full entity key of the corresponding Expression Registry entry |
| Score | Expiration time as Unix milliseconds |

One set per tenant. Populated on write, consulted and cleaned up when expired entries are popped.

### Tenant Tracking Set

Redis set for tracking tenants with active expression data.

| Property | Value |
|----------|-------|
| Key | `atlas:expression:_tenants` |
| Members | Serialized `tenant.Model` (JSON) |

Used by the RevertTask to scan all tenants for expired expressions.

## Relationships

No explicit relationships between stored keys. Expression entries are keyed by character ID within a tenant scope. Expiry Set members reference the corresponding Expression Registry entity key.

## Indexes

Not applicable for direct key access. The Expiry Set's sorted-set score provides ordered access to expired entries.

## Migration Rules

No schema migrations. Redis keys are created on demand and removed when popped as expired or explicitly cleared.
