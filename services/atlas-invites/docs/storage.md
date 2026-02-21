# Storage

## Tables

This service uses Redis for persistent storage via the `atlas-redis` library.

### Invite Registry

Tenant-scoped key-value store holding serialized invite models, keyed by invite ID.

- Namespace: `invite`
- Key: invite ID (uint32, string-encoded)
- Value: JSON-serialized invite Model

### Active Tenants

Set tracking tenants that have created invites, used by the timeout task to scan across tenants.

- Key: `invite:active-tenants`
- Type: Redis SET
- Members: JSON-serialized tenant models

### ID Generator

Per-tenant monotonic ID generator for invite IDs.

- Namespace: `invite`

## Relationships

N/A

## Indexes

| Index | Namespace | Type | Key | Value |
|-------|-----------|------|-----|-------|
| target-type | invite:target-type | String index | `{targetId}:{inviteType}` | invite ID strings |
| target | invite:target | Uint32 index | targetId | invite IDs |
| originator | invite:originator | Uint32 index | originatorId | invite IDs |

## Migration Rules

N/A
