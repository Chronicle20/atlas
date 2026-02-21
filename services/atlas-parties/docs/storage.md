# Storage

This service uses Redis for state storage via `atlas-redis` tenant registries.

## Tables

### Party Registry

Key prefix: `party`

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Party identifier (key) |
| leaderId | uint32 | Character ID of party leader |
| members | []uint32 | List of character IDs in party |

ID generation: Sequential via `atlas.NewIDGenerator` with prefix `party`.

### Character Registry

Key prefix: `party-character`

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Character identifier (key) |
| name | string | Character name |
| level | byte | Character level |
| jobId | job.Id | Character job identifier |
| field | field.Model | Character location |
| partyId | uint32 | Current party identifier |
| online | bool | Online status |
| gm | int | GM level |

## Relationships

- Character-to-party lookup via `Uint32Index` (prefix `party`/`char-party`): maps character ID to party ID

## Indexes

| Name | Type | Key | Value |
|------|------|-----|-------|
| char-party | Uint32Index | characterId | partyId |

## Migration Rules

Not applicable. Redis state is populated from Kafka events and REST lookups at runtime.
