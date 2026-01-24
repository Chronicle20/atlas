# Storage

## Tables

None. This service uses in-memory storage only.

## In-Memory Registries

### Character Registry

Singleton registry tracking character presence in maps.

| Key | Value |
|-----|-------|
| MapKey | []uint32 (character IDs) |

### Spawn Point Registry

Singleton registry tracking spawn point cooldowns.

| Key | Value |
|-----|-------|
| MapKey | []CooldownSpawnPoint |

## Relationships

None.

## Indexes

None.

## Migration Rules

Not applicable. State is not persisted and is rebuilt from events on service restart.
