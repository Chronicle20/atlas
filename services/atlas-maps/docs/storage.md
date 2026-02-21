# Storage

## Tables

### character_map_visits

Records the first time a character visits a map.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | Primary key |
| tenant_id | UUID | Not null |
| character_id | uint32 | Not null |
| map_id | uint32 | Not null |
| first_visited_at | timestamp | Not null, default CURRENT_TIMESTAMP |

## Relationships

None.

## Indexes

| Name | Columns | Type |
|------|---------|------|
| idx_visits_tenant_char_map | tenant_id, character_id, map_id | Unique |
| idx_visits_tenant_char | tenant_id, character_id | Non-unique |

## Migration Rules

- Table migration via GORM AutoMigrate on service startup
- Schema changes are additive

## In-Memory Registries

### Character Registry

Singleton registry tracking character presence in maps. State is not persisted and is rebuilt from events on service restart.

| Key | Value |
|-----|-------|
| MapKey | []uint32 (character IDs) |

### Spawn Point Registry (Redis)

Redis-backed registry tracking spawn point cooldowns. Lazily initialized from atlas-data on first access per map. Uses Lua scripts for atomic eligibility checks and cooldown updates.

| Key Pattern | Type | Value |
|-------------|------|-------|
| atlas:maps:spawn:{tenant}:{worldId}:{channelId}:{mapId}:{instance} | Hash | field: spawn point ID, value: JSON-encoded spawn point with NextSpawnAt (Unix ms) |

### Weather Registry

Singleton registry tracking active weather effects per map instance. State is not persisted. Expired entries are removed by the weather task.

| Key | Value |
|-----|-------|
| FieldKey | WeatherEntry |
