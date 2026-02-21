# Storage

## Tables

None. This service uses Redis for state storage.

## Redis Registries

### Messenger Registry

Tenant-scoped Redis registry storing messenger state.

| Configuration | Value |
|---------------|-------|
| Type | `atlas.TenantRegistry[uint32, Model]` |
| Key prefix | `messenger` |
| Key format | uint64 string of messenger ID |
| Value | JSON-serialized `messenger.Model` |

### Messenger ID Generator

Redis-backed auto-incrementing ID generator for messenger IDs.

| Configuration | Value |
|---------------|-------|
| Type | `atlas.IDGenerator` |
| Key prefix | `messenger` |

### Messenger Create Lock

Redis-backed distributed lock for messenger creation operations.

| Configuration | Value |
|---------------|-------|
| Type | `atlas.Lock` |
| Key prefix | `messenger-create` |
| TTL | 10 seconds |
| Lock key format | `{tenantKey}:{characterId}` |

### Character Registry

Tenant-scoped Redis registry storing character state for messenger membership tracking.

| Configuration | Value |
|---------------|-------|
| Type | `atlas.TenantRegistry[uint32, Model]` |
| Key prefix | `messenger-character` |
| Key format | uint64 string of character ID |
| Value | JSON-serialized `character.Model` |

## Relationships

None.

## Indexes

None.

## Migration Rules

None. State is ephemeral and does not require migrations.
