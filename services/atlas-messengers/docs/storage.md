# Storage

## Tables

None. This service uses in-memory storage only.

## In-Memory Registries

### Messenger Registry

Stores messenger state per tenant.

| Field | Type | Description |
|-------|------|-------------|
| tenantMessengerId | map[tenant.Model]uint32 | Next messenger ID per tenant |
| messengerReg | map[tenant.Model]map[uint32]Model | Messengers per tenant |
| tenantLock | map[tenant.Model]*sync.RWMutex | Per-tenant locks |

### Character Registry

Stores character state per tenant.

| Field | Type | Description |
|-------|------|-------------|
| characterReg | map[tenant.Model]map[uint32]Model | Characters per tenant |
| tenantLock | map[tenant.Model]*sync.RWMutex | Per-tenant locks |

## Relationships

None.

## Indexes

None.

## Migration Rules

None. State is not persisted.
