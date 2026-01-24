# Storage

## Tables

None. This service uses in-memory storage only.

## In-Memory Cache

### Saga Cache

Sagas are stored in an in-memory cache scoped by tenant ID.

| Key | Type | Description |
|-----|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| transactionId | uuid.UUID | Saga transaction ID |

Cache structure: `map[tenantId]map[transactionId]Saga`

## Relationships

None.

## Indexes

None.

## Migration Rules

Not applicable. In-memory storage does not persist across service restarts.

## Data Lifecycle

- Sagas are created when a saga command is received or a POST request is made
- Sagas are updated on each step status change
- Sagas are removed when:
  - All steps complete successfully
  - Compensation completes after failure
  - Validation fails with no prior steps to compensate
