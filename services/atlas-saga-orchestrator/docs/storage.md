# Storage

## Tables

None. This service uses in-memory storage only.

## In-Memory Cache

### Saga Cache

Sagas are stored in an in-memory cache scoped by tenant ID. The cache is a singleton initialized via `GetCache()` and protected by a read-write mutex for concurrent access.

| Key | Type | Description |
|-----|------|-------------|
| tenantId | uuid.UUID | Tenant identifier (from request headers) |
| transactionId | uuid.UUID | Saga transaction ID |

Cache structure: `map[tenantId]map[transactionId]Saga`

### Cache Interface

| Method | Parameters | Returns | Description |
|--------|-----------|---------|-------------|
| GetAll | tenantId | []Saga | Returns all sagas for a tenant |
| GetById | tenantId, transactionId | (Saga, bool) | Returns a saga by transaction ID |
| Put | tenantId, saga | - | Adds or updates a saga |
| Remove | tenantId, transactionId | bool | Removes a saga, returns true if found |

## Relationships

None.

## Indexes

None.

## Migration Rules

Not applicable. In-memory storage does not persist across service restarts.

## Data Lifecycle

- Sagas are created when a saga command is received via Kafka or a POST request is made via REST
- Sagas are updated on each step status change (completion, failure, result data)
- Sagas are removed when:
  - All steps complete successfully
  - Compensation completes after failure
  - Validation fails with no prior steps to compensate
- A `ResetCache` function exists for testing, which replaces the singleton with a fresh empty cache
